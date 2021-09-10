/*
Licensed Materials - Property of IBM
cfc
@ Copyright IBM Corp. 2018 All Rights Reserved
US Government Users Restricted Rights - Use, duplication or disclosure
restricted by GSA ADP Schedule Contract with IBM Corp.
*/

// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/golang/glog"

	apiv1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/flowcontrol"

	"github.com/open-cluster-management/management-ingress/pkg/file"
	"github.com/open-cluster-management/management-ingress/pkg/ingress"
	"github.com/open-cluster-management/management-ingress/pkg/ingress/annotations"
	"github.com/open-cluster-management/management-ingress/pkg/ingress/annotations/class"
	ngx_config "github.com/open-cluster-management/management-ingress/pkg/ingress/controller/config"
	"github.com/open-cluster-management/management-ingress/pkg/ingress/controller/process"
	ngx_template "github.com/open-cluster-management/management-ingress/pkg/ingress/controller/template"
	"github.com/open-cluster-management/management-ingress/pkg/ingress/status"
	"github.com/open-cluster-management/management-ingress/pkg/ingress/store"
	ing_net "github.com/open-cluster-management/management-ingress/pkg/net"
	"github.com/open-cluster-management/management-ingress/pkg/net/dns"
	"github.com/open-cluster-management/management-ingress/pkg/task"
	"github.com/open-cluster-management/management-ingress/pkg/watch"
)

var (
	tmplPath    = "/opt/ibm/router/nginx/template/nginx.tmpl"
	cfgPath     = "/opt/ibm/router/nginx/conf/nginx.conf"
	nginxBinary = "/opt/ibm/router/nginx/sbin/nginx"
)

// NewNGINXController creates a new NGINX Ingress controller.
// If the environment variable NGINX_BINARY exists it will be used
// as source for nginx commands
func NewNGINXController(config *Configuration, fs file.Filesystem) *NGINXController {
	ngx := os.Getenv("NGINX_BINARY")

	if ngx == "" {
		ngx = nginxBinary
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{
		Interface: config.Client.CoreV1().Events(config.Namespace),
	})

	h, err := dns.GetSystemNameServers()
	if err != nil {
		glog.Warningf("unexpected error reading system nameservers: %v", err)
	}

	n := &NGINXController{
		binary: ngx,

		configmap: &apiv1.ConfigMap{},

		isIPV6Enabled: ing_net.IsIPv6Enabled(),

		resolver:        h,
		cfg:             config,
		sslCertTracker:  store.NewSSLCertTracker(),
		syncRateLimiter: flowcontrol.NewTokenBucketRateLimiter(config.SyncRateLimit, 1),

		recorder: eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{
			Component: "nginx-ingress-controller",
		}),

		stopCh:   make(chan struct{}),
		stopLock: &sync.Mutex{},

		fileSystem: fs,

		// create an empty configuration.
		runningConfig: &ingress.Configuration{},
	}

	n.listers, n.controllers = n.createListers(n.stopCh)

	n.syncQueue = task.NewTaskQueue(n.syncIngress)

	n.annotations = annotations.NewAnnotationExtractor(n)

	if config.UpdateStatus {
		n.syncStatus = status.NewStatusSyncer(status.Config{
			Client:              config.Client,
			IngressLister:       n.listers.Ingress,
			ElectionID:          config.ElectionID,
			IngressClass:        class.IngressClass,
			DefaultIngressClass: class.DefaultClass,
		})
	} else {
		glog.Warning("Update of ingress status is disabled (flag --update-status=false was specified)")
	}

	var onChange func()
	onChange = func() {
		template, err := ngx_template.NewTemplate(tmplPath, fs)
		if err != nil {
			// this error is different from the rest because it must be clear why nginx is not working
			glog.Errorf(`
-------------------------------------------------------------------------------
Error loading new template : %v
-------------------------------------------------------------------------------
`, err)
			return
		}

		n.t = template
		glog.Info("new NGINX template loaded")
		n.SetForceReload(true)
	}

	ngxTpl, err := ngx_template.NewTemplate(tmplPath, fs)
	if err != nil {
		glog.Fatalf("invalid NGINX template: %v", err)
	}

	n.t = ngxTpl

	// TODO: refactor
	if _, ok := fs.(*file.DefaultFs); !ok {
		watch.NewDummyFileWatcher(tmplPath, onChange)
	} else {
		_, err = watch.NewFileWatcher(tmplPath, onChange)
		if err != nil {
			glog.Fatalf("unexpected error watching template %v: %v", tmplPath, err)
		}
	}

	return n
}

// NGINXController ...
type NGINXController struct {
	cfg *Configuration

	listers     *ingress.StoreLister
	controllers *cacheController

	annotations annotations.Extractor

	recorder record.EventRecorder

	syncQueue *task.Queue

	syncStatus status.Sync

	// local store of SSL certificates
	// (only certificates used in ingress)
	sslCertTracker *store.SSLCertTracker

	syncRateLimiter flowcontrol.RateLimiter

	// stopLock is used to enforce only a single call to Stop is active.
	// Needed because we allow stopping through an http endpoint and
	// allowing concurrent stoppers leads to stack traces.
	stopLock *sync.Mutex

	stopCh chan struct{}

	// ngxErrCh channel used to detect errors with the nginx processes
	ngxErrCh chan error

	// runningConfig contains the running configuration in the Backend
	runningConfig *ingress.Configuration

	forceReload int32

	t *ngx_template.Template

	configmap *apiv1.ConfigMap

	binary   string
	resolver []net.IP

	// returns true if IPV6 is enabled in the pod
	isIPV6Enabled bool

	isShuttingDown bool

	fileSystem file.Filesystem
}

// Start start a new NGINX master process running in foreground.
func (n *NGINXController) Start() {
	glog.Infof("starting Ingress controller")

	n.controllers.Run(n.stopCh)

	// initial sync of secrets to avoid unnecessary reloads
	glog.Info("running initial sync of secrets")
	for _, obj := range n.listers.Ingress.List() {
		ing := obj.(*networking.Ingress)

		if !class.IsValid(ing) {
			a := ing.GetAnnotations()[class.IngressKey]
			glog.Infof("ignoring add for ingress %v based on annotation %v with value %v", ing.Name, class.IngressKey, a)
			continue
		}

		n.readSecrets(ing)
	}

	if n.syncStatus != nil {
		go n.syncStatus.Run()
	}

	go wait.Until(n.checkMissingSecrets, 30*time.Second, n.stopCh)

	done := make(chan error, 1)
	// #nosec
	cmd := exec.Command(n.binary, "-c", cfgPath)

	// put nginx in another process group to prevent it
	// to receive signals meant for the controller
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	glog.Info("starting NGINX process...")
	n.start(cmd)

	go n.syncQueue.Run(time.Second, n.stopCh)
	// force initial sync
	n.syncQueue.Enqueue(&networking.Ingress{})

	for {
		select {
		case err := <-done:
			if n.isShuttingDown {
				break
			}

			// if the nginx master process dies the workers continue to process requests,
			// passing checks but in case of updates in ingress no updates will be
			// reflected in the nginx configuration which can lead to confusion and report
			// issues because of this behavior.
			// To avoid this issue we restart nginx in case of errors.
			if process.IsRespawnIfRequired(err) {
				process.WaitUntilPortIsAvailable(n.cfg.ListenPorts.HTTP)
				// release command resources
				if err := cmd.Process.Release(); err != nil {
					glog.Warningf("unexpected error release command resources: %v", err)
				}
				// #nosec
				cmd = exec.Command(n.binary, "-c", cfgPath)
				// start a new nginx master process if the controller is not being stopped
				n.start(cmd)
			}
		case <-n.stopCh:
			break
		}
	}
}

// Stop gracefully stops the NGINX master process.
func (n *NGINXController) Stop() error {
	n.isShuttingDown = true

	n.stopLock.Lock()
	defer n.stopLock.Unlock()

	// Only try draining the workqueue if we haven't already.
	if n.syncQueue.IsShuttingDown() {
		return fmt.Errorf("shutdown already in progress")
	}

	glog.Infof("shutting down controller queues")
	close(n.stopCh)
	go n.syncQueue.Shutdown()
	if n.syncStatus != nil {
		n.syncStatus.Shutdown()
	}

	// Send stop signal to Nginx
	glog.Info("stopping NGINX process...")
	// #nosec
	cmd := exec.Command(n.binary, "-c", cfgPath, "-s", "quit")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	// Wait for the Nginx process disappear
	timer := time.NewTicker(time.Second * 1)
	for range timer.C {
		if !process.IsNginxRunning() {
			glog.Info("NGINX process has stopped")
			timer.Stop()
			break
		}
	}

	return nil
}

func (n *NGINXController) start(cmd *exec.Cmd) {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		glog.Fatalf("nginx error: %v", err)
		n.ngxErrCh <- err
		return
	}

	go func() {
		n.ngxErrCh <- cmd.Wait()
	}()
}

// SetConfig sets the configured configmap
func (n *NGINXController) SetConfig(cmap *apiv1.ConfigMap) {
	n.configmap = cmap

	m := map[string]string{}
	if cmap != nil {
		m = cmap.Data
	}

	c := ngx_template.ReadConfig(m)
	if c.SSLSessionTicketKey != "" {
		d, err := base64.StdEncoding.DecodeString(c.SSLSessionTicketKey)
		if err != nil {
			glog.Warningf("unexpected error decoding key ssl-session-ticket-key: %v", err)
			c.SSLSessionTicketKey = ""
		}

		if err := ioutil.WriteFile("/etc/nginx/tickets.key", d, 0600); err != nil {
			glog.Warningf("unexpected error writing /etc/nginx/tickets.key: %v", err)
		}
	}
}

// OnUpdate is called periodically by syncQueue to keep the configuration in sync.
//
// 1. converts configmap configuration to custom configuration object
// 2. write the custom template (the complexity depends on the implementation)
// 3. write the configuration file
//
// returning nill implies the backend will be reloaded.
// if an error is returned means requeue the update
func (n *NGINXController) OnUpdate(ingressCfg ingress.Configuration) error {
	cfg := ngx_template.ReadConfig(n.configmap.Data)
	cfg.Resolver = n.resolver

	// the limit of open files is per worker process
	// and we leave some room to avoid consuming all the FDs available
	wp, err := strconv.Atoi(cfg.WorkerProcesses)
	glog.V(3).Infof("number of worker processes: %v", wp)
	if err != nil {
		wp = 1
	}
	maxOpenFiles := (rlimitMaxNumFiles() / wp) - 1024
	glog.V(3).Infof("maximum number of open file descriptors : %v", rlimitMaxNumFiles())
	if maxOpenFiles < 1024 {
		// this means the value of RLIMIT_NOFILE is too low.
		maxOpenFiles = 1024
	}

	tc := ngx_config.TemplateConfig{
		MaxOpenFiles:  maxOpenFiles,
		BacklogSize:   sysctlSomaxconn(),
		Backends:      ingressCfg.Backends,
		Servers:       ingressCfg.Servers,
		Cfg:           cfg,
		IsIPV6Enabled: n.isIPV6Enabled && !cfg.DisableIpv6,
		ListenPorts:   n.cfg.ListenPorts,
	}

	content, err := n.t.Write(tc)

	if err != nil {
		return err
	}

	err = n.testTemplate(content)
	if err != nil {
		return err
	}

	if glog.V(2) {
		src, _ := ioutil.ReadFile(cfgPath)
		if !bytes.Equal(src, content) {
			tmpfile, err := ioutil.TempFile("", "new-nginx-cfg")
			if err != nil {
				return err
			}
			// #nosec
			defer tmpfile.Close()
			err = ioutil.WriteFile(tmpfile.Name(), content, 0600)
			if err != nil {
				return err
			}

			// executing diff can return exit code != 0
			// #nosec
			diffOutput, _ := exec.Command("diff", "-u", cfgPath, tmpfile.Name()).CombinedOutput()

			glog.Infof("NGINX configuration diff\n")
			glog.Infof("%v\n", string(diffOutput))

			// Do not use defer to remove the temporal file.
			// This is helpful when there is an error in the
			// temporal configuration (we can manually inspect the file).
			// Only remove the file when no error occurred.
			if err := os.Remove(tmpfile.Name()); err != nil {
				return err
			}
		}
	}

	err = ioutil.WriteFile(cfgPath, content, 0600)
	if err != nil {
		return err
	}
	// #nosec
	o, err := exec.Command(n.binary, "-s", "reload", "-c", cfgPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v\n%v", err, string(o))
	}

	return nil
}

// testTemplate checks if the NGINX configuration inside the byte array is valid
// running the command "nginx -t" using a temporal file.
func (n NGINXController) testTemplate(cfg []byte) error {
	if len(cfg) == 0 {
		return fmt.Errorf("invalid nginx configuration (empty)")
	}
	tmpfile, err := ioutil.TempFile("", "nginx-cfg")
	if err != nil {
		return err
	}
	// #nosec
	defer tmpfile.Close()
	err = ioutil.WriteFile(tmpfile.Name(), cfg, 0600)
	if err != nil {
		return err
	}
	// #nosec
	out, err := exec.Command(n.binary, "-t", "-c", tmpfile.Name()).CombinedOutput()
	if err != nil {
		// this error is different from the rest because it must be clear why nginx is not working
		oe := fmt.Sprintf(`
-------------------------------------------------------------------------------
Error: %v
%v
-------------------------------------------------------------------------------
`, err, string(out))
		return errors.New(oe)
	}

	if err := os.Remove(tmpfile.Name()); err != nil {
		return err
	}

	return nil
}

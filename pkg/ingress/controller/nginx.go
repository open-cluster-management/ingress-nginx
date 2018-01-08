package controller

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/golang/glog"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/kubectl/scheme"
	"k8s.io/kubernetes/pkg/util/filesystem"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/file"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/annotations"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/annotations/class"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/controller/process"
	ngx_template "github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/controller/template"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/status"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/store"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/net"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/net/dns"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/task"
)

var (
	tmplPath    = "/opt/ibm/router/nginx/template/nginx.tmpl"
	cfgPath     = "/opt/ibm/router/nginx/conf/nginx.conf"
	nginxBinary = "/usr/sbin/nginx"
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
		backendDefaults: ngx_config.NewDefault().Backend,
		binary:          ngx,

		configmap: &apiv1.ConfigMap{},

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

	// if config.UpdateStatus {
	// 	n.syncStatus = status.NewStatusSyncer(status.Config{
	// 		Client:                 config.Client,
	// 		PublishService:         config.PublishService,
	// 		IngressLister:          n.listers.Ingress,
	// 		ElectionID:             config.ElectionID,
	// 		IngressClass:           class.IngressClass,
	// 		DefaultIngressClass:    class.DefaultClass,
	// 		UpdateStatusOnShutdown: config.UpdateStatusOnShutdown,
	// 		UseNodeInternalIP:      config.UseNodeInternalIP,
	// 	})
	// } else {
	// 	glog.Warning("Update of ingress status is disabled (flag --update-status=false was specified)")
	// }

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
	if _, ok := fs.(filesystem.DefaultFs); !ok {
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

	isShuttingDown bool

	fileSystem filesystem.Filesystem
}

// Start start a new NGINX master process running in foreground.
func (n *NGINXController) Start() {
	glog.Infof("starting Ingress controller")

	n.controllers.Run(n.stopCh)

	// initial sync of secrets to avoid unnecessary reloads
	glog.Info("running initial sync of secrets")
	for _, obj := range n.listers.Ingress.List() {
		ing := obj.(*extensions.Ingress)

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
	n.syncQueue.Enqueue(&extensions.Ingress{})

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
				cmd.Process.Release()
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

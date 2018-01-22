/*
Licensed Materials - Property of IBM
cfc
@ Copyright IBM Corp. 2018 All Rights Reserved
US Government Users Restricted Rights - Use, duplication or disclosure
restricted by GSA ADP Schedule Contract with IBM Corp.
*/

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/file"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/controller"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/version"
)

func main() {
	fmt.Println(version.String())

	showVersion, conf, err := parseFlags()
	if showVersion {
		os.Exit(0)
	}

	if err != nil {
		glog.Fatal(err)
	}

	fs, err := file.NewLocalFS()
	if err != nil {
		glog.Fatal(err)
	}

	kubeClient, err := createApiserverClient(conf.APIServerHost, conf.KubeConfigFile)
	if err != nil {
		handleFatalInitError(err)
	}

	if conf.Namespace != "" {
		_, err = kubeClient.CoreV1().Namespaces().Get(conf.Namespace, metav1.GetOptions{})
		if err != nil {
			glog.Fatalf("no namespace with name %v found: %v", conf.Namespace, err)
		}
	}

	if conf.ResyncPeriod.Seconds() < 10 {
		glog.Fatalf("resync period (%vs) is too low", conf.ResyncPeriod.Seconds())
	}

	conf.Client = kubeClient

	ngx := controller.NewNGINXController(conf, fs)

	go handleSigterm(ngx, func(code int) {
		os.Exit(code)
	})

	ngx.Start()
}

type exiter func(code int)

func handleSigterm(ngx *controller.NGINXController, exit exiter) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	<-signalChan
	glog.Infof("Received SIGTERM, shutting down")

	exitCode := 0
	if err := ngx.Stop(); err != nil {
		glog.Infof("Error during shutdown %v", err)
		exitCode = 1
	}

	glog.Infof("Handled quit, awaiting pod deletion")
	time.Sleep(10 * time.Second)

	glog.Infof("Exiting with %v", exitCode)
	exit(exitCode)
}

// createApiserverClient creates new Kubernetes Apiserver client. When kubeconfig or apiserverHost param is empty
// the function assumes that it is running inside a Kubernetes cluster and attempts to
// discover the Apiserver. Otherwise, it connects to the Apiserver specified.
//
// apiserverHost param is in the format of protocol://address:port/pathPrefix, e.g.http://localhost:8001.
// kubeConfig location of kubeconfig file
func createApiserverClient(apiserverHost string, kubeConfig string) (*kubernetes.Clientset, error) {
	cfg, err := buildConfigFromFlags(apiserverHost, kubeConfig)
	if err != nil {
		return nil, err
	}

	cfg.QPS = defaultQPS
	cfg.Burst = defaultBurst
	cfg.ContentType = "application/vnd.kubernetes.protobuf"

	glog.Infof("Creating API client for %s", cfg.Host)

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	v, err := client.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	glog.Infof("Running in Kubernetes Cluster version v%v.%v (%v) - git (%v) commit %v - platform %v",
		v.Major, v.Minor, v.GitVersion, v.GitTreeState, v.GitCommit, v.Platform)

	return client, nil
}

const (
	// High enough QPS to fit all expected use cases. QPS=0 is not set here, because
	// client code is overriding it.
	defaultQPS = 1e6
	// High enough Burst to fit all expected use cases. Burst=0 is not set here, because
	// client code is overriding it.
	defaultBurst = 1e6

	fakeCertificate = "default-fake-certificate"
)

// buildConfigFromFlags builds REST config based on master URL and kubeconfig path.
// If both of them are empty then in cluster config is used.
func buildConfigFromFlags(masterURL, kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath == "" && masterURL == "" {
		kubeconfig, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}

		return kubeconfig, nil
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			ClusterInfo: clientcmdapi.Cluster{
				Server: masterURL,
			},
		}).ClientConfig()
}

/**
 * Handles fatal init error that prevents server from doing any work. Prints verbose error
 * message and quits the server.
 */
func handleFatalInitError(err error) {
	glog.Fatalf("Error while initializing connection to Kubernetes apiserver. "+
		"This most likely means that the cluster is misconfigured (e.g., it has "+
		"invalid apiserver certificates or service accounts configuration). Reason: %s\n"+
		"Refer to the troubleshooting guide for more information: "+
		"https://github.com/kubernetes/ingress-nginx/blob/master/docs/troubleshooting.md", err)
}

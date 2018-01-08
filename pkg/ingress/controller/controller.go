package controller

import (
	"time"

	clientset "k8s.io/client-go/kubernetes"

	ngx_config "github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/controller/config"
)

// Configuration contains all the settings required by an Ingress controller
type Configuration struct {
	APIServerHost  string
	KubeConfigFile string
	Client         clientset.Interface

	ResyncPeriod time.Duration

	Namespace string

	DefaultSSLCertificate string

	UpdateStatus bool
	ElectionID   string

	ListenPorts *ngx_config.ListenPorts

	SyncRateLimit float32
}

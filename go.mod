module github.com/open-cluster-management/management-ingress

go 1.15

require (
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/aledbf/process-exporter v0.0.0-20170909183352-5917bc766b95 // indirect
	github.com/coreos/etcd v3.3.22+incompatible // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20200426045556-49ad98f6dac1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/mock v1.4.3 // indirect
	github.com/howeyc/gopass v0.0.0-20190910152052-7cb4b85ec19c // indirect
	github.com/imdario/mergo v0.3.9
	github.com/juju/ratelimit v1.0.1 // indirect
	github.com/kylelemons/godebug v1.1.0
	github.com/mitchellh/go-ps v1.0.0
	github.com/mitchellh/mapstructure v1.3.2
	github.com/moul/http2curl v1.0.0 // indirect
	github.com/ncabatoff/process-exporter v0.7.1
	github.com/parnurzeal/gorequest v0.2.16 // indirect
	github.com/paultag/sniff v0.0.0-20200207005214-cf7e4d167732 // indirect
	github.com/petar/GoLLRB v0.0.0-20190514000832-33fb24c13b99 // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	github.com/zakjan/cert-chain-resolver v0.0.0-20200409100953-fa92b0b5236f
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/go-playground/pool.v3 v3.1.1
	k8s.io/api v0.18.5
	k8s.io/apimachinery v0.18.5
	k8s.io/client-go v0.18.5
	k8s.io/ingress-nginx v0.0.0-20200630043014-0e19740ee2e4
	k8s.io/klog v1.0.0
	k8s.io/kubernetes v1.18.8
)

replace (
	k8s.io/api => k8s.io/api v0.18.5
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.5
	k8s.io/apiserver => k8s.io/apiserver v0.18.5
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.5
	k8s.io/client-go => k8s.io/client-go v0.18.5
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.18.5
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.5
	k8s.io/code-generator => k8s.io/code-generator v0.18.5
	k8s.io/component-base => k8s.io/component-base v0.18.5
	k8s.io/cri-api => k8s.io/cri-api v0.18.5
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.18.5
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.5
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.5
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.5
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.5
	k8s.io/kubectl => k8s.io/kubectl v0.18.5
	k8s.io/kubelet => k8s.io/kubelet v0.18.5
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.5
	k8s.io/metrics => k8s.io/metrics v0.18.5
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.5
	github.com/gorilla/websocket => github.com/gorilla/websocket v1.4.2
	github.com/coreos/etcd => go.etcd.io/etcd v3.3.22+incompatible
)

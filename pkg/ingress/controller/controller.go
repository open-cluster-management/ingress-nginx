/*
Licensed Materials - Property of IBM
cfc
@ Copyright IBM Corp. 2018 All Rights Reserved
US Government Users Restricted Rights - Use, duplication or disclosure
restricted by GSA ADP Schedule Contract with IBM Corp.
*/

package controller

import (
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/golang/glog"

	apiv1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	clientset "k8s.io/client-go/kubernetes"

	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/annotations"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/annotations/class"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/annotations/parser"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/annotations/proxy"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/annotations/rewrite"
	ngx_config "github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/controller/config"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/resolver"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/task"
)

const (
	defServerName          = "_"
	rootLocation           = "/"
	kubernetesLocation     = "/kubernetes/"
	kubernetesSvc          = "default/kubernetes"
	kubernetesUpstreamName = "upstream-kubernetes"
)

// Configuration contains all the settings required by an Ingress controller
type Configuration struct {
	APIServerHost  string
	KubeConfigFile string
	Client         clientset.Interface

	ResyncPeriod time.Duration
	ConfigMapName  string

	Namespace string

	DefaultSSLCertificate string

	UpdateStatus bool
	ElectionID   string

	ListenPorts *ngx_config.ListenPorts

	SyncRateLimit float32
}

// SetForceReload sets if the ingress controller should be reloaded or not
func (n *NGINXController) SetForceReload(shouldReload bool) {
	if shouldReload {
		atomic.StoreInt32(&n.forceReload, 1)
		n.syncQueue.Enqueue(&extensions.Ingress{})
	} else {
		atomic.StoreInt32(&n.forceReload, 0)
	}
}

// sync collects all the pieces required to assemble the configuration file and
// then sends the content to the backend (OnUpdate) receiving the populated
// template as response reloading the backend if is required.
func (n *NGINXController) syncIngress(item interface{}) error {
	n.syncRateLimiter.Accept()

	if n.syncQueue.IsShuttingDown() {
		return nil
	}

	if element, ok := item.(task.Element); ok {
		if name, ok := element.Key.(string); ok {
			if obj, exists, _ := n.listers.Ingress.GetByKey(name); exists {
				ing := obj.(*extensions.Ingress)
				n.readSecrets(ing)
			}
		}
	}

	// Sort ingress rules using the ResourceVersion field
	ings := n.listers.Ingress.List()
	sort.SliceStable(ings, func(i, j int) bool {
		ir := ings[i].(*extensions.Ingress).ResourceVersion
		jr := ings[j].(*extensions.Ingress).ResourceVersion
		return ir < jr
	})

	// filter ingress rules
	var ingresses []*extensions.Ingress
	for _, ingIf := range ings {
		ing := ingIf.(*extensions.Ingress)
		if !class.IsValid(ing) {
			continue
		}

		ingresses = append(ingresses, ing)
	}

	upstreams, servers := n.getBackendServers(ingresses)

	pcfg := ingress.Configuration{
		Backends: upstreams,
		Servers:  servers,
	}

	if n.runningConfig.Equal(&pcfg) {
		glog.V(3).Infof("skipping backend reload (no changes detected)")
		return nil
	}

	glog.Infof("backend reload required")

	err := n.OnUpdate(pcfg)
	if err != nil {
		glog.Errorf("unexpected failure restarting the backend: \n%v", err)
		return err
	}

	glog.Infof("ingress backend successfully reloaded...")

	n.runningConfig = &pcfg
	n.SetForceReload(false)

	return nil
}

// readSecrets extracts information about secrets from an Ingress rule
func (n *NGINXController) readSecrets(ing *extensions.Ingress) {
	for _, tls := range ing.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}

		key := fmt.Sprintf("%v/%v", ing.Namespace, tls.SecretName)
		n.syncSecret(key)
	}

	key, _ := parser.GetStringAnnotation("auth-tls-secret", ing)
	if key == "" {
		return
	}
	n.syncSecret(key)
}

// getKubernetesUpstream create kubernetes upstream
func (n *NGINXController) getKubernetesUpstream() *ingress.Backend {
	upstream := &ingress.Backend{
		Name: kubernetesUpstreamName,
	}

	svcObj, svcExists, err := n.listers.Service.GetByKey(kubernetesSvc)
	if err != nil {
		glog.Warningf("unexpected error searching the kubernetes backend %v: %v", kubernetesSvc, err)
		return upstream
	}

	if !svcExists {
		glog.Warningf("service %v does not exist", kubernetesSvc)
		return upstream
	}

	svc := svcObj.(*apiv1.Service)
	upstream.Service = svc
	upstream.ClusterIP = svc.Spec.ClusterIP
	upstream.Port = intstr.FromInt(443)
	upstream.Secure = true
	return upstream
}

// createUpstreams creates the NGINX upstreams for each service referenced in
// Ingress rules. The servers inside the upstream are endpoints.
func (n *NGINXController) createUpstreams(data []*extensions.Ingress, ku *ingress.Backend) map[string]*ingress.Backend {
	upstreams := make(map[string]*ingress.Backend)
	upstreams[kubernetesUpstreamName] = ku

	for _, ing := range data {
		anns := n.getIngressAnnotations(ing)

		var defBackend string
		if ing.Spec.Backend != nil {
			defBackend = fmt.Sprintf("%v-%v-%v",
				ing.GetNamespace(),
				ing.Spec.Backend.ServiceName,
				ing.Spec.Backend.ServicePort.String())

			glog.V(3).Infof("creating upstream %v", defBackend)
			upstreams[defBackend] = newUpstream(defBackend)
			if !upstreams[defBackend].Secure {
				upstreams[defBackend].Secure = anns.SecureUpstream.Secure
			}
			if upstreams[defBackend].SecureCACert.Secret == "" {
				upstreams[defBackend].SecureCACert = anns.SecureUpstream.CACert
			}
			if upstreams[defBackend].UpstreamHashBy == "" {
				upstreams[defBackend].UpstreamHashBy = anns.UpstreamHashBy
			}
			if upstreams[defBackend].ClientCACert.Secret == "" {
				upstreams[defBackend].ClientCACert = anns.SecureUpstream.ClientCACert
			}
		}

		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}

			for _, path := range rule.HTTP.Paths {
				name := fmt.Sprintf("%v-%v-%v",
					ing.GetNamespace(),
					path.Backend.ServiceName,
					path.Backend.ServicePort.String())

				if _, ok := upstreams[name]; ok {
					continue
				}

				glog.V(3).Infof("creating upstream %v", name)
				upstreams[name] = newUpstream(name)
				upstreams[name].Port = path.Backend.ServicePort

				if !upstreams[name].Secure {
					upstreams[name].Secure = anns.SecureUpstream.Secure
				}

				if upstreams[name].SecureCACert.Secret == "" {
					upstreams[name].SecureCACert = anns.SecureUpstream.CACert
				}

				if upstreams[name].UpstreamHashBy == "" {
					upstreams[name].UpstreamHashBy = anns.UpstreamHashBy
				}

				if upstreams[name].ClientCACert.Secret == "" {
					upstreams[name].ClientCACert = anns.SecureUpstream.ClientCACert
				}

				svcKey := fmt.Sprintf("%v/%v", ing.GetNamespace(), path.Backend.ServiceName)

				s, err := n.listers.Service.GetByName(svcKey)
				if err != nil {
					glog.Warningf("error obtaining service: %v", err)
					continue
				}

				upstreams[name].Service = s
				upstreams[name].ClusterIP = s.Spec.ClusterIP
			}
		}
	}

	return upstreams
}

// createServers initializes a map that contains information about the list of
// FDQN referenced by ingress rules and the common name field in the referenced
// SSL certificates. Each server is configured with location / using a default
// backend specified by the user or the one inside the ingress spec.
func (n *NGINXController) createServers(data []*extensions.Ingress,
	upstreams map[string]*ingress.Backend,
	ku *ingress.Backend) map[string]*ingress.Server {

	servers := make(map[string]*ingress.Server, len(data))

	// Tries to fetch the default Certificate from nginx configuration.
	// If it does not exists, use the ones generated on Start()
	var defaultPemFileName, defaultPemSHA string
	defaultCertificate, err := n.getPemCertificate(n.cfg.DefaultSSLCertificate)
	if err == nil {
		defaultPemFileName = defaultCertificate.PemFileName
		defaultPemSHA = defaultCertificate.PemSHA
	}

	// initialize the default server
	servers[defServerName] = &ingress.Server{
		Hostname:       defServerName,
		SSLCertificate: defaultPemFileName,
		SSLPemChecksum: defaultPemSHA,
		Locations: []*ingress.Location{
			{
				Path:     kubernetesLocation,
				Backend:  ku.Name,
				Service:  ku.Service,
				AuthType: ingress.IDToken,
				Rewrite: rewrite.Config{
					Target: "/",
				},
				Proxy: proxy.DefaultProxyConfig,
			},
		}}

	// initialize all the servers
	for _, ing := range data {
		anns := n.getIngressAnnotations(ing)
		un := ""

		if ing.Spec.Backend != nil {
			// replace default backend
			defUpstream := fmt.Sprintf("%v-%v-%v", ing.GetNamespace(), ing.Spec.Backend.ServiceName, ing.Spec.Backend.ServicePort.String())
			if backendUpstream, ok := upstreams[defUpstream]; ok {
				un = backendUpstream.Name

				// Special case:
				// ingress only with a backend and no rules
				// this case defines a "catch all" server
				defLoc := servers[defServerName].Locations[0]
				if len(ing.Spec.Rules) == 0 {
					defLoc.Backend = backendUpstream.Name
					defLoc.Service = backendUpstream.Service
					defLoc.Ingress = ing

					// we need to use the ingress annotations
					defLoc.ConfigurationSnippet = anns.ConfigurationSnippet
				}
			}
		}

		for _, rule := range ing.Spec.Rules {
			host := rule.Host
			if host == "" {
				host = defServerName
			}
			if _, ok := servers[host]; ok {
				// server already configured
				continue
			}

			servers[host] = &ingress.Server{
				Hostname: host,
				Locations: []*ingress.Location{
					{
						Path:    rootLocation,
						Backend: un,
						Service: &apiv1.Service{},
					},
				},
			}
		}
	}

	// configure default location, alias, and SSL
	for _, ing := range data {
		for _, rule := range ing.Spec.Rules {
			host := rule.Host
			if host == "" {
				host = defServerName
			}

			// only add a certificate if the server does not have one previously configured
			if servers[host].SSLCertificate != "" {
				continue
			}

			if len(ing.Spec.TLS) == 0 {
				glog.V(3).Infof("ingress %v/%v for host %v does not contains a TLS section", ing.Namespace, ing.Name, host)
				continue
			}

			tlsSecretName := ""
			found := false
			for _, tls := range ing.Spec.TLS {
				if sets.NewString(tls.Hosts...).Has(host) {
					tlsSecretName = tls.SecretName
					found = true
					break
				}
			}

			if !found {
				// does not contains a TLS section but none of the host match
				continue
			}

			if tlsSecretName == "" {
				glog.V(3).Infof("host %v is listed on tls section but secretName is empty. Using default cert", host)
				servers[host].SSLCertificate = defaultPemFileName
				servers[host].SSLPemChecksum = defaultPemSHA
				continue
			}

			key := fmt.Sprintf("%v/%v", ing.Namespace, tlsSecretName)
			bc, exists := n.sslCertTracker.Get(key)
			if !exists {
				glog.Warningf("ssl certificate \"%v\" does not exist in local store", key)
				continue
			}

			cert := bc.(*ingress.SSLCert)
			err = cert.Certificate.VerifyHostname(host)

			servers[host].SSLCertificate = cert.PemFileName
			servers[host].SSLFullChainCertificate = cert.FullChainPemFileName
			servers[host].SSLPemChecksum = cert.PemSHA
			servers[host].SSLExpireTime = cert.ExpireTime
		}
	}

	return servers
}

// getBackendServers returns a list of Upstream and Server to be used by the backend
// An upstream can be used in multiple servers if the namespace, service name and port are the same
func (n *NGINXController) getBackendServers(ingresses []*extensions.Ingress) ([]*ingress.Backend, []*ingress.Server) {
	ku := n.getKubernetesUpstream()
	upstreams := n.createUpstreams(ingresses, ku)
	servers := n.createServers(ingresses, upstreams, ku)

	for _, ing := range ingresses {
		anns := n.getIngressAnnotations(ing)

		for _, rule := range ing.Spec.Rules {
			host := rule.Host
			if host == "" {
				host = defServerName
			}
			server := servers[host]
			if server == nil {
				server = servers[defServerName]
			}

			if rule.HTTP == nil &&
				host != defServerName {
				glog.V(3).Infof("ingress rule %v/%v does not contain HTTP rules, using default backend", ing.Namespace, ing.Name)
				continue
			}

			for _, path := range rule.HTTP.Paths {
				upsName := fmt.Sprintf("%v-%v-%v",
					ing.GetNamespace(),
					path.Backend.ServiceName,
					path.Backend.ServicePort.String())

				ups := upstreams[upsName]

				// if there's no path defined we assume /
				nginxPath := rootLocation
				if path.Path != "" {
					nginxPath = path.Path
				}

				addLoc := true
				for _, loc := range server.Locations {
					if loc.Path == nginxPath {
						addLoc = false

						if ups.ClusterIP == "" {
							break
						}

						glog.V(3).Infof("replacing ingress rule %v/%v location %v upstream %v (%v)", ing.Namespace, ing.Name, loc.Path, ups.Name, loc.Backend)
						loc.Backend = ups.Name
						loc.Port = ups.Port
						loc.Service = ups.Service
						loc.Ingress = ing
						loc.ConfigurationSnippet = anns.ConfigurationSnippet
						loc.Rewrite = anns.Rewrite
						loc.Proxy = anns.Proxy
						loc.XForwardedPrefix = anns.XForwardedPrefix
						loc.AuthType = anns.AuthType
						loc.AuthzType = anns.AuthzType
						loc.UpstreamURI = anns.UpstreamURI
						loc.LocationModifier = anns.LocationModifier
						loc.Connection = anns.Connection
						break
					}
				}
				// is a new location
				if addLoc {
					glog.V(3).Infof("adding location %v in ingress rule %v/%v upstream %v", nginxPath, ing.Namespace, ing.Name, ups.Name)
					if ups.ClusterIP == "" {
						continue
					}

					loc := &ingress.Location{
						Path:                 nginxPath,
						Backend:              ups.Name,
						Service:              ups.Service,
						Port:                 ups.Port,
						Ingress:              ing,
						ConfigurationSnippet: anns.ConfigurationSnippet,
						Rewrite:              anns.Rewrite,
						Proxy:                anns.Proxy,
						XForwardedPrefix:     anns.XForwardedPrefix,
						AuthType:             anns.AuthType,
						AuthzType:            anns.AuthzType,
						LocationModifier:     anns.LocationModifier,
						UpstreamURI:          anns.UpstreamURI,
						Connection:           anns.Connection,
					}

					server.Locations = append(server.Locations, loc)
				}
			}
		}
	}

	aUpstreams := make([]*ingress.Backend, 0, len(upstreams))

	// create the list of upstreams and skip those without endpoints
	for _, upstream := range upstreams {
		if upstream.ClusterIP == "" {
			continue
		}
		aUpstreams = append(aUpstreams, upstream)
	}

	aServers := make([]*ingress.Server, 0, len(servers))
	for _, value := range servers {
		sort.SliceStable(value.Locations, func(i, j int) bool {
			return value.Locations[i].Path > value.Locations[j].Path
		})
		aServers = append(aServers, value)
	}

	sort.SliceStable(aServers, func(i, j int) bool {
		return aServers[i].Hostname < aServers[j].Hostname
	})

	return aUpstreams, aServers
}

// GetAuthCertificate is used by the auth-tls annotations to get a cert from a secret
func (n NGINXController) GetAuthCertificate(name string) (*resolver.AuthSSLCert, error) {
	if _, exists := n.sslCertTracker.Get(name); !exists {
		n.syncSecret(name)
	}

	_, err := n.listers.Secret.GetByName(name)
	if err != nil {
		return &resolver.AuthSSLCert{}, fmt.Errorf("unexpected error: %v", err)
	}

	bc, exists := n.sslCertTracker.Get(name)
	if !exists {
		return &resolver.AuthSSLCert{}, fmt.Errorf("secret %v does not exist", name)
	}
	cert := bc.(*ingress.SSLCert)
	return &resolver.AuthSSLCert{
		Secret:      name,
		CAFileName:  cert.CAFileName,
		PemFileName: cert.PemFileName,
		PemSHA:      cert.PemSHA,
	}, nil
}

// GetSecret searches for a secret in the local secrets Store
func (n NGINXController) GetSecret(name string) (*apiv1.Secret, error) {
	return n.listers.Secret.GetByName(name)
}

// GetService searches for a service in the local secrets Store
func (n NGINXController) GetService(name string) (*apiv1.Service, error) {
	return n.listers.Service.GetByName(name)
}

func (n *NGINXController) extractAnnotations(ing *extensions.Ingress) {
	glog.V(3).Infof("updating annotations information for ingress %v/%v", ing.Namespace, ing.Name)
	anns := n.annotations.Extract(ing)
	err := n.listers.IngressAnnotation.Update(anns)
	if err != nil {
		glog.Errorf("unexpected error updating annotations information for ingress %v/%v: %v", anns.Namespace, anns.Name, err)
	}
}

// getByIngress returns the parsed annotations from an Ingress
func (n *NGINXController) getIngressAnnotations(ing *extensions.Ingress) *annotations.Ingress {
	key := fmt.Sprintf("%v/%v", ing.Namespace, ing.Name)
	item, exists, err := n.listers.IngressAnnotation.GetByKey(key)
	if err != nil {
		glog.Errorf("unexpected error getting ingress annotation %v: %v", key, err)
		return &annotations.Ingress{}
	}
	if !exists {
		glog.Errorf("ingress annotation %v was not found", key)
		return &annotations.Ingress{}
	}
	return item.(*annotations.Ingress)
}

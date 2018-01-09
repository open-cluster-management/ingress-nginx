package ingress

import (
	"time"

	apiv1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/annotations/rewrite"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/resolver"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/store"
)

var (
	// DefaultSSLDirectory defines the location where the SSL certificates will be generated
	// This directory contains all the SSL certificates that are specified in Ingress rules.
	// The name of each file is <namespace>-<secret name>.pem. The content is the concatenated
	// certificate and key.
	DefaultSSLDirectory = "/opt/ibm/router/nginx/ssl"
)

const (
	// IDToken auth type
	IDToken = "id-token"
	// AccessToken auth type
	AccessToken = "access-token"
)

// StoreLister returns the configured stores for ingresses, services,
// endpoints, secrets and configmaps.
type StoreLister struct {
	Ingress           store.IngressLister
	Service           store.ServiceLister
	Endpoint          store.EndpointLister
	Secret            store.SecretLister
	IngressAnnotation store.IngressAnnotationsLister
}

// Configuration holds the definition of all the parts required to describe all
// ingresses reachable by the ingress controller (using a filter by namespace)
type Configuration struct {
	// Backends are a list of backends used by all the Ingress rules in the
	// ingress controller. This list includes the default backend
	Backends []*Backend `json:"backends,omitEmpty"`
	// Servers
	Servers []*Server `json:"servers,omitEmpty"`
}

// Backend describes one or more remote server/s (endpoints) associated with a service
// +k8s:deepcopy-gen=true
type Backend struct {
	// Name represents an unique apiv1.Service name formatted as <namespace>-<name>-<port>
	Name      string             `json:"name"`
	Service   *apiv1.Service     `json:"service,omitempty"`
	Port      intstr.IntOrString `json:"port"`
	ClusterIP string             `json:"clusterIP"`
	// This indicates if the communication protocol between the backend and the endpoint is HTTP or HTTPS
	// Allowing the use of HTTPS
	// The endpoint/s must provide a TLS connection.
	// The certificate used in the endpoint cannot be a self signed certificate
	Secure bool `json:"secure"`
	// SecureCACert has the filename and SHA1 of the certificate authorities used to validate
	// a secured connection to the backend
	SecureCACert resolver.AuthSSLCert `json:"secureCACert"`
	// Consistent hashing by NGINX variable
	UpstreamHashBy string `json:"upstream-hash-by,omitempty"`
}

// Server describes a website
type Server struct {
	// Hostname returns the FQDN of the server
	Hostname string `json:"hostname"`
	// Locations list of URIs configured in the server.
	Locations []*Location `json:"locations,omitempty"`
	// SSLCertificate path to the SSL certificate on disk
	SSLCertificate string `json:"sslCertificate"`
	// SSLFullChainCertificate path to the SSL certificate on disk
	// This certificate contains the full chain (ca + intermediates + cert)
	SSLFullChainCertificate string `json:"sslFullChainCertificate"`
	// SSLExpireTime has the expire date of this certificate
	SSLExpireTime time.Time `json:"sslExpireTime"`
	// SSLPemChecksum returns the checksum of the certificate file on disk.
	// There is no restriction in the hash generator. This checksim can be
	// used to  determine if the secret changed without the use of file
	// system notifications
	SSLPemChecksum string `json:"sslPemChecksum"`
	// Alias return the alias of the server name
	Alias string `json:"alias,omitempty"`
}

// Location describes an URI inside a server.
// Also contains additional information about annotations in the Ingress.
//
// In some cases when more than one annotations is defined a particular order in the execution
// is required.
// The chain in the execution order of annotations should be:
// - Whitelist
// - RateLimit
// - BasicDigestAuth
// - ExternalAuth
// - Redirect
type Location struct {
	// Path is an extended POSIX regex as defined by IEEE Std 1003.1,
	// (i.e this follows the egrep/unix syntax, not the perl syntax)
	// matched against the path of an incoming request. Currently it can
	// contain characters disallowed from the conventional "path"
	// part of a URL as defined by RFC 3986. Paths must begin with
	// a '/'. If unspecified, the path defaults to a catch all sending
	// traffic to the backend.
	Path string `json:"path"`
	// Ingress returns the ingress from which this location was generated
	Ingress *extensions.Ingress `json:"ingress"`
	// Backend describes the name of the backend to use.
	Backend string `json:"backend"`
	// Service describes the referenced services from the ingress
	Service *apiv1.Service `json:"service,omitempty"`
	// Port describes to which port from the service
	Port intstr.IntOrString `json:"port"`
	// ConfigurationSnippet contains additional configuration for the backend
	// to be considered in the configuration of the location
	ConfigurationSnippet string `json:"configurationSnippet"`
	// Rewrite describes the redirection this location.
	// +optional
	Rewrite rewrite.Config `json:"rewrite,omitempty"`
	// XForwardedPrefix allows to add a header X-Forwarded-Prefix to the request with the
	// original location.
	// +optional
	XForwardedPrefix bool `json:"xForwardedPrefix,omitempty"`
	// AuthType indicates the authentication method used in the location
	AuthType string `json:"authType,omitempty"`
	// AuthzType indicates the authorization method used in the location
	AuthzType string `json:"authzType,omitempty"`
	// Location Modifier indicates the location match operator
	LocationModifier string `json:"locationModifier,omitempty"`
	// Upstream uri gives the additional uri to the current path of location
	UpstreamURI string `json:"upstreamURI,omitempty"`
}

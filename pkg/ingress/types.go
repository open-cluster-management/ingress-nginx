package ingress

import "github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/store"

// StoreLister returns the configured stores for ingresses, services,
// endpoints, secrets and configmaps.
type StoreLister struct {
	Ingress           store.IngressLister
	Service           store.ServiceLister
	Endpoint          store.EndpointLister
	Secret            store.SecretLister
	ConfigMap         store.ConfigMapLister
	IngressAnnotation store.IngressAnnotationsLister
}

package config

// ListenPorts describe the ports required to run the
// NGINX Ingress controller
type ListenPorts struct {
	HTTP  int
	HTTPS int
}

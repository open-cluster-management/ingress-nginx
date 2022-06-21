# Management Ingress

## Overview
management-ingress is a NGINX based ingress controller to serve Open Cluster Management service.

## Get started
You can add below Kubernetes annotations to specific Ingress objects to customize their behavior. See the [examples](examples) to get more details of the usage.

| Name | Description | Values |
| --- | --- | --- |
| ingress.open-cluster-management.io/auth-type | Authentication method for management service | string |
| ingress.open-cluster-management.io/authz-type | Authorization method for management service | string |
| ingress.open-cluster-management.io/rewrite-target | Target URI where the traffic must be redirected | string |
| ingress.open-cluster-management.io/app-root | Base URI fort the server | string |
| ingress.open-cluster-management.io/configuration-snippet | Additional configuration to the NGINX location | string |
| ingress.open-cluster-management.io/secure-backends | uses https to reach the services | bool |
| ingress.open-cluster-management.io/secure-verify-ca-secret | secret name that stores ca cert for upstream service | string |
| ingress.open-cluster-management.io/secure-client-ca-secret | secret name that stores ca cert/key for client authentication of upstream server | string |
| ingress.open-cluster-management.io/upstream-uri | URI of upstream | string |
| ingress.open-cluster-management.io/location-modifier | Location modifier | string |
| ingress.open-cluster-management.io/proxy-connect-timeout | proxy connect timeout | string |
| ingress.open-cluster-management.io/proxy-send-timeout | proxy send timeout | string |
| ingress.open-cluster-management.io/proxy-read-timeout | proxy read timeout | string |
| ingress.open-cluster-management.io/proxy-buffer-size | buffer size of response | string |
| ingress.open-cluster-management.io/proxy-body-size | max response body | string |
| ingress.open-cluster-management.io/connection | override connection header | string |

## Developing
### Prerequisites
- Go 1.15+
- Docker v19.03.0+
- OpenShift 3.11+

### Build the binary
```shell
make docker-binary
```

### Build the image
```shell
make docker-image
```

### Installation
Follow [management-ingress-chart](https://github.com/stolostron/management-ingress-chart) documentation to install management ingress in your OpenShift cluster, and replace the deployment `management-ingress` image name with your own.

## Contributing
* See [CONTRIBUTING.md](CONTRIBUTING.md) for information about the workflow that we expect, and instructions on the developer certificate of origin that we require.

Rebuild Image: Thu Jun 16 12:48:59 EDT 2022

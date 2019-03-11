# icp-management-ingress
nginx based ingress controller to server all icp management service

The design doc is [here](https://github.ibm.com/IBMPrivateCloud/roadmap/blob/master/feature-specs/kubernetes/management-ingress-controller-refactor.md)

# annotations

| Name | Description | Values |
| --- | --- | --- |
| icp.management.ibm.com/auth-type | Authentication method for management service | string |
| icp.management.ibm.com/authz-type | Authorization method for management service | string |
| icp.management.ibm.com/rewrite-target | Target URI where the traffic must be redirected | string |
| icp.management.ibm.com/app-root | Base URI fort the server | string |
| icp.management.ibm.com/configuration-snippet | Additional configuration to the NGINX location | string |
| icp.management.ibm.com/secure-backends | uses https to reach the services | bool |
| icp.management.ibm.com/secure-verify-ca-secret | secret name that stores ca cert for upstream service | string |
| icp.management.ibm.com/secure-client-ca-secret | secret name that stores ca cert/key for client authentication of upstream server | string |
| icp.management.ibm.com/upstream-uri | URI of upstream | string |
| icp.management.ibm.com/location-modifier | Location modifier | string |
| icp.management.ibm.com/proxy-connect-timeout | proxy connect timeout | string |
| icp.management.ibm.com/proxy-send-timeout | proxy send timeout | string |
| icp.management.ibm.com/proxy-read-timeout | proxy read timeout | string |
| icp.management.ibm.com/proxy-buffer-size | buffer size of response | string |
| icp.management.ibm.com/proxy-body-size | max response body | string |
| icp.management.ibm.com/connection | override connection header | string |

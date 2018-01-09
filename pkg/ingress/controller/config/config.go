package config

import (
	apiv1 "k8s.io/api/core/v1"

	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress"
)

// Configuration represents the content of nginx.conf file
type Configuration struct {
	// Sets the name of the configmap that contains the headers to pass to the client
	AddHeaders string `json:"add-headers,omitempty"`

	// AllowBackendServerHeader enables the return of the header Server from the backend
	// instead of the generic nginx string.
	// http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_hide_header
	// By default this is disabled
	AllowBackendServerHeader bool `json:"allow-backend-server-header"`

	// AccessLogPath sets the path of the access logs if enabled
	// http://nginx.org/en/docs/http/ngx_http_log_module.html#access_log
	// By default access logs go to /var/log/nginx/access.log
	AccessLogPath string `json:"access-log-path,omitempty"`

	// ErrorLogPath sets the path of the error logs
	// http://nginx.org/en/docs/ngx_core_module.html#error_log
	// By default error logs go to /var/log/nginx/error.log
	ErrorLogPath string `json:"error-log-path,omitempty"`

	// EnableDynamicTLSRecords enables dynamic TLS record sizes
	// https://blog.cloudflare.com/optimizing-tls-over-tcp-to-reduce-latency
	// By default this is enabled
	EnableDynamicTLSRecords bool `json:"enable-dynamic-tls-records"`

	// EnableModsecurity enables the modsecurity module for NGINX
	// By default this is disabled
	EnableModsecurity bool `json:"enable-modsecurity"`

	// EnableModsecurity enables the OWASP ModSecurity Core Rule Set (CRS)
	// By default this is disabled
	EnableOWASPCoreRules bool `json:"enable-owasp-modsecurity-crs"`

	// ClientHeaderBufferSize allows to configure a custom buffer
	// size for reading client request header
	// http://nginx.org/en/docs/http/ngx_http_core_module.html#client_header_buffer_size
	ClientHeaderBufferSize string `json:"client-header-buffer-size"`

	// Defines a timeout for reading client request header, in seconds
	// http://nginx.org/en/docs/http/ngx_http_core_module.html#client_header_timeout
	ClientHeaderTimeout int `json:"client-header-timeout,omitempty"`

	// Sets buffer size for reading client request body
	// http://nginx.org/en/docs/http/ngx_http_core_module.html#client_body_buffer_size
	ClientBodyBufferSize string `json:"client-body-buffer-size,omitempty"`

	// Defines a timeout for reading client request body, in seconds
	// http://nginx.org/en/docs/http/ngx_http_core_module.html#client_body_timeout
	ClientBodyTimeout int `json:"client-body-timeout,omitempty"`

	// DisableAccessLog disables the Access Log globally from NGINX ingress controller
	//http://nginx.org/en/docs/http/ngx_http_log_module.html
	DisableAccessLog bool `json:"disable-access-log,omitempty"`

	// DisableIpv6 disable listening on ipv6 address
	DisableIpv6 bool `json:"disable-ipv6,omitempty"`

	// EnableUnderscoresInHeaders enables underscores in header names
	// http://nginx.org/en/docs/http/ngx_http_core_module.html#underscores_in_headers
	// By default this is disabled
	EnableUnderscoresInHeaders bool `json:"enable-underscores-in-headers"`

	// IgnoreInvalidHeaders set if header fields with invalid names should be ignored
	// http://nginx.org/en/docs/http/ngx_http_core_module.html#ignore_invalid_headers
	// By default this is enabled
	IgnoreInvalidHeaders bool `json:"ignore-invalid-headers"`

	// EnableVtsStatus allows the replacement of the default status page with a third party module named
	// nginx-module-vts - https://github.com/vozlt/nginx-module-vts
	// By default this is disabled
	EnableVtsStatus bool `json:"enable-vts-status,omitempty"`

	// Vts config on http level
	// Description: Sets parameters for a shared memory zone that will keep states for various keys. The cache is shared between all worker processe
	// https://github.com/vozlt/nginx-module-vts#vhost_traffic_status_zone
	// Default value is 10m
	VtsStatusZoneSize string `json:"vts-status-zone-size,omitempty"`

	// Vts config on http level
	// Description: Enables the keys by user defined variable. The key is a key string to calculate traffic.
	// The name is a group string to calculate traffic. The key and name can contain variables such as $host,
	// $server_name. The name's group belongs to filterZones if specified. The key's group belongs to serverZones
	// if not specified second argument name. The example with geoip module is as follows:
	// https://github.com/vozlt/nginx-module-vts#vhost_traffic_status_filter_by_set_key
	// Default value is $geoip_country_code country::*
	VtsDefaultFilterKey string `json:"vts-default-filter-key,omitempty"`

	// RetryNonIdempotent since 1.9.13 NGINX will not retry non-idempotent requests (POST, LOCK, PATCH)
	// in case of an error. The previous behavior can be restored using the value true
	RetryNonIdempotent bool `json:"retry-non-idempotent"`

	// http://nginx.org/en/docs/ngx_core_module.html#error_log
	// Configures logging level [debug | info | notice | warn | error | crit | alert | emerg]
	// Log levels above are listed in the order of increasing severity
	ErrorLogLevel string `json:"error-log-level,omitempty"`

	// https://nginx.org/en/docs/http/ngx_http_v2_module.html#http2_max_field_size
	// HTTP2MaxFieldSize Limits the maximum size of an HPACK-compressed request header field
	HTTP2MaxFieldSize string `json:"http2-max-field-size,omitempty"`

	// https://nginx.org/en/docs/http/ngx_http_v2_module.html#http2_max_header_size
	// HTTP2MaxHeaderSize Limits the maximum size of the entire request header list after HPACK decompression
	HTTP2MaxHeaderSize string `json:"http2-max-header-size,omitempty"`

	// Enables or disables the header HSTS in servers running SSL
	HSTS bool `json:"hsts,omitempty"`

	// Enables or disables the use of HSTS in all the subdomains of the servername
	// Default: true
	HSTSIncludeSubdomains bool `json:"hsts-include-subdomains,omitempty"`

	// HTTP Strict Transport Security (often abbreviated as HSTS) is a security feature (HTTP header)
	// that tell browsers that it should only be communicated with using HTTPS, instead of using HTTP.
	// https://developer.mozilla.org/en-US/docs/Web/Security/HTTP_strict_transport_security
	// max-age is the time, in seconds, that the browser should remember that this site is only to be
	// accessed using HTTPS.
	HSTSMaxAge string `json:"hsts-max-age,omitempty"`

	// Enables or disables the preload attribute in HSTS feature
	HSTSPreload bool `json:"hsts-preload,omitempty"`

	// Time during which a keep-alive client connection will stay open on the server side.
	// The zero value disables keep-alive client connections
	// http://nginx.org/en/docs/http/ngx_http_core_module.html#keepalive_timeout
	KeepAlive int `json:"keep-alive,omitempty"`

	// Sets the maximum number of requests that can be served through one keep-alive connection.
	// http://nginx.org/en/docs/http/ngx_http_core_module.html#keepalive_requests
	KeepAliveRequests int `json:"keep-alive-requests,omitempty"`

	// LargeClientHeaderBuffers Sets the maximum number and size of buffers used for reading
	// large client request header.
	// http://nginx.org/en/docs/http/ngx_http_core_module.html#large_client_header_buffers
	// Default: 4 8k
	LargeClientHeaderBuffers string `json:"large-client-header-buffers"`

	// Enable json escaping
	// http://nginx.org/en/docs/http/ngx_http_log_module.html#log_format
	LogFormatEscapeJSON bool `json:"log-format-escape-json,omitempty"`

	// Customize upstream log_format
	// http://nginx.org/en/docs/http/ngx_http_log_module.html#log_format
	LogFormatUpstream string `json:"log-format-upstream,omitempty"`

	// Customize stream log_format
	// http://nginx.org/en/docs/http/ngx_http_log_module.html#log_format
	LogFormatStream string `json:"log-format-stream,omitempty"`

	// Maximum number of simultaneous connections that can be opened by each worker process
	// http://nginx.org/en/docs/ngx_core_module.html#worker_connections
	MaxWorkerConnections int `json:"max-worker-connections,omitempty"`

	// Sets the bucket size for the map variables hash tables.
	// Default value depends on the processor’s cache line size.
	// http://nginx.org/en/docs/http/ngx_http_map_module.html#map_hash_bucket_size
	MapHashBucketSize int `json:"map-hash-bucket-size,omitempty"`

	// If UseProxyProtocol is enabled ProxyRealIPCIDR defines the default the IP/network address
	// of your external load balancer
	ProxyRealIPCIDR []string `json:"proxy-real-ip-cidr,omitempty"`

	// Sets the name of the configmap that contains the headers to pass to the backend
	ProxySetHeaders string `json:"proxy-set-headers,omitempty"`

	// Maximum size of the server names hash tables used in server names, map directive’s values,
	// MIME types, names of request header strings, etcd.
	// http://nginx.org/en/docs/hash.html
	// http://nginx.org/en/docs/http/ngx_http_core_module.html#server_names_hash_max_size
	ServerNameHashMaxSize int `json:"server-name-hash-max-size,omitempty"`

	// Size of the bucket for the server names hash tables
	// http://nginx.org/en/docs/hash.html
	// http://nginx.org/en/docs/http/ngx_http_core_module.html#server_names_hash_bucket_size
	ServerNameHashBucketSize int `json:"server-name-hash-bucket-size,omitempty"`

	// Size of the bucket for the proxy headers hash tables
	// http://nginx.org/en/docs/hash.html
	// https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_headers_hash_max_size
	ProxyHeadersHashMaxSize int `json:"proxy-headers-hash-max-size,omitempty"`

	// Maximum size of the bucket for the proxy headers hash tables
	// http://nginx.org/en/docs/hash.html
	// https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_headers_hash_bucket_size
	ProxyHeadersHashBucketSize int `json:"proxy-headers-hash-bucket-size,omitempty"`

	// Enables or disables emitting nginx version in error messages and in the “Server” response header field.
	// http://nginx.org/en/docs/http/ngx_http_core_module.html#server_tokens
	// Default: true
	ShowServerTokens bool `json:"server-tokens"`

	// Enabled ciphers list to enabled. The ciphers are specified in the format understood by
	// the OpenSSL library
	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_ciphers
	SSLCiphers string `json:"ssl-ciphers,omitempty"`

	// Specifies a curve for ECDHE ciphers.
	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_ecdh_curve
	SSLECDHCurve string `json:"ssl-ecdh-curve,omitempty"`

	// The secret that contains Diffie-Hellman key to help with "Perfect Forward Secrecy"
	// https://www.openssl.org/docs/manmaster/apps/dhparam.html
	// https://wiki.mozilla.org/Security/Server_Side_TLS#DHE_handshake_and_dhparam
	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_dhparam
	SSLDHParam string `json:"ssl-dh-param,omitempty"`

	// SSL enabled protocols to use
	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_protocols
	SSLProtocols string `json:"ssl-protocols,omitempty"`

	// Enables or disables the use of shared SSL cache among worker processes.
	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_session_cache
	SSLSessionCache bool `json:"ssl-session-cache,omitempty"`

	// Size of the SSL shared cache between all worker processes.
	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_session_cache
	SSLSessionCacheSize string `json:"ssl-session-cache-size,omitempty"`

	// Enables or disables session resumption through TLS session tickets.
	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_session_tickets
	SSLSessionTickets bool `json:"ssl-session-tickets,omitempty"`

	// Sets the secret key used to encrypt and decrypt TLS session tickets.
	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_session_tickets
	// By default, a randomly generated key is used.
	// Example: openssl rand 80 | base64 -w0
	SSLSessionTicketKey string `json:"ssl-session-ticket-key,omitempty"`

	// Time during which a client may reuse the session parameters stored in a cache.
	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_session_timeout
	SSLSessionTimeout string `json:"ssl-session-timeout,omitempty"`

	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_buffer_size
	// Sets the size of the buffer used for sending data.
	// 4k helps NGINX to improve TLS Time To First Byte (TTTFB)
	// https://www.igvita.com/2013/12/16/optimizing-nginx-tls-time-to-first-byte/
	SSLBufferSize string `json:"ssl-buffer-size,omitempty"`

	// Enables or disables the use of the PROXY protocol to receive client connection
	// (real IP address) information passed through proxy servers and load balancers
	// such as HAproxy and Amazon Elastic Load Balancer (ELB).
	// https://www.nginx.com/resources/admin-guide/proxy-protocol/
	UseProxyProtocol bool `json:"use-proxy-protocol,omitempty"`

	// Enables or disables the use of the nginx module that compresses responses using the "gzip" method
	// http://nginx.org/en/docs/http/ngx_http_gzip_module.html
	UseGzip bool `json:"use-gzip,omitempty"`

	// Enables or disables the use of the NGINX Brotli Module for compression
	// https://github.com/google/ngx_brotli
	EnableBrotli bool `json:"enable-brotli,omitempty"`

	// Brotli Compression Level that will be used
	BrotliLevel int `json:"brotli-level,omitempty"`

	// MIME Types that will be compressed on-the-fly using Brotli module
	BrotliTypes string `json:"brotli-types,omitempty"`

	// Enables or disables the HTTP/2 support in secure connections
	// http://nginx.org/en/docs/http/ngx_http_v2_module.html
	// Default: true
	UseHTTP2 bool `json:"use-http2,omitempty"`

	// MIME types in addition to "text/html" to compress. The special value “*” matches any MIME type.
	// Responses with the “text/html” type are always compressed if UseGzip is enabled
	GzipTypes string `json:"gzip-types,omitempty"`

	// Defines the number of worker processes. By default auto means number of available CPU cores
	// http://nginx.org/en/docs/ngx_core_module.html#worker_processes
	WorkerProcesses string `json:"worker-processes,omitempty"`

	// Defines a timeout for a graceful shutdown of worker processes
	// http://nginx.org/en/docs/ngx_core_module.html#worker_shutdown_timeout
	WorkerShutdownTimeout string `json:"worker-shutdown-timeout,omitempty"`

	// Defines the load balancing algorithm to use. The deault is round-robin
	LoadBalanceAlgorithm string `json:"load-balance,omitempty"`

	// Sets the bucket size for the variables hash table.
	// http://nginx.org/en/docs/http/ngx_http_map_module.html#variables_hash_bucket_size
	VariablesHashBucketSize int `json:"variables-hash-bucket-size,omitempty"`

	// Sets the maximum size of the variables hash table.
	// http://nginx.org/en/docs/http/ngx_http_map_module.html#variables_hash_max_size
	VariablesHashMaxSize int `json:"variables-hash-max-size,omitempty"`

	// Activates the cache for connections to upstream servers.
	// The connections parameter sets the maximum number of idle keepalive connections to
	// upstream servers that are preserved in the cache of each worker process. When this
	// number is exceeded, the least recently used connections are closed.
	// http://nginx.org/en/docs/http/ngx_http_upstream_module.html#keepalive
	// Default: 32
	UpstreamKeepaliveConnections int `json:"upstream-keepalive-connections,omitempty"`

	// Sets the maximum size of the variables hash table.
	// http://nginx.org/en/docs/http/ngx_http_map_module.html#variables_hash_max_size
	LimitConnZoneVariable string `json:"limit-conn-zone-variable,omitempty"`

	// Sets the timeout between two successive read or write operations on client or proxied server connections.
	// If no data is transmitted within this time, the connection is closed.
	// http://nginx.org/en/docs/stream/ngx_stream_proxy_module.html#proxy_timeout
	ProxyStreamTimeout string `json:"proxy-stream-timeout,omitempty"`

	// Sets the number of datagrams expected from the proxied server in response
	// to the client request if the UDP protocol is used.
	// http://nginx.org/en/docs/stream/ngx_stream_proxy_module.html#proxy_responses
	// Default: 1
	ProxyStreamResponses int `json:"proxy-stream-responses,omitempty"`

	// Sets the ipv4 addresses on which the server will accept requests.
	BindAddressIpv4 []string `json:"bind-address-ipv4,omitempty"`

	// Sets the ipv6 addresses on which the server will accept requests.
	BindAddressIpv6 []string `json:"bind-address-ipv6,omitempty"`

	// Sets the header field for identifying the originating IP address of a client
	// Default is X-Forwarded-For
	ForwardedForHeader string `json:"forwarded-for-header,omitempty"`

	// Append the remote address to the X-Forwarded-For header instead of replacing it
	// Default: false
	ComputeFullForwardedFor bool `json:"compute-full-forwarded-for,omitempty"`

	// EnableOpentracing enables the nginx Opentracing extension
	// https://github.com/rnburn/nginx-opentracing
	// By default this is disabled
	EnableOpentracing bool `json:"enable-opentracing"`

	// ZipkinCollectorHost specifies the host to use when uploading traces
	ZipkinCollectorHost string `json:"zipkin-collector-host"`

	// ZipkinCollectorPort specifies the port to use when uploading traces
	ZipkinCollectorPort int `json:"zipkin-collector-port"`

	// ZipkinServiceName specifies the service name to use for any traces created
	// Default: nginx
	ZipkinServiceName string `json:"zipkin-service-name"`

	// HTTPSnippet adds custom configuration to the http section of the nginx configuration
	HTTPSnippet string `json:"http-snippet"`

	// ServerSnippet adds custom configuration to all the servers in the nginx configuration
	ServerSnippet string `json:"server-snippet"`

	// LocationSnippet adds custom configuration to all the locations in the nginx configuration
	LocationSnippet string `json:"location-snippet"`

	// HTTPRedirectCode sets the HTTP status code to be used in redirects.
	// Supported codes are 301,302,307 and 308
	// Default: 308
	HTTPRedirectCode int `json:"http-redirect-code"`
}

// TemplateConfig contains the nginx configuration to render the file nginx.conf
type TemplateConfig struct {
	ProxySetHeaders         map[string]string
	AddHeaders              map[string]string
	MaxOpenFiles            int
	BacklogSize             int
	Backends                []*ingress.Backend
	Servers                 []*ingress.Server
	HealthzURI              string
	CustomErrors            bool
	Cfg                     Configuration
	IsIPV6Enabled           bool
	IsSSLPassthroughEnabled bool
	RedirectServers         map[string]string
	ListenPorts             *ListenPorts
	PublishService          *apiv1.Service
}

// ListenPorts describe the ports required to run the
// NGINX Ingress controller
type ListenPorts struct {
	HTTP  int
	HTTPS int
}

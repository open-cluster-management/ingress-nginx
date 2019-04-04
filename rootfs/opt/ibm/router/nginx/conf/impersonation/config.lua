--
-- Impersonating authorization module for a token stored in a secret.
--
local _M = {}

--
-- The auth module to use.  Either service-account or secret
--
_M.auth_module = "service-account"
_M.secret = {
    path = "/var/run/secrets/kubernetes.io/serviceaccount/token",
}
_M.service_account = {
    name = "default",
    namespace = "icp-system",
    cluster_role_binding = "admin-users",
    kube_secret_path = "/var/run/secrets/kubernetes.io/serviceaccount/token"
}

return _M
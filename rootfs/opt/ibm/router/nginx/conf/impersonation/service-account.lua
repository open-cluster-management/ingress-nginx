local tokens = ngx.shared.tokens
local config = require "impersonation.config"
local common = require "impersonation.common"
local cjson = require "cjson"
local http = require "lib.resty.http"
local ngx_decode_base64 = ngx.decode_base64

local sa = common.getenv("IMPERSONATION_SA_NAME", config.service_account.name)
local sa_ns = common.getenv("IMPERSONATION_SA_NAMESPACE", config.service_account.namespace)
local cluster_role_binding = common.getenv("IMPERSONATION_SA_CLUSTERROLEBINDING", config.service_account.cluster_role_binding)
local kube_port = common.getenv("APISERVER_SECURE_PORT", "8001")


local function checkForErrors(rsp)
    if not rsp then                                                                                                                          
        return err                                                                           
    end                                                                                     
    if (rsp.body == "" or rsp.body == nil) then
        return "The response body is empty."
    end
    if rsp.status < 200 or rsp.status > 299 then
        return "Status: " .. rsp.status .. " Body: " .. rsp.body 
    end
    return nil
end

local function set_kube_token()
    --
    -- Get the default service account token
    --
    local kube_token = common.read_file(config.service_account.kube_secret_path)
    if not kube_token then
        return "No token found in file: " .. config.service_account.kube_secret_path .. ". Check that that the secret contains the token and the secret is mounted at the correct path."
    end
    kube_token = "Bearer " .. kube_token

    --
    -- Call kube to get the service account secret
    --- 
    local httpc = http.new()
    local rsp, err = httpc:request_uri("https://127.0.0.1:" .. kube_port .. "/api/v1/namespaces/" .. sa_ns .. "/serviceaccounts/" .. sa, {
        ssl_verify = false,
        method = "GET",                                                                                                                       
        headers = {["Authorization"] = kube_token}                                                                                  
    })

    err = checkForErrors(rsp)
    if err then                                                                                                                    
        return "Failed to access impersonation service account: " .. sa_ns .. "/" .. sa .. " Error: " .. err                                                                          
    end                                                                                         

    local body = tostring(rsp.body)
    body = cjson.decode(body)
    local sa_secret = body.secrets[1].name

    --
    -- Verify the service account is in the cluster role binding
    --
    local httpc = http.new()
    local rsp, err = httpc:request_uri("https://127.0.0.1:" .. kube_port .. "/apis/rbac.authorization.k8s.io/v1/clusterrolebindings/" .. cluster_role_binding, {
        ssl_verify = false,
        method = "GET",                                                                                                                       
        headers = {["Authorization"] = kube_token}                                                                                  
    })

    err = checkForErrors(rsp)
    if err then                                                                                                                    
        return "Failed to access cluster role binding: " .. cluster_role_binding .. " Error: " .. err                                                                          
    end
    body = tostring(rsp.body)
    body = cjson.decode(body)
    local subjects = body.subjects
    local found = false
    if #subjects > 0 then
        for i = 1, #subjects do
           if subjects[i].kind == "ServiceAccount" and subjects[i].name == sa and subjects[i].namespace == sa_ns then
              found = true
           end
        end
    else 
        return "There are no subjects in the cluster role binding: " .. cluster_role_binding
    end
    if not found then
        return "The service account " .. sa_ns .. "/" .. sa .. " is not found in cluster role binding " .. cluster_role_binding
    end
 
    --
    -- Call kube to get the token from the secret
    --
    rsp, err = httpc:request_uri("https://127.0.0.1:" .. kube_port .. "/api/v1/namespaces/" .. sa_ns .. "/secrets/" .. sa_secret, {
        ssl_verify = false,
        method = "GET",                                                                                                                       
        headers = {["Authorization"] = kube_token}                                                                                  
    })

    err = checkForErrors(rsp)
    if err then                                                                                                                    
        return "Failed to access impersonation service account secret: " .. sa_ns .. "/" .. sa .. "/" .. sa_secret .. " Error: " .. err                                                                         
    end                                                                                                                                           
    
    body = tostring(rsp.body)
    body = cjson.decode(body)
    local token = ngx_decode_base64(body.data.token)
    
    --
    -- Add the kube token to the shared dictionary
    --
    local success, err, force = tokens:set(common.kube_token, token)
    return nil
end

-- Expose interface.                                               
local _M = {}                                           
_M.set_kube_token = set_kube_token                                             
return _M

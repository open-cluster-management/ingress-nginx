local tokens = ngx.shared.tokens
local config = require "impersonation.config"
local common = require "impersonation.common"

local function set_kube_token()
    token = common.read_file(config.secret.path)
    if not token then
        return "No token found in file: " .. config.secret.path .. ". Check that that the secret contains the token and the secret is mounted at the correct path."
    end
    local success, err, force = tokens:set(common.kube_token, token)
end

-- Expose interface.                                               
local _M = {}                                           
_M.set_kube_token = set_kube_token                                             
return _M

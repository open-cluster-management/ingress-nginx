--
-- Adds the impersonation headers for kubernetes calls.
--
-- First the a kubernetes token is retrieved from the impersonation auth module.  By default
-- this is a service account token.  This token is stored in tokens shared dictionary so it doesn't
-- need to be retreived on every request.
-- Next the ICP id token is retreieved from the authorization header and verified.
-- Next the kube token is added to the Authorization header and the impersonation headers
-- are added from the ICP id token; the user (from the sub propertery and the groups from the teamrolemappings
-- property).
-- Finally, the updated request is returned and sent to kubernetes. 

local tokens = ngx.shared.tokens
local cjson = require "cjson"
local config = require "impersonation.config"
local impersonation_auth = require("impersonation." .. config.auth_module)
local common = require "impersonation.common"
local jwt = require "resty.jwt"
local validators = require "resty.jwt-validators"
local jwt_public_key = nil
local wlp_client_id = common.getenv("WLP_CLIENT_ID", "")
local oidc_issuer_url = common.getenv("OIDC_ISSUER_URL", "")

--
-- Return 403 forbidden error
--
local function exit_403(msg)
    ngx.status = ngx.HTTP_FORBIDDEN
    ngx.header["Content-Type"] = "text/html; charset=UTF-8"
    ngx.say("[Impersonation-LUA]" .. msg)
    ngx.log(ngx.ERR, "Impersonation error: " .. msg)
    return ngx.exit(ngx.HTTP_FORBIDDEN)
end

--
-- Log the impersonation headers
--
local function dump_impersonation_headers()
    local start = "impersonate"
    local headers = ""
    h = ngx.req.get_headers()                                      
    for k, v in pairs(h) do     
     if k:sub(1, #start) == start then
        if type(v) == "table" then
            for k1, v1 in pairs(v) do
                headers = headers .. " " .. k .. ": " .. v1
            end
        else
            headers = headers .. " "  .. k .. ": " .. v
        end
      end
    end 
    ngx.log(ngx.NOTICE, headers)
end

--
-- Splits an id token into its 3 parts and returns a table with each part as an entry.
--
local function split_token(token)
    if token:find(".", 1, true) then
        local t = {}
        for str in token:gmatch("([^%.]+)") do
            table.insert(t, str)
        end
        return t
    end
    return nil
end

--
-- Get a value from the shared dictionary
--
local function get_token_shared_dict(name)
    return tokens:get(name)
end

--
-- Retrieves the public key that can be used to verify the id token
--
local function get_public_key()
    if jwt_public_key == nil then
       jwt_public_key = common.read_file("/var/run/secrets/platform-auth-public.pem")
    end
    return jwt_public_key
end

--
-- Verifies an id token by ensuring its signature is valid, its not expired, its audience value matches
-- the client id, and the issuer is what we expect.
--
local function verify_id_token(id_token)
    jwt:set_alg_whitelist({ RS256 = 1 })
    jwt_obj = jwt:verify(get_public_key(), 
                         id_token,
                        {
                            exp = validators.opt_is_not_expired(),
                            iss = validators.opt_equals(oidc_issuer_url),
                            aud = validators.opt_equals(wlp_client_id)
                        })
    return jwt_obj['verified'], jwt_obj['reason']
end

--
-- Get the kube token from the shared dictionary.  If it's not there then call
-- the configured authorization module to set the kube token.  The authorization module
-- is configured in the config.lua module _M.auth_module
--
local function get_kube_token()
    local token = get_token_shared_dict(common.kube_token)
    local err = nil
    if token == nil then
        err = impersonation_auth.set_kube_token()
    end
    if err then
        return nil, err
    end
    token = get_token_shared_dict(common.kube_token)
    return token, err
end

--
-- Add the Impersonate-User header to the request.
--
local function add_impersonation_headers_for_user(user)
    ngx.req.set_header('Impersonate-User', user)
end

--
-- Adds the impersonation headers for a the user and groups to the request.
--
local function add_impersonation_headers(user, groups)
    local header_value = {}
    if #groups > 0 then
        for groupCount = 1, #groups do
           header_value[groupCount] = groups[groupCount]
        end
    end
    add_impersonation_headers_for_user(user)
    ngx.req.set_header('Impersonate-Group', header_value)
end

--
-- Add the kube token to the authorization header.
--
local function add_kube_auth_header(kube_token) 
    ngx.req.set_header('Authorization', 'Bearer ' .. kube_token)
end

--
-- Main entry point.
--
-- Check the token in the authorization header.  If we can't spilt out the token info part of the token
-- the it's not a valid id token.
-- If it can be spilt, then check who the issuer is.  If the issuer is kubernetes service
-- account then just send that token to kube directly without adding any impersonation headers.
-- Otherwise the token should have been issued by ICP, so verify the token and if valid
-- add the impersonation headers.
--
local function add_auth_headers()
    local auth_header = ngx.req.get_headers()["Authorization"]
    if not auth_header then
        return exit_403("No token found in the authorization header.")
    end

    local t = split_token(auth_header)
    if not t then
        return exit_403("Authorization header does not contain a valid id token.")
    end

    local token_info_string = t[2]
    if not token_info_string then
        return exit_403("Provided token is not in the correct format.")
    end

    local token_info_json = ngx.decode_base64(token_info_string)
    local token_info = cjson.decode(token_info_json)
    local issuer = token_info.iss
    if issuer ~= "kubernetes/serviceaccount" then
        local _, _, id_token = string.find(auth_header, "Bearer%s+(.+)")
        local valid_token, fail_reason = verify_id_token(id_token)
        if valid_token then
            local kube_token, err = get_kube_token()
            if err then
                exit_403("Error getting a kubernetes token to use for impersonation: " .. err)
            end
            add_impersonation_headers(token_info.sub, token_info.teamRoleMappings)
            add_kube_auth_header(kube_token)
            dump_impersonation_headers()
        else
            return exit_403("Unable to validate token: " .. fail_reason)
        end     
    end
end

--
-- Removed the kube token from the shared dictionary.  It will be repopulated on the next request.
--
local function reset()
    tokens:set(common.kube_token, nil)
end

-- Expose interface.                                               
local _M = {}                                           
_M.add_auth_headers = add_auth_headers
_M.reset = reset                      
return _M


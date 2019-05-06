local cjson = require "cjson"
local jwt = require "resty.jwt"
local cookiejar = require "resty.cookie"
local http = require "lib.resty.http"
local impersonate = require "impersonation.impersonate"

local common = require "common"

local SECRET_KEY = nil
local BODY_AUTH_ERROR_RESPONSE = nil

local errorpages_dir_path = os.getenv("AUTH_ERROR_PAGE_DIR_PATH")
local cluster_domain = os.getenv("CLUSTER_DOMAIN")
local impersonation_enabled = os.getenv("ENABLE_IMPERSONATION")

if errorpages_dir_path == nil then
    ngx.log(ngx.WARN, "AUTH_ERROR_PAGE_DIR_PATH not set.")
else
    local p = errorpages_dir_path .. "/401.html"
    ngx.log(ngx.NOTICE, "Reading 401 response from `" .. p .. "`.")
    BODY_401_ERROR_RESPONSE = common.get_file_content(p)
    if (BODY_401_ERROR_RESPONSE == nil or BODY_401_ERROR_RESPONSE == '') then
        -- Normalize to '', for sending empty response bodies.
        BODY_401_ERROR_RESPONSE = ''
        ngx.log(ngx.WARN, "401 error response is empty.")
    end
    local p = errorpages_dir_path .. "/403.html"
    ngx.log(ngx.NOTICE, "Reading 403 response from `" .. p .. "`.")
    BODY_403_ERROR_RESPONSE = common.get_file_content(p)
    if (BODY_403_ERROR_RESPONSE == nil or BODY_403_ERROR_RESPONSE == '') then
        -- Normalize to '', for sending empty response bodies.
        BODY_403_ERROR_RESPONSE = ''
        ngx.log(ngx.WARN, "403 error response is empty.")
    end
    local p = errorpages_dir_path .. "/404.html"
    ngx.log(ngx.NOTICE, "Reading 404 response from `" .. p .. "`.")
    BODY_403_ERROR_RESPONSE = common.get_file_content(p)
    if (BODY_404_ERROR_RESPONSE == nil or BODY_404_ERROR_RESPONSE == '') then
        -- Normalize to '', for sending empty response bodies.
        BODY_404_ERROR_RESPONSE = ''
        ngx.log(ngx.WARN, "404 error response is empty.")
    end
end

-- Refs:
-- https://github.com/openresty/lua-nginx-module#access_by_lua
-- https://github.com/SkyLothar/lua-resty-jwt


local function exit_401()
    ngx.status = ngx.HTTP_UNAUTHORIZED
    ngx.header["Content-Type"] = "text/html; charset=UTF-8"
    ngx.header["WWW-Authenticate"] = "oauthjwt"
    ngx.say(BODY_401_ERROR_RESPONSE)
    return ngx.exit(ngx.HTTP_UNAUTHORIZED)
end


local function exit_403()
    ngx.status = ngx.HTTP_FORBIDDEN
    ngx.header["Content-Type"] = "text/html; charset=UTF-8"
    ngx.say(BODY_403_ERROR_RESPONSE)
    return ngx.exit(ngx.HTTP_FORBIDDEN)
end

local function exit_404()
    ngx.status = ngx.HTTP_NOT_FOUND
    ngx.header["Content-Type"] = "text/html; charset=UTF-8"
    ngx.say(BODY_404_ERROR_RESPONSE)
    return ngx.exit(ngx.HTTP_NOT_FOUND)
end

local function pdp_exit_403()
    ngx.status = ngx.HTTP_FORBIDDEN
    ngx.header["Content-Type"] = "text/html; charset=UTF-8"
    ngx.say("Error 403 : Access Forbidden" )
    return ngx.exit(ngx.HTTP_FORBIDDEN)
end

local function validate_id_token_or_exit()
    -- Inspect Authorization header in current request.
    local auth_header = ngx.var.http_Authorization
    local token = nil

    if auth_header ~= nil then
        ngx.log(ngx.DEBUG, "Authorization header found. Attempt to extract token.")
         _, _, token = string.find(auth_header, "Bearer%s+(.+)")
    else
        ngx.log(ngx.DEBUG, "Authorization header not found.")
        local cookie, err = cookiejar:new()
        token = cookie:get("cfc-access-token-cookie")
        if token == nil then
            ngx.log(ngx.DEBUG, "cfc-access-token-cookie not found.")
            return exit_401()
        else
            local httpc = http.new()
            local res, err = httpc:request_uri("https://platform-identity-provider.kube-system.svc."..cluster_domain..":4300/v1/auth/exchangetoken", {
              method = "POST",
              ssl_verify = false,
              body = "access_token=" .. token,
              headers = {["Content-Type"] = "application/x-www-form-urlencoded"}
            })
            if not res then
              ngx.log(ngx.NOTICE, "Failed to request exchangetoken=",err)
              return exit_401()
            end
            ngx.log(ngx.NOTICE, "Response status =",res.status)
            if (res.body == "" or res.body == nil) then
              ngx.log(ngx.NOTICE, "Empty response body=",err)
              return exit_401()
            end
            local x = tostring(res.body)
            local data = cjson.decode(x).id_token
            ngx.log(ngx.DEBUG, "id token:",data)
            ngx.log(
                ngx.DEBUG, "Use token from cfc-access-token-cookie, " ..
                "set corresponding Authorization header for upstream."
                )
            ngx.req.set_header('Authorization', 'Bearer '.. data)
        end
    end
    if impersonation_enabled == 'true' then
      impersonate.add_auth_headers()
    end
    return data
end

local function validate_access_token_or_exit()
    local auth_header = ngx.var.http_Authorization
    local token = nil
    if auth_header ~= nil then
        ngx.log(ngx.DEBUG, "Authorization header found. Attempt to extract token.")
        _, _, token = string.find(auth_header, "Bearer%s+(.+)")
    else
        ngx.log(ngx.DEBUG, "Authorization header not found.")
        -- Presence of Authorization header overrides cookie method entirely.
        -- Read cookie. Note: ngx.var.cookie_* cannot access a cookie with a
        -- dash in its name.
        local cookie, err = cookiejar:new()
        token = cookie:get("cfc-access-token-cookie")
        if token == nil then
            ngx.log(ngx.DEBUG, "cfc-access-token-cookie not found.")
        else
            ngx.log(
                ngx.DEBUG, "Use token from cfc-access-token-cookie, " ..
                "set corresponding Authorization header for upstream."
                )
            ngx.req.set_header('Authorization', 'Bearer '.. token)
        end
    end

    if token == nil then
        ngx.log(ngx.NOTICE, "No auth token in request.")
        return exit_401()
    end

    ngx.log(ngx.DEBUG, "Received OIDC token.")
    local httpc = http.new()
    local res, err = httpc:request_uri("https://platform-identity-provider.kube-system.svc."..cluster_domain..":4300/v1/auth/userInfo", {
        method = "POST",
        ssl_verify = false,
        body = "access_token=" .. token,
        headers = {
          ["Content-Type"] = "application/x-www-form-urlencoded",
        }
      })

    if not res then
        ngx.log(ngx.NOTICE, "Failed to request userinfo=",err)
        return exit_401()
    end
      ngx.log(ngx.NOTICE, "Response status =",res.status)
    if (res.body == "" or res.body == nil or res.status >= 400)
    then
        ngx.log(ngx.NOTICE, "Empty response body=",err)
        return exit_401()
    elseif (res.status == 200)
    then
      local x = tostring(res.body)
      local data = cjson.decode(x).sub
      ngx.log(ngx.DEBUG, "UID:",data)
    end
  return data
end

local function validateauthuri()
  if ngx.var.request_uri == "/idprovider/v1/auth/getClientCredentials/" then
    return true
  end
  if ngx.var.request_uri == "/idprovider/v1/auth/getClientCredentials" then
    return true
  end
  if ngx.var.request_uri == "/idprovider/v1/auth/admintoken/" then
    return true
  end
  if ngx.var.request_uri == "/idprovider/v1/auth/admintoken" then
    return true
  end
  return false
end

local function validate_policy_or_exit()
      local httpc = http.new()
      ngx.log(ngx.NOTICE, "URL=https://iam-pdp.kube-system.svc."..cluster_domain..":7998/v1/authz")

      local method = ngx.req.get_method()
      ngx.log(ngx.NOTICE, "Method = ", method)

      ngx.log(ngx.NOTICE, "URI=", ngx.var.request_uri)
      local list = {}
      for word in string.gmatch(ngx.var.request_uri,'([^/]+)') do table.insert(list,word) end
      if list[1] == "idprovider" and not validateauthuri()  then
        if string.find(ngx.var.request_uri, "getClientCredentials//") or string.find(ngx.var.request_uri, "admintoken//") then
           return pdp_exit_403()
        else
           return 0
        end
      end

      local auth_token = ngx.req.get_headers()["Authorization"]
      ngx.log(ngx.DEBUG, "Auth Token received.")

      local cookie, err = cookiejar:new()
      local token = cookie:get("cfc-access-token-cookie")

      if token == nil then
         ngx.log(ngx.DEBUG, "cfc-access-token-cookie not found.")
      end

      if auth_token == nil and token == nil then
         return exit_401()
      end

      if auth_token == nil then
        auth_token="bearer " .. token
        ngx.log(ngx.NOTICE, "Auth token taken from cookie.")
      end

      if auth_token == nil then
        ngx.log(ngx.NOTICE, "No auth token in request.")
        return exit_401()
      end

      uri = method.." "..ngx.var.request_uri
      ngx.log(ngx.NOTICE, "Full URI = ", uri)

      ngx.log(ngx.DEBUG, "New Token received")
      local data = {
           ["action"] = uri,
           ["subject"] = {
                   ["id"] = "",
                   ["type"] = ""
           },
           ["resource"] = {
                   ["crn"] = "",
                   ["attributes"] = {
                              ["serviceName"] = "",
                              ["accountId"] = ""
                   }
           }
      }
      local res, err = httpc:request_uri("https://iam-pdp.kube-system.svc."..cluster_domain..":7998/v1/authz", {
        method = "POST",
        ssl_verify = false,
        body = cjson.encode(data),
        headers = {
          ["Content-Type"] = "application/json",
          ["Accept"] = "application/json",
          ["Authorization"] = auth_token
        }
      })

    if not res then
        ngx.log(ngx.NOTICE, "Failed to call pdp =",err)
        return pdp_exit_403()
    end

      ngx.log(ngx.NOTICE, "Response status =",res.status)
    if (res.body == "" or res.body == nil) then
        ngx.log(ngx.NOTICE, "Failed to call pdp =",err)
        return pdp_exit_403()
    end
      local x = tostring(res.body)
      local data = cjson.decode(x).decision
      ngx.log(ngx.NOTICE, "UID:",data)

   if data ~= "Permit" then
        ngx.log(ngx.NOTICE, "Access Denied by PDP")
        return pdp_exit_403()
   end

  return data
end



-- Expose interface.
local _M = {}
_M.validate_access_token_or_exit = validate_access_token_or_exit
_M.validate_id_token_or_exit = validate_id_token_or_exit
_M.validate_policy_or_exit = validate_policy_or_exit

return _M

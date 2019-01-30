local common = require "common"

local host_headers_check_enabled = os.getenv("HOST_HEADERS_CHECK_ENABLED");
local allowed_host_headers = os.getenv("ALLOWED_HOST_HEADERS");
local https_port = os.getenv("HTTPS_PORT");

local function exit_403()
    ngx.status = ngx.HTTP_FORBIDDEN
    ngx.header["Content-Type"] = "text/html; charset=UTF-8"
    ngx.say("Error 403 : Access Forbidden" )
    return ngx.exit(ngx.HTTP_FORBIDDEN)
end

local function validate_host_header()
    if host_headers_check_enabled ~= "true" then
        ngx.log(ngx.DEBUG, "skip host header checking...")
        return
    end
    local host = ngx.req.get_headers()["host"]
    local xhost = ngx.req.get_headers()["x-forwarded-host"]
    local invalid_host = 1
    local invalid_xhost = 1
    local hosts_headers = allowed_host_headers:split()
    if (host == nil) then
       ngx.log(ngx.NOTICE, "invalid host header : "..host..".")
       return exit_403()
    end
    for k,v in pairs(hosts_headers) do
      if host == v..":"..https_port or host == v..":"..8443 then
        invalid_host = 0
      end
      if xhost == nil or xhost == v..":"..https_port or xhost == v..":"..8443 then
        invalid_xhost = 0
      end
    end
    if invalid_host == 1 then
       ngx.log(ngx.NOTICE, "invalid host header : "..host..".")
       return exit_403()
    end
    if invalid_xhost == 1 then
       ngx.log(ngx.NOTICE, "invalid x-forwarded-host header : "..xhost..".")
       return exit_403()
    end
end

-- Expose interface.
local _M = {}
_M.validate_host_header = validate_host_header

return _M

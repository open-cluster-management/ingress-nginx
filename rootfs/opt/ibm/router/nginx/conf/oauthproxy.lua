local cjson = require "cjson"
local jwt = require "resty.jwt"
local cookiejar = require "resty.cookie"
local http = require "lib.resty.http"
local common = require "common"

local SECRET_KEY = nil

local errorpages_dir_path = os.getenv("AUTH_ERROR_PAGE_DIR_PATH")

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


local function validate_access_token_or_exit()

    -- Inspect Authorization header in current request.
    local auth_header = ngx.var.http_Authorization
    local token = nil

    if auth_header ~= nil then
        ngx.log(ngx.NOTICE, "Authorization header found. Attempt to extract token.")
        _, _, token = string.find(auth_header, "Bearer%s+(.+)")
    end

    if token == nil then

        -- look for token in forwarded  header
        local forwarded_token = ngx.req.get_headers()["x-forwarded-access-token"]
        if forwarded_token ~= nil then
            ngx.log(ngx.NOTICE, "access token found in forwarded header.")

            -- set token
            token = forwarded_token

            -- attempt to set forwared token in cookie as well, if not already set
            local cookie, err = cookiejar:new()
            access_token_cookie = cookie:get("acm-access-token-cookie")
            if access_token_cookie == nil then
                ngx.log(ngx.NOTICE, "acm-access-token-cookie not found,setting it.")
                -- set cookie, max age 12h in seconds
                local ok, err = cookie:set({key = "acm-access-token-cookie", value = forwarded_token, path = "/", max_age = 43200})
                if err ~= nil then
                    ngx.log(ngx.NOTICE, "Error setting the cookie", err)
                end

            else
                ngx.log(ngx.NOTICE, "acm-access-token-cookie already set.")
            end

        else
            ngx.log(ngx.NOTICE, "Token not found in Auth header or forwarded headers.")

            -- Presence of Authorization header overrides cookie method entirely.
            -- Read cookie. Note: ngx.var.cookie_* cannot access a cookie with a
            -- dash in its name.

            local cookie, err = cookiejar:new()
            token = cookie:get("acm-access-token-cookie")
        end

        if token ~= nil then
            ngx.log(
                ngx.NOTICE, "Use token found, " ..
                "set corresponding Authorization header for upstream."
                )
            ngx.req.set_header('Authorization', 'Bearer '.. token)
        end

    end

    if token == nil then
        ngx.log(ngx.NOTICE, "No auth token in request.")
        return exit_401()
    end

    ngx.log(ngx.NOTICE, "Received access token.")


    -- user info
    local userid = ngx.req.get_headers()["X-Forwarded-User"]
    if userid == nil then
        ngx.log(ngx.NOTICE, "User Id Not found")
        -- return exit_401()
    end

    ngx.log(ngx.NOTICE, "UserID =", userid)
    return userid
end

-- Expose interface.
local _M = {}
_M.validate_access_token_or_exit = validate_access_token_or_exit

return _M

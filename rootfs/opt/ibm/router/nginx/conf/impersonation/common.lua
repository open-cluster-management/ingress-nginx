
local function read_file(f)
    local file = io.open(f, "r")
    if file == nil then
      ngx.log(ngx.ERR, "Unable to open file: " .. f)
      return ""
    end
    local data = file:read("*all")
    file:close()
    return data
end

local function getenv(name, default) 
    local v = os.getenv(name)
    if v == nil then
       v = default
    end
    return v
end


-- Expose interface and constants.                                               
local _M = {}                                           
_M.read_file = read_file
_M.kube_token = "kube_token"
_M.getenv = getenv                                       
return _M
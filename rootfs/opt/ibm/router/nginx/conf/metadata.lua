ngx.header.content_type = 'application/json'

public_ip = ngx.var.server_addr

ngx.say('{"PUBLIC_IPV4": "' .. public_ip .. '"}')

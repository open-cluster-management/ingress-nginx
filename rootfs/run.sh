#!/bin/sh
if [ "${ENABLE_IMPERSONATION}" = "true" ]
then 
   echo "Adding impersonation support."
   if [ "${APISERVER_SECURE_PORT}" = "" ]
   then
       echo "APISERVER_SECURE_POrt not set, using default."
       APISERVER_SECURE_PORT="8001"
   fi
   # Update the nginx template to include the server for kubernetes. 
   sed -i "s/{{APISERVER_SECURE_PORT}}/${APISERVER_SECURE_PORT}/g" /opt/ibm/router/nginx/conf/impersonation/nginx-kube.conf
   sed -i '/http {/a \    include /opt/ibm/router/nginx/conf/impersonation/nginx-kube.conf;' /opt/ibm/router/nginx/template/nginx.tmpl
   # Get the public key from the cert that signs the id tokens.  This is used to verify the id token is valid. 
   openssl x509 -pubkey -noout -in /var/run/secrets/platform-auth/tls.crt > /var/run/secrets/platform-auth-public.pem
   echo "Impersonation support added."
fi
echo "Starting ICP Management ingress"
$@
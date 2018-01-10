FROM conductor/openresty:1.11.2.4

ARG VCS_REF
ARG VCS_URL
ARG IMAGE_NAME
ARG IMAGE_DESCRIPTION

# http://label-schema.org/rc1/
LABEL org.label-schema.vendor="IBM" \
      org.label-schema.name="$IMAGE_NAME" \
      org.label-schema.description="$IMAGE_DESCRIPTION" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url=$VCS_URL \
      org.label-schema.license="Licensed Materials - Property of IBM" \
      org.label-schema.schema-version="1.0"

ENV AUTH_ERROR_PAGE_DIR_PATH=/opt/ibm/router/nginx/conf/errorpages SECRET_KEY_FILE_PATH=/etc/cfc/conf/auth-token-secret OIDC_ENABLE=false ADMINROUTER_ACTIVATE_AUTH_MODULE=true PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/opt/ibm/router/nginx/sbin

RUN apk --no-cache add openssl python python-dev py-pip build-base diffutils \
  && pip install dumb-init \
  && apk del python python-dev py-pip build-base \
  && rm -rf /var/cache/apk/* \
  && mkdir -p /var/log/nginx \
  && ln -sf /dev/stdout /var/log/nginx/access.log \
  && ln -sf /dev/stderr /var/log/nginx/error.log

COPY rootfs /

ENTRYPOINT ["/usr/bin/dumb-init"]

CMD ["/icp-management-ingress"]

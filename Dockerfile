# Copyright (c) 2021 Red Hat, Inc.
# Copyright Contributors to the Open Cluster Management project

FROM registry.ci.openshift.org/stolostron/builder:go1.18-linux AS builder
WORKDIR /go/src/github.com/stolostron/management-ingress
COPY . .
RUN make docker-binary

FROM registry.access.redhat.com/ubi8/ubi-minimal

# Docker Build Arguments
ARG PREFIX_DIR="/opt/ibm/router"
ARG RESTY_VERSION="1.19.3.2"
ARG ROLLBACK_RESTY_VERSION="1.17.8.2"
ARG RESTY_J="1"
ARG RESTY_CONFIG_OPTIONS="\
    --with-file-aio \
    --with-http_addition_module \
    --with-http_auth_request_module \
    --with-http_dav_module \
    --with-http_flv_module \
    --with-http_gunzip_module \
    --with-http_gzip_static_module \
    --with-http_image_filter_module=dynamic \
    --with-http_mp4_module \
    --with-http_random_index_module \
    --with-http_realip_module \
    --with-http_secure_link_module \
    --with-http_slice_module \
    --with-http_ssl_module \
    --with-http_stub_status_module \
    --with-http_sub_module \
    --with-http_v2_module \
    --with-http_xslt_module=dynamic \
    --without-mail_pop3_module \
    --without-mail_imap_module \
    --without-mail_smtp_module \
    --with-mail \
    --with-mail_ssl_module \
    --with-pcre-jit \
    --with-stream \
    --with-stream_ssl_module \
    --with-threads \
    "
ARG RESTY_CONFIG_OPTIONS_MORE="--prefix=${PREFIX_DIR}"

LABEL resty_version="${RESTY_VERSION}"
LABEL resty_config_options="${RESTY_CONFIG_OPTIONS}"
LABEL resty_config_options_more="${RESTY_CONFIG_OPTIONS_MORE}"

COPY ./external-deps/* /tmp/

# 1) Install apk dependencies
RUN set -ex \
    && ARCH=$(uname -m) \
    && microdnf install -y perl \
        libxslt-devel \
        make \
        perl-devel \
        zlib-devel \
        file \
        gd \
	gd-devel \
        libgcc \
	wget \
        libxslt \
        zlib \
        gcc \
        gcc-c++ \
        fontconfig-devel \
        freetype-devel \
        libX11-devel \
        libXpm-devel \
        libjpeg-devel libpng-devel \
	tar \
        gzip \
        pcre-devel \
	openssl-devel \
        patch \
    && cd /tmp \
    && tar xzf openresty-${RESTY_VERSION}.tar.gz \
    && tar xzf openresty-${ROLLBACK_RESTY_VERSION}.tar.gz

# 2) Build OpenResty
RUN if [[ "$(uname -m)" != "s390x" ]]; then \
        cd /tmp/openresty-${RESTY_VERSION} \
        && sed -ire "s/openresty/server/g" `find ./ -name ngx_http_special_response.c` \
        # next two lines fix two compilation errors with OpenResty 1.19.3.2
        && sed -ire '1i #include "lualib.h"' `find ./ -name lj_ccallback.c` \
        && sed -ire "s/for .int /int i; for (/g" `find ./ -name lib_jit.c` \
        # patch to fix this issue https://github.com/openresty/luajit2/issues/122 in ppc64le
        && patch -l /tmp/openresty-${RESTY_VERSION}/bundle/LuaJIT-2.1-20201027/src/lj_def.h /tmp/lj_def.h.patch \
        && ./configure -j${RESTY_J} ${RESTY_CONFIG_OPTIONS} ${RESTY_CONFIG_OPTIONS_MORE} \
        && make -j${RESTY_J} \
        && make -j${RESTY_J} install \
        && ln -sf /opt/ibm/router/nginx/sbin/nginx /opt/ibm/router/bin/openresty; \
    elif [[ "$(uname -m)" = "s390x" ]]; then \
        cd /tmp \
        && rm -rf \
            openresty-${RESTY_VERSION}/bundle/LuaJIT-2.1-* \
            openresty-${RESTY_VERSION}/bundle/lua-resty-core-* \
            openresty-${RESTY_VERSION}/bundle/ngx_lua-* \
            openresty-${RESTY_VERSION}/bundle/ngx_stream_lua-* \
        && cp -r openresty-${ROLLBACK_RESTY_VERSION}/bundle/LuaJIT-2.1-* openresty-${RESTY_VERSION}/bundle/ \
        && cp -r openresty-${ROLLBACK_RESTY_VERSION}/bundle/lua-resty-core-* openresty-${RESTY_VERSION}/bundle/ \
        && cp -r openresty-${ROLLBACK_RESTY_VERSION}/bundle/ngx_lua-* openresty-${RESTY_VERSION}/bundle/ \
        && cp -r openresty-${ROLLBACK_RESTY_VERSION}/bundle/ngx_stream_lua-* openresty-${RESTY_VERSION}/bundle/ \
        && cd /tmp/openresty-${RESTY_VERSION} \
        && patch -l /tmp/openresty-${RESTY_VERSION}/configure /tmp/configure.diff \
        && ./configure -j${RESTY_J} ${RESTY_CONFIG_OPTIONS} ${RESTY_CONFIG_OPTIONS_MORE} \
        && make -j${RESTY_J} \
        && make -j${RESTY_J} install \
        && ln -sf /opt/ibm/router/nginx/sbin/nginx /opt/ibm/router/bin/openresty; \
     fi
# 3) Cleanup
RUN microdnf remove -y patch \
    && microdnf clean all \
    && cd /tmp \
    && rm -rf * \
    && mkdir -p /var/log/nginx \
    && ln -sf /dev/stdout ${PREFIX_DIR}/nginx/logs/access.log \
    && ln -sf /dev/stderr ${PREFIX_DIR}/nginx/logs/error.log \
    && ln -sf /dev/stdout /var/log/nginx/access.log \
    && ln -sf /dev/stderr /var/log/nginx/error.log

ARG VCS_REF
ARG VCS_URL
ARG IMAGE_NAME
ARG IMAGE_DESCRIPTION
ARG IMAGE_VENDOR
ARG IMAGE_SUMMARY
# http://label-schema.org/rc1/
LABEL org.label-schema.vendor="Red Hat" \
      org.label-schema.name="$IMAGE_NAME" \
      org.label-schema.description="$IMAGE_DESCRIPTION" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url=$VCS_URL \
      org.label-schema.license="Open Cluster Management for Kubernetes EULA" \
      org.label-schema.schema-version="1.0" \
      name="$IMAGE_NAME" \
      vendor="$IMAGE_VENDOR" \
      description="$IMAGE_DESCRIPTION" \
      summary="$IMAGE_SUMMARY"

ENV AUTH_ERROR_PAGE_DIR_PATH=/opt/ibm/router/nginx/conf/errorpages PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/opt/ibm/router/nginx/sbin

COPY --from=builder /go/src/github.com/stolostron/management-ingress/rootfs /

RUN chmod -R 777 /opt/ibm/router

USER 1001

CMD ["/management-ingress"]

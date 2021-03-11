# Copyright (c) 2020 Red Hat, Inc.
# Dockerfile - alpine
# https://github.com/openresty/docker-openresty

ARG RESTY_IMAGE_BASE="alpine"
ARG RESTY_IMAGE_TAG="latest"

#FROM ${RESTY_IMAGE_BASE}:${RESTY_IMAGE_TAG}
FROM registry.access.redhat.com/ubi7/ubi:7.9 AS openresty_base
LABEL maintainer="Evan Wies <evan@neomantra.net>"

# Docker Build Arguments
ARG PREFIX_DIR="/opt/ibm/router"
ARG RESTY_VERSION="1.13.6.2"
ARG RESTY_OPENSSL_VERSION="1.1.1j"
ARG RESTY_PCRE_VERSION="8.42"
ARG RESTY_J="1"
ARG RESTY_CONFIG_OPTIONS="\
    --with-file-aio \
    --with-http_addition_module \
    --with-http_auth_request_module \
    --with-http_dav_module \
    --with-http_flv_module \
    --with-http_geoip_module=dynamic \
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
    --with-ipv6 \
    --with-mail \
    --with-mail_ssl_module \
    --with-md5-asm \
    --with-pcre-jit \
    --with-sha1-asm \
    --with-stream \
    --with-stream_ssl_module \
    --with-threads \
    "
ARG RESTY_CONFIG_OPTIONS_MORE="--prefix=${PREFIX_DIR}"

LABEL resty_version="${RESTY_VERSION}"
LABEL resty_openssl_version="${RESTY_OPENSSL_VERSION}"
LABEL resty_pcre_version="${RESTY_PCRE_VERSION}"
LABEL resty_config_options="${RESTY_CONFIG_OPTIONS}"
LABEL resty_config_options_more="${RESTY_CONFIG_OPTIONS_MORE}"

# These are not intended to be user-specified
ARG _RESTY_CONFIG_DEPS="--with-luajit --with-openssl=/tmp/openssl-${RESTY_OPENSSL_VERSION} --with-pcre=/tmp/pcre-${RESTY_PCRE_VERSION}"

COPY docker/openresty/1.13.6.2/non-fips-code/Makefile.openssl1.1.1j.patch /tmp/Makefile.openssl1.1.1j.patch

# 1) Install apk dependencies
# 2) Download and untar OpenSSL, PCRE, and OpenResty
# 3) Build OpenResty
# 4) Cleanup

# 1) Install apk dependencies
# 2) Download and untar OpenSSL, PCRE, and OpenResty

COPY external-deps/openssl-${RESTY_OPENSSL_VERSION}.tar.gz /tmp
COPY external-deps/pcre-${RESTY_PCRE_VERSION}.tar.gz /tmp
COPY external-deps/openresty-${RESTY_VERSION}.tar.gz /tmp
COPY external-deps/centos-release-7-7.1908.0.el7.centos.x86_64.rpm /tmp
COPY external-deps/alsadi-dumb-init-epel-7.repo /etc/yum.repos.d/alsadi-dumb-init-epel-7.repo

RUN yum install --skip-broken -y perl \
        libxslt-devel \
        linux-headers \
        make \
        perl-devel \
        zlib-devel \
        file \
        gd \
        libgcc \
        libxslt \
        zlib \
        gcc \
        gcc-c++ \
        fontconfig-devel \
        freetype-devel \
        libX11-devel \
        libXpm-devel \
        libjpeg-devel libpng-devel \
        patch \
# backup ubi release info
        && mkdir /tmp/release && mv /etc/*release* /tmp/release \
        && rpm -Uvh --force /tmp/centos-release-7-7.1908.0.el7.centos.x86_64.rpm && sed -i 's/$releasever/7/g' /etc/yum.repos.d/* \
    && yum install --skip-broken -y GeoIP-devel \
        ncurses-devel \
        readline-devel \
        kernel-devel \
        gd-devel \
# recovery ubi release info
        && rm /etc/*release* && mv /tmp/release/* /etc/ && rm -rf /tmp/release \
    && cd /tmp \
    && tar xzf openssl-${RESTY_OPENSSL_VERSION}.tar.gz \
    && tar xzf pcre-${RESTY_PCRE_VERSION}.tar.gz \
    && tar xzf openresty-${RESTY_VERSION}.tar.gz \
# 3) Build OpenResty
    && cd /tmp/openresty-${RESTY_VERSION} \
    && sed -ire "s/openresty/server/g" `find ./ -name ngx_http_special_response.c` \
    && ./configure -j${RESTY_J} ${_RESTY_CONFIG_DEPS} ${RESTY_CONFIG_OPTIONS} ${RESTY_CONFIG_OPTIONS_MORE} \
    && patch ./build/nginx-1.13.6/objs/Makefile /tmp/Makefile.openssl1.1.1j.patch \
    && make -j${RESTY_J} \
    && make -j${RESTY_J} install \
    && ln -sf /opt/ibm/router/nginx/sbin/nginx /opt/ibm/router/bin/openresty \
# 4) Cleanup
    && yum remove -y patch \
    && yum clean all \
    && cd /tmp \
    && rm -rf \
        openssl-${RESTY_OPENSSL_VERSION} \
        openssl-${RESTY_OPENSSL_VERSION}.tar.gz \
        openresty-${RESTY_VERSION}.tar.gz openresty-${RESTY_VERSION} \
        pcre-${RESTY_PCRE_VERSION}.tar.gz pcre-${RESTY_PCRE_VERSION} \
        Makefile.openssl1.1.1j.patch \
    && ln -sf /dev/stdout ${PREFIX_DIR}/nginx/logs/access.log \
    && ln -sf /dev/stderr ${PREFIX_DIR}/nginx/logs/error.log

# Add additional binaries into PATH for convenience
ENV PATH=$PATH:${PREFIX_DIR}/luajit/bin:${PREFIX_DIR}/nginx/sbin:${PREFIX_DIR}/bin

# Copy nginx configuration files
# COPY nginx.conf ${PREFIX_DIR}/nginx/conf/nginx.conf

# CMD ["${PREFIX_DIR}/bin/openresty", "-g", "daemon off;"]

FROM openresty_base

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
      org.label-schema.license="Red Hat Advanced Cluster Management for Kubernetes EULA" \
      org.label-schema.schema-version="1.0" \
      name="$IMAGE_NAME" \
      vendor="$IMAGE_VENDOR" \
      description="$IMAGE_DESCRIPTION" \
      summary="$IMAGE_SUMMARY"

ENV AUTH_ERROR_PAGE_DIR_PATH=/opt/ibm/router/nginx/conf/errorpages SECRET_KEY_FILE_PATH=/etc/cfc/conf/auth-token-secret OIDC_ENABLE=false ADMINROUTER_ACTIVATE_AUTH_MODULE=true PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/opt/ibm/router/nginx/sbin

RUN yum remove -y centos-release \
  && yum update -y --exclude=GeoIP* --exclude=readline* \
  && yum install -y  openssl python python-devl \
  && yum install -y dumb-init \
  && rpm -e kernel-devel \
  && yum clean all \
  && mkdir -p /var/log/nginx \
  && rpm -e kernel-headers glibc-headers --nodeps \
  && ln -sf /dev/stdout /var/log/nginx/access.log \
  && ln -sf /dev/stderr /var/log/nginx/error.log

COPY rootfs /

RUN chmod -R 777 /opt/ibm/router

USER 1001

ENTRYPOINT ["/usr/bin/dumb-init"]

CMD ["/management-ingress"]

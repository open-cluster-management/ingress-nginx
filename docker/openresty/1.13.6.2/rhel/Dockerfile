# Dockerfile - rhel 7.5 (centos 7) - RPM version
# https://github.com/openresty/docker-openresty

FROM registry.access.redhat.com/rhel7/rhel:latest

LABEL maintainer="Evan Wies <evan@neomantra.net>"

ARG RESTY_IMAGE_BASE="rhel"
ARG RESTY_LUAROCKS_VERSION="2.4.4"
ARG RESTY_RPM_FLAVOR=""
ARG RESTY_RPM_VERSION="yum-3.4.3-158.el7.noarch"
ARG RESTY_RPM_ARCH="x86_64"

LABEL resty_luarocks_version="${RESTY_LUAROCKS_VERSION}"
LABEL resty_rpm_flavor="${RESTY_RPM_FLAVOR}"
LABEL resty_rpm_version="${RESTY_RPM_VERSION}"
LABEL resty_rpm_arch="${RESTY_RPM_ARCH}"

RUN yum-config-manager --add-repo https://openresty.org/package/${RESTY_IMAGE_BASE}/openresty.repo \
    && yum update -y \
    && yum upgrade -y \
    && yum install -y \
        gettext \
        make \
        openresty.${RESTY_RPM_ARCH} \
        openresty-opm.noarch \
        openresty-resty.noarch \
        unzip \
    && yum install -y openssl wget \
    && cd /tmp \
    && curl -fSL https://github.com/luarocks/luarocks/archive/${RESTY_LUAROCKS_VERSION}.tar.gz -o luarocks-${RESTY_LUAROCKS_VERSION}.tar.gz \
    && tar xzf luarocks-${RESTY_LUAROCKS_VERSION}.tar.gz \
    && cd luarocks-${RESTY_LUAROCKS_VERSION} \
    && ./configure \
        --prefix=/usr/local/openresty/luajit \
        --with-lua=/usr/local/openresty/luajit \
        --lua-suffix=jit-2.1.0-beta3 \
        --with-lua-include=/usr/local/openresty/luajit/include/luajit-2.1 \
    && make build \
    && make install \
    && cd /tmp \
    && wget https://bootstrap.pypa.io/get-pip.py \
    && python get-pip.py \
    && pip install --no-cache-dir dumb-init \
    && rm get-pip.py \
    && rm -rf luarocks-${RESTY_LUAROCKS_VERSION} luarocks-${RESTY_LUAROCKS_VERSION}.tar.gz \
    && yum remove -y make wget \
    && yum clean all \
    && ln -sf /dev/stdout /usr/local/openresty/nginx/logs/access.log \
    && ln -sf /dev/stderr /usr/local/openresty/nginx/logs/error.log

# Unused, present for parity with other Dockerfiles
# This makes some tooling/testing easier, as specifying a build-arg
# and not consuming it fails the build.
ARG RESTY_J="1"

# Add additional binaries into PATH for convenience
ENV PATH=$PATH:/usr/local/openresty/luajit/bin:/usr/local/openresty/nginx/sbin:/usr/local/openresty/bin

# Copy nginx configuration files
# COPY nginx.conf /usr/local/openresty/nginx/conf/nginx.conf

# CMD ["/usr/bin/openresty", "-g", "daemon off;"]


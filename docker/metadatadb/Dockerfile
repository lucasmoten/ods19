FROM centos:centos7

# Inject any MITM CA certs necessary when using this image in DI2E Jenkins
# These certs will not be baked into images developers build locally
# https://confluence.di2e.net/display/DI2E/DI2E+Proxy+Server+Certificates
RUN yum -y install ca-certificates
RUN update-ca-trust force-enable
COPY Dockerfile ./build/*.crt /etc/pki/ca-trust/source/anchors/
RUN for f in /etc/pki/ca-trust/source/anchors; do chmod 644 $f; done
COPY Dockerfile ./build/*.crt /usr/local/share/ca-certificates/
RUN for f in /usr/local/share/ca-certificates; do chmod 644 $f; done
COPY Dockerfile ./build/*.crt /usr/share/pki/ca-trust-source/anchors/
RUN for f in /usr/share/pki/ca-trust-source/anchors; do chmod 644 $f; done
RUN update-ca-trust extract

RUN yum -y install --setopt=tsflags=nodocs epel-release && \
    yum -y install --setopt=tsflags=nodocs mariadb-server bind-utils pwgen psmisc hostname && \
    yum -y erase vim-minimal && \
    yum -y update --exclude=iputils && yum clean all

VOLUME /var/lib/mysql

# Certificates used for data in transit
COPY ./build/*.pem /

# Keys and files used for data at rest
COPY mariakeys.txt /

# Binary tool
COPY ./build/odrive-database /usr/local/bin/
RUN chmod +x /usr/local/bin/odrive-database

# Configuration file
COPY my.cnf /etc/my.cnf
COPY docker-entrypoint.sh /
ENTRYPOINT ["/docker-entrypoint.sh"]

EXPOSE 3306
CMD ["mysqld_safe"]
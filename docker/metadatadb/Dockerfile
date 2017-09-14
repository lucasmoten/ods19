FROM centos:centos7

RUN yum -y install --setopt=tsflags=nodocs epel-release && \
    yum -y install --setopt=tsflags=nodocs mariadb-server bind-utils pwgen psmisc hostname && \
    yum -y erase vim-minimal && \
    yum -y update --exclude=iputils && yum clean all

VOLUME /var/lib/mysql

# Certificates used for data in transit
COPY ca.pem /
COPY client-cert.pem /
COPY client-key.pem /
COPY server-cert.pem /
COPY server-key.pem /

# Keys and files used for data at rest
COPY mariakeys.txt /

# Binary tool
COPY odrive-database /usr/local/bin/
RUN chmod +x /usr/local/bin/odrive-database

# Configuration file
COPY my.cnf /etc/my.cnf
COPY docker-entrypoint.sh /
ENTRYPOINT ["/docker-entrypoint.sh"]

EXPOSE 3306
CMD ["mysqld_safe"]
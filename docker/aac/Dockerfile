
FROM centos:6.7


RUN yum -y update

#
# Install Redis
RUN yum -y install epel-release; \
    yum -y install redis;
RUN sed -i 's/daemonize no/daemonize yes/' /etc/redis.conf 

#
# Install support tools
RUN yum -y install wget; \
    yum -y install java-1.8.0-openjdk-headless; \
    yum -y install net-tools;  \
    yum -y install tar;  \
    yum -y install which

# Redis
EXPOSE 6379

# AAC
EXPOSE 9093

# Prepare for CTE AAC
RUN useradd cte
#All files (edited) that are supplied by us
ADD build/aac-fs.tgz /
RUN chown -R cte:cte /opt/cte
RUN ln -s /opt/cte/cte-aac-service-server-1.0.0-SNAPSHOT /opt/cte/cte-aac-service-server-1.0.0
RUN ln -s /opt/cte/cte-aac-service-server-1.0.0/bin/cte-aac-service-server /etc/init.d/cte-aac-service-server-1.0.0
RUN chown cte:cte /opt/cte



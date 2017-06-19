
FROM centos:6.7


RUN yum -y update

#
# Install Redis
RUN yum -y install epel-release; \
    yum -y --nogpgcheck install redis;
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
RUN useradd aac
#All files (edited) that are supplied by us
ADD build/aac-fs.tgz /
RUN chown -R aac:aac /opt/cte
RUN ln -s /opt/cte/aac-1.1.4-SNAPSHOT /opt/cte/aac-1.1.4
RUN ln -s /opt/cte/aac-1.1.4/bin/aac /etc/init.d/aac-1.1.4
RUN chown aac:aac /opt/cte



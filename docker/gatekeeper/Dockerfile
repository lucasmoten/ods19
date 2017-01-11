FROM centos:6.7

COPY resources/fs/etc/yum.repos.d/nginx.repo /etc/yum.repos.d/

RUN yum -y update; \
    yum install -y epel-release; \
    yum install -y nginx; \
    yum -y install java-1.8.0-openjdk-headless;

#
# From inside the container: 
#   service nginx start

ADD fs.tgz /

RUN mkdir /etc/nginx/html/apps
RUN ln -s /etc/nginx/html/apps /etc/nginx/apps

RUN rpm -i `ls *.rpm`

EXPOSE 8080

FROM centos:6.7

RUN yum -y update; \
    yum -y install wget; \
    yum -y install tar; \
    yum -y install openssl-devel; \
    yum -y install sudo; \
    yum -y install rpm-build; \
    yum -y install git; \
    yum -y install gcc; \
    yum -y install yum-plugin-ovl; \
    yum -y install anacron; \
    yum -y install glibc; \
    yum -y install epel-release; \
    yum -y --disablerepo="*" --enablerepo="epel" install python34;

# Go
RUN wget https://go-boringcrypto.storage.googleapis.com/go1.12.6b4.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.12.6b4.linux-amd64.tar.gz
ENV PATH /usr/local/go/bin:$PATH
ENV GOPATH /go

# GLIBC 2.15 Gotten manually to satisfy GLIBC 2.14+ as Centos6 comes with GLIBC 2.12
RUN wget http://ftp.redsleeve.org/pub/steam/glibc-2.15-60.el6.x86_64.rpm
RUN wget http://ftp.redsleeve.org/pub/steam/glibc-common-2.15-60.el6.x86_64.rpm
RUN wget http://ftp.redsleeve.org/pub/steam/glibc-devel-2.15-60.el6.x86_64.rpm
RUN wget http://ftp.redsleeve.org/pub/steam/glibc-headers-2.15-60.el6.x86_64.rpm
RUN rpm -Uvh glibc-2.15-60.el6.x86_64.rpm glibc-common-2.15-60.el6.x86_64.rpm glibc-devel-2.15-60.el6.x86_64.rpm glibc-headers-2.15-60.el6.x86_64.rpm

RUN mkdir -p /go/src/bitbucket.di2e.net/dime/object-drive-server
COPY build /go/src/bitbucket.di2e.net/dime/object-drive-server

EXPOSE 4430

ENV ODRIVE_BINARY_DIR /go/src/bitbucket.di2e.net/dime/object-drive-server/cmd/odrive
ENV ODRIVE_ROOT /go/src/bitbucket.di2e.net/dime/object-drive-server
ENV OD_TOKENJAR_LOCATION /go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/token.jar

WORKDIR /go/src/bitbucket.di2e.net/dime/object-drive-server
RUN chmod +x build_package_install.py service_wrapper.py
RUN ./build_package_install.py
WORKDIR /go/src/bitbucket.di2e.net/dime/object-drive-server
CMD ["./service_wrapper.py"]

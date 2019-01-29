FROM centos:6.7

# For RPM build

RUN yum -y update; \
    yum -y install wget; \
    yum -y install tar; \
    yum -y install openssl-devel; \
    yum -y install sudo; \
    yum -y install rpm-build; \
    yum -y install git; \
    yum -y install epel-release; \
    yum -y install make; \
    yum -y install gcc;

# Go
RUN wget https://go-boringcrypto.storage.googleapis.com/go1.11.5b4.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.11.5b4.linux-amd64.tar.gz
ENV PATH /usr/local/go/bin:/go/bin:$PATH
ENV GOPATH /go

# GLIBC 2.14 needed for golang 1.11.5b4 compile with CGO
RUN wget http://ftp.redsleeve.org/pub/steam/glibc-2.15-60.el6.x86_64.rpm
RUN wget http://ftp.redsleeve.org/pub/steam/glibc-common-2.15-60.el6.x86_64.rpm
RUN wget http://ftp.redsleeve.org/pub/steam/glibc-devel-2.15-60.el6.x86_64.rpm
RUN wget http://ftp.redsleeve.org/pub/steam/glibc-headers-2.15-60.el6.x86_64.rpm
RUN rpm -Uvh glibc-2.15-60.el6.x86_64.rpm glibc-common-2.15-60.el6.x86_64.rpm glibc-devel-2.15-60.el6.x86_64.rpm glibc-headers-2.15-60.el6.x86_64.rpm

RUN mkdir -p /go/src/bitbucket.di2e.net/dime/object-drive-server
VOLUME /go/src/bitbucket.di2e.net/dime/object-drive-server
ENV ODRIVE_BINARY_DIR /go/src/bitbucket.di2e.net/dime/object-drive-server/cmd/odrive
ENV ODRIVE_ROOT /go/src/bitbucket.di2e.net/dime/object-drive-server
WORKDIR /go/src/bitbucket.di2e.net/dime/object-drive-server
CMD ["bash"]

FROM centos:6.7

# Inject any MITM CA certs necessary when using this image in DI2E Jenkins
# These certs will not be baked into images developers build locally
# https://confluence.di2e.net/display/DI2E/DI2E+Proxy+Server+Certificates
RUN yum -y install ca-certificates
RUN update-ca-trust force-enable
COPY Dockerfile ./build/*.crt /etc/pki/ca-trust/source/anchors/
RUN for f in /etc/pki/ca-trust/source/anchors/; do chmod 644 $f; done
RUN update-ca-trust extract

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
RUN wget https://go-boringcrypto.storage.googleapis.com/go1.12.9b4.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.12.9b4.linux-amd64.tar.gz
ENV PATH /usr/local/go/bin:/go/bin:$PATH
ENV GOPATH /go

# Update GLIBC to 2.15 to satisfy GLIBC 2.14+ required for CGO and GoBoring as Centos 6 comes with CLIBC 2.12
# These files were originally sourced from http://ftp.redsleeve.org/pub/steam
COPY ./build/glibc/glibc-2.15-60.el6.x86_64.rpm glibc-2.15-60.el6.x86_64.rpm
COPY ./build/glibc/glibc-common-2.15-60.el6.x86_64.rpm glibc-common-2.15-60.el6.x86_64.rpm
COPY ./build/glibc/glibc-devel-2.15-60.el6.x86_64.rpm glibc-devel-2.15-60.el6.x86_64.rpm
COPY ./build/glibc/glibc-headers-2.15-60.el6.x86_64.rpm glibc-headers-2.15-60.el6.x86_64.rpm
RUN rpm -Uvh glibc-2.15-60.el6.x86_64.rpm glibc-common-2.15-60.el6.x86_64.rpm glibc-devel-2.15-60.el6.x86_64.rpm glibc-headers-2.15-60.el6.x86_64.rpm

RUN mkdir -p /go/src/bitbucket.di2e.net/dime/object-drive-server
VOLUME /go/src/bitbucket.di2e.net/dime/object-drive-server

ENV ODRIVE_BINARY_DIR /go/src/bitbucket.di2e.net/dime/object-drive-server/cmd/odrive
ENV ODRIVE_ROOT /go/src/bitbucket.di2e.net/dime/object-drive-server
WORKDIR /go/src/bitbucket.di2e.net/dime/object-drive-server
CMD ["bash"]

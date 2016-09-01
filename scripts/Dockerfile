FROM centos:6.7

# For RPM build

RUN yum -y update; \
    yum -y install wget; \
    yum -y install tar; \
    yum -y install openssl-devel; \
    yum -y install sudo; \
    yum -y install rpm-build; \
    yum -y install git; \
    yum -y install gcc;

RUN wget https://storage.googleapis.com/golang/go1.6.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.6.linux-amd64.tar.gz
ENV PATH /usr/local/go/bin:$PATH
ENV GOPATH /go
RUN mkdir -p /go/src/decipher.com/object-drive-server
VOLUME /go/src/decipher.com/object-drive-server
ENV ODRIVE_BINARY_DIR /go/src/decipher.com/object-drive-server/cmd/odrive
ENV ODRIVE_ROOT /go/src/decipher.com/object-drive-server
WORKDIR /go/src/decipher.com/object-drive-server
CMD ["bash"]

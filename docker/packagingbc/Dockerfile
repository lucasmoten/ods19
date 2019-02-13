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
    yum -y install glibc;

# Go
RUN wget https://go-boringcrypto.storage.googleapis.com/go1.11.5b4.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.11.5b4.linux-amd64.tar.gz
ENV PATH /usr/local/go/bin:$PATH
ENV GOPATH /go

# GLIBC 2.14 needed for golang 1.11.5b4 compile with CGO
#RUN wget http://ftp.gnu.org/gnu/glibc/glibc-2.14.tar.gz 
#RUN mkdir -p /glibc_install && tar -C /glibc_install -xzf glibc-2.14.tar.gz && cd /glibc_install && cd glibc-2.14 && mkdir build && cd build && ../configure --prefix=/opt/glibc-2.14 && make -j4 && sudo make install
#ENV LD_LIBRARY_PATH=/opt/glibc-2.14/lib
#RUN cp /glibc_install/glibc-2.14/build/libc.so.6 /opt/glibc-2.14/lib/libc.so.6
#RUN mkdir -p /lib64
#RUN cp /glibc_install/glibc-2.14/build/libc.so.6 /lib64/libc.so.6
# GLIBC 2.15 Gotten another way...
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
RUN chmod +x build_package_install.py
RUN chmod +x service_wrapper.py
RUN ./build_package_install.py
WORKDIR /go/src/bitbucket.di2e.net/dime/object-drive-server
CMD ["./service_wrapper.py"]

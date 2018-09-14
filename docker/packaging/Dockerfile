FROM deciphernow/centos-openssl-go:1.7

RUN yum install -y anacron
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
CMD ["bash"]

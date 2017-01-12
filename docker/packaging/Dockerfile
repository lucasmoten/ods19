FROM deciphernow/centos-openssl-go:1.7

COPY build /go/src/decipher.com/object-drive-server

EXPOSE 4430

ENV ODRIVE_BINARY_DIR /go/src/decipher.com/object-drive-server/cmd/odrive
ENV ODRIVE_ROOT /go/src/decipher.com/object-drive-server
ENV OD_TOKENJAR_LOCATION /go/src/decipher.com/object-drive-server/defaultcerts/token.jar

WORKDIR /go/src/decipher.com/object-drive-server
RUN chmod +x build_package_install.py
RUN chmod +x service_wrapper.py
RUN ./build_package_install.py
WORKDIR /go/src/decipher.com/object-drive-server
CMD ["bash"]

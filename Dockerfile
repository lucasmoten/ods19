FROM ubuntu

# Dockerfile for Jenkins build. Only works on Linux environments.

RUN mkdir -p /usr/local/bin/Go/src/decipher.com/oduploader
COPY ./ /usr/local/bin/Go/src/decipher.com/oduploader
WORKDIR /usr/local/bin/Go/src/decipher.com/oduploader/cmd/metadataconnector
ENV GOPATH /usr/local/bin/Go
CMD ["/usr/local/bin/Go/src/decipher.com/oduploader/cmd/metadataconnector/metadataconnector"]


FROM ubuntu

# Dockerfile for Jenkins build. Only works on Linux environments.

RUN mkdir -p /usr/local/bin/Go/src/decipher.com/object-drive-server
COPY ./ /usr/local/bin/Go/src/decipher.com/object-drive-server
WORKDIR /usr/local/bin/Go/src/decipher.com/object-drive-server/cmd/odrive
ENV GOPATH /usr/local/bin/Go
CMD ["/usr/local/bin/Go/src/decipher.com/object-drive-server/cmd/odrive/odrive"]


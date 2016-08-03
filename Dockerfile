FROM ubuntu

# Dockerfile for Jenkins build. Only works on Linux environments.

RUN apt-get update -y
RUN apt-get install -y pkg-config libssl1.0.0 libssl-dev
RUN apt-get install -y ca-certificates
RUN mkdir -p /usr/local/bin/Go/src/decipher.com/object-drive-server
COPY ./ /usr/local/bin/Go/src/decipher.com/object-drive-server
WORKDIR /usr/local/bin/Go/src/decipher.com/object-drive-server/cmd/odrive
ENV GOPATH /usr/local/bin/Go
CMD ["/usr/local/bin/Go/src/decipher.com/object-drive-server/cmd/odrive/odrive"]

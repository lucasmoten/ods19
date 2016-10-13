FROM ubuntu

# Dockerfile for CI build. Only works on Linux environments.

RUN apt-get update -y
RUN apt-get install -y pkg-config libssl1.0.0 libssl-dev
RUN apt-get install -y ca-certificates
RUN mkdir -p /go/src/decipher.com/object-drive-server
COPY ./ /go/src/decipher.com/object-drive-server
WORKDIR /go/src/decipher.com/object-drive-server/cmd/odrive
ENV GOPATH /go
CMD ["/go/src/decipher.com/object-drive-server/cmd/odrive/odrive"]

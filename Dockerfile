FROM ubuntu

# Dockerfile for CI build. Only works on Linux environments.

RUN apt-get update -y
RUN apt-get install -y pkg-config libssl1.0.0 libssl-dev
RUN apt-get install -y ca-certificates
RUN mkdir -p /go/src/github.com/deciphernow/object-drive-server
COPY ./ /go/src/github.com/deciphernow/object-drive-server
WORKDIR /go/src/github.com/deciphernow/object-drive-server/cmd/odrive
ENV GOPATH /go
CMD ["/go/src/github.com/deciphernow/object-drive-server/cmd/odrive/odrive"]

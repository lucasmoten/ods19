FROM deciphernow/centos-openssl-go:1.7

ENV OD_TOKENJAR_LOCATION /go/src/github.com/deciphernow/object-drive-server/defaultcerts/token.jar

COPY build /go/src/github.com/deciphernow/object-drive-server
EXPOSE 4430

WORKDIR /go/src/github.com/deciphernow/object-drive-server/cmd/odrive
RUN go build -race -o main .
CMD ["./main", "--staticRoot", "/go/src/github.com/deciphernow/object-drive-server/server/static", "--templateDir", "/go/src/github.com/deciphernow/object-drive-server/server/static/templates"]

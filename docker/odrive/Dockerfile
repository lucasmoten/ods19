FROM deciphernow/centos-openssl-go:1.9.4

ENV OD_TOKENJAR_LOCATION /go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/token.jar

COPY build /go/src/bitbucket.di2e.net/dime/object-drive-server
EXPOSE 4430

WORKDIR /go/src/bitbucket.di2e.net/dime/object-drive-server/cmd/odrive
RUN go build -race -o main .
CMD ["./main", "--staticRoot", "/go/src/bitbucket.di2e.net/dime/object-drive-server/server/static", "--templateDir", "/go/src/bitbucket.di2e.net/dime/object-drive-server/server/static/templates"]

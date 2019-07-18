FROM library/golang:1.12.9
ARG BUILD_NUMBER="1"
ARG COMMIT_ID="1"
ARG BUILD_VERSION="1"
ENV OD_TOKENJAR_LOCATION /go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/token.jar

COPY build /go/src/bitbucket.di2e.net/dime/object-drive-server
EXPOSE 4430

WORKDIR /go/src/bitbucket.di2e.net/dime/object-drive-server/cmd/odrive
# -race requires CGO here
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -race -ldflags "-X main.Build=$BUILD_NUMBER -X main.Commit=$COMMIT_ID -X main.Version=$BUILD_VERSION" -o main .
CMD ["./main", "--conf", "./odrive.yml", "--staticRoot", "/go/src/bitbucket.di2e.net/dime/object-drive-server/server/static", "--templateDir", "/go/src/bitbucket.di2e.net/dime/object-drive-server/server/static/templates"]

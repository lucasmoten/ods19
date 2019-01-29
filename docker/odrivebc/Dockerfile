FROM goboring/golang:1.11.5b4
ARG go_build_ldflags='-X main.Build=1 -X main.Commit=1 -X main.Version=1'
ENV OD_TOKENJAR_LOCATION /go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/token.jar

COPY build /go/src/bitbucket.di2e.net/dime/object-drive-server
EXPOSE 4430

WORKDIR /go/src/bitbucket.di2e.net/dime/object-drive-server/cmd/odrive
# CGO must be enabled for boring crypto as well as -race
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -race -ldflags "${go_build_ldflags}" -o main .
CMD ["./main", "--staticRoot", "/go/src/bitbucket.di2e.net/dime/object-drive-server/server/static", "--templateDir", "/go/src/bitbucket.di2e.net/dime/object-drive-server/server/static/templates"]

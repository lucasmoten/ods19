FROM deciphernow/centos-openssl-go:1.7

ENV OD_TOKENJAR_LOCATION /go/src/decipher.com/object-drive-server/defaultcerts/token.jar

COPY build /go/src/decipher.com/object-drive-server
EXPOSE 4430

WORKDIR /go/src/decipher.com/object-drive-server/cmd/odrive
RUN go build -race -o main .
CMD ["./main"]

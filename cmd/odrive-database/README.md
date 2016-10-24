# odrive-database

This cli binary is distributed with odrive to make setting up databases easier.

Build the code like this


```bash
go-bindata migrations schema ../../defaultcerts/client-mysql/id ../../defaultcerts/client-mysql/trust
go build   # set GOOS=linux and GOARCH=amd64 to cross compile
```

After the binary is built, get help like this

```
odrive-database help
```

Default configurations for docker containers are hardcoded into the binary. During local development,
provide the `useEmbedded=true` flag to use embedded credentials.

```
odrive-database status --useEmbedded=true
```

To connect to another database you must use valid object-drive-server configs. 

```
sudo su
source /opt/odrive/env.sh
odrive-database status
```

Alternatively, provide a path to a valid yaml file with the `conf` parameter.

```
odrive-database status --conf=foo.yml
```




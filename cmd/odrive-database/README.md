# odrive-database

This cli binary is distributed with odrive to make setting up databases easier.

Build the code like this


```
go-bindata migrations schema ../../defaultcerts/client-mysql/id ../../defaultcerts/client-mysql/trust
go build
```

After the binary is built, get help like this

```
./odrive-database help
```

Default configurations for docker containers are hardcoded into the binary, but
to connect to another database you must use valid object-drive-server configs.




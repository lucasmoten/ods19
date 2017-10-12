# Object Drive Server

# API Documentation

API documentation for the Object Drive service may be reviewed at the root of an instantiated object-drive server,
previewed [here](./docs/home.md), or accessed from this [live instance on Bedrock](https://bedrock.363-283.io/services/object-drive/1.0/)


# Configuration

Detailed here: https://gitlab.363-283.io/cte/object-drive/wikis/object-drive-environment-variables

See also the example docker-compose file **.ci/docker-compose.yml** for example environment variables.
Note that some vars are not set directly inline, because they contain secrets (e.g. AWS vars).

# Clone this repository

All dependent Go code is relative to the **GOPATH**. Create the the directory **$GOPATH/src/decipher.com**
and clone this project there. This will allow imports like this to resolve correctly.

```go
import "decipher.com/object-drive-server/somepackage"
```

# Openssl bindings dependency

This project depends on OpenSSL, and binds to C code (uses CGO). This means you
may need to set the `PKG_CONFIG_PATH` variable. This can vary by distribution.

If you're using brew on a Mac, you might set it like this:

```bash
export PKG_CONFIG_PATH="$(brew --prefix openssl)/lib/pkgconfig"
```

On Ubuntu, it might be

```bash
export PKG_CONFIG_PATH="/usr/lib/x86_64-linux-gnu/pkgconfig"
```

# Setting up your development environment

Developing on this project requires maven, docker, and nodejs configurations.
Also, a separate build "root" directory must be specified by setting the `OD_ROOT`
environment variable. The build script will check out and build other dependencies
there. Consider this a volatile directory. A fine location to set would be

```
export OD_ROOT=$HOME/my_code/od_root
```

After that is done, run

```
./odb build
```

`odb` is a python script that builds binaries and docker containers for this
project and its dependencies. It also inspects your build environment, and
notifies you when tools are missing.

This project requires edits to your **/etc/hosts** file. These are the most
common settings:

```
127.0.0.1 localhost dockervm fqdn.for.metadatadb.local gatekeeper metadatadb aac metadataconnector zk pk ui builder kafka twl-server-generic2 gateway metadatadb
```

Tests can be run locally if the suite of containers defined in **docker/docker-compose.yml**
are built and running. Run `go test ./...` from the root of this project.

# Vendoring

We are using a vendoring tool called `govendor` to pin our dependencies to a specific commit.

```
go get github.com/kardianos/govendor
go install github.com/kardianos/govendor
```

The **govendor** tool should now be in $GOPATH/bin. Make sure that is on your PATH.
Sync the dependencies to the local **/vendor** folder like this:

```
govendor sync
```
Note that when you do this, the vendor/ directory will have a vendor.json file, and a bunch of directories for repos.
Sometimes it is necessary to delete all of the directories under vendor/ and re-run `govendor sync` to get `go build ./...`
to build with a consistent source tree.

# Other Configuration

Binaries for the main server are built under **cmd/odrive**.

# odutil

Another tool is compiled under **cmd/odutil**. Currently it can upload and
download files from S3. AWS credentials are taken from the environment.

Upload

```
odutil -cmd upload -input somefile.txt -bucket decipher-tools -key some/path/somefile.txt
```

Download

```
odutil -cmd download -input somefile.txt -bucket decipher-tools -key some/path/somefile.txt
```

Generating current docs (no longer checked in):

```
./makedocs
```

Making an rpm (will build docs as well):

```
cd $GOPATH/src/decipher.com/object-drive-server
#make an rpm as version 1.0.9 and call it build number 2600.  It will be in current directory when done
./makerpm 1.0.9 2600
```

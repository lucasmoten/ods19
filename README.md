# Object Drive Server

# API Documentation

API documentation may be reviewed at the root of an instantiated object-drive server,
previewed [here](./docs/home.md), or accessed from this [live instance on Bedrock](https://bedrock.363-283.io/services/object-drive/1.0/)

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

# Configuration

Detailed here: https://gitlab.363-283.io/cte/object-drive/wikis/object-drive-environment-variables

See also the example docker-compose file **.ci/docker-compose.yml** for example environment variables.
Note that some vars are not set directly inline, because they contain secrets (e.g. AWS vars).

# Hosting The Code

All dependent Go code is relative to the **GOPATH**. Create the the directory **$GOPATH/src/decipher.com**
and clone this project there. This will allow imports like this to resolve correctly.

```go
import "decipher.com/object-drive-server/somepackage"
```

# Openssl bindings dependency 

Due to internal openssl use, `pkg-config` must be setup for the go code to compile. See our Dockerfiles 
show exactly how this is done on the different Linux distributions.  On OSX (ElCapitan specifically) 
more can go wrong, in addition to installing `pkg-config` for openssl, you may need to help the system 
to find the proper packages with this set in your .bash_profile (when you get an inability to find "bios.h" 
during `go build ./...`) 

```bash
export PKG_CONFIG_PATH="$(brew --prefix openssl)/lib/pkgconfig"
```

Note that $OD_ROOT is where `cte/object-drive` is checked out.

> The cte/object-drive project pulls together the environment that cte/object-drive-server executes in.  Refer to that project to get all of the dependencies setup to actually execute odrive (npm, gulp, proper hub.docker.com login, archiva setup, and so on).   

```
$OD_ROOT/object-drive
```

Metadataconnector Browser:

* Make sure that you set these environment variables:
  * OD_AWS_REGION=us-east-1
  * OD_AWS_ACCESS_KEY_ID
  * OD_AWS_SECRET_KEY
  * OD_ZK_URL=zk_1:2181,zk_2:2181,zk_3:2181

# Checking out and building

You should be able to build the source like this.

```
$ git clone ssh://git@gitlab.363-283.io:2252/cte/object-drive-server.git $GOPATH/src/decipher.com
$ cd $GOPATH/src/decipher.com
```

This invokes the Python build script that fetches dependencies, builds binaries,
and exports required certificates.

# Generating Thrift Code

Once you have the latest version of the go-thrift library installed, put it's
**generator** binary on your PATH. Then run the top-level thrift Service IDL
file through the generator.

Install the Thrift code generator with:

```
go install github.com/samuel/go-thrift/generator
```

Example (from within /services/foo/thrift):

```
generator -go.signedbytes=true Foo.thrift ../generated
```

# Running Tests

Run **every** test in the project with a `./...` recursive walk.

```
cd $GOPATH/src/decipher.com/object-drive-server
go test ./... -v
```

Only run short tests by specifying `-short=true`:

```
go test ./... -short=true -v
```

Hooray for automated tests!


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
#make an rpm and call it build number 2600.  It will be in current directory when done
./makerpm 2600
```



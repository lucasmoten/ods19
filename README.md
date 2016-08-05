# Object Drive Server

# API Documentation

API documentation is hosted on the private Bedrock network here (subject to change):

https://bedrock.363-283.io/services/object-drive/1.0/

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

# Hosting The Code

Detailed here: https://gitlab.363-283.io/cte/object-drive/wikis/object-drive-environment-variables

All dependent Go code is relative to the **GOPATH**. If the source tree on your
disk looks like this:

```
$GOPATH/
  bin
  src/
    decipher.com/
      object-drive-server/
        somepackage/
```

...Import statements for `somepackage` will look like this in Go:

```go
import "decipher.com/object-drive-server/somepackage"
```

The other code (Java, etc) should be found at **OD_ROOT**. Python build scripts
in the `cte/object-drive` project checkout and compile the code under **OD_ROOT**

Note that $OD_ROOT is where `cte/object-drive` is checked out.
Both directories ($GOPATH $OD_ROOT) allow compile and build steps
to reference each other.

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

Example (from within /services/audit/thrift):

```
generator -go.signedbytes=true AuditService.thrift ../generated
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


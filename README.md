# Large File Uploader

This is an encrypted file storage API with a REST interface.

# API documentation

API documentation is hosted on the private Bedrock network here (subject to change):

http://10.2.11.150:8000/

# Project Management

Issues are tracked internally in this [Google Doc](https://docs.google.com/spreadsheets/d/1Eiuu8uH6O6_uPtz6icOgLof3JYExhPDo9RelJDFsDeA/edit#gid=538633894)

# Vendoring

We are using a vendoring tool called `govendor` to pin our dependencies to a specific commit.

```
#get the vendoring tool
go get github.com/kardianos/govendor
cd $GOPATH/src/github.com/kardianos/govendor
go install

# now govendor is in $GOPATH/bin, which should be in your path along with $GOROOT/bin
#sync up all dependencies
cd $GOPATH/src/decipher.com/object-drive-server
govendor sync
```

# Hosting The Code

Required environment variables:
* **OD_ROOT** the directory to check out non-Go source dependencies into, including
  the `cte/object-drive` repository itself.
* **GOPATH** the Go source tree. `object-drive-server` will be checked out to
  a path in this tree.
* **AWS_REGION=us-east-1**  (or your region)
* **AWS_ACCESS_KEY_ID**  get credentials from your system administrator
* **AWS_SECRET_KEY** get credentials from your system administrator

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
  * AWS_REGION=us-east-1
  * AWS_ACCESS_KEY_ID
  * AWS_SECRET_KEY
  * ZKURL=zk_1:2181,zk_2:2181,zk_3:2181

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

Binaries for the main server are built under **/cmd/metadataconnector**. By default,
the main configuration is read from a **conf.json** from the same directory.



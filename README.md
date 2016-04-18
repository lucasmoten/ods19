# Large File Uploader

This is an encrypted file storage API with a REST interface.

# Project Management

Issues are tracked internally in this [Google Doc](https://docs.google.com/spreadsheets/d/1Eiuu8uH6O6_uPtz6icOgLof3JYExhPDo9RelJDFsDeA/edit#gid=538633894)

# Vendoring

We are now moved up to Go1.6, so that vendoring is transparent.  We are using a vendoring tool.

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

When you checkout Go code, Go does not like to have a path.
It uses a consistent directory structure instead.
If you set $GOPATH to point to a location of your go code,
go code needs to be hosted in the tree:

```
$GOPATH/
  bin
  src/
    decipher.com/
      object-drive-server/
```

Checkouts from gitlab are cloned into decipher.com,
which allows for cross-references between packages.
Note that some things will not build until package are retrieved.
AWS packages:  github.com/aws-sdk-go is retrieved with
go get (rather than a manual clone).

The other code (Java, etc) should be found at $OD_ROOT, which
is the directory into which we check out the object-drive project.
Having these conventions is essential, because we have references
across repositories to built artifacts and static files.

Note that $OD_ROOT is where object-drive is checked out.
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

# Checking out and building

You should be able to build the source like this.

```
$ git clone ssh://git@gitlab.363-283.io:2252/rob.fielding/object-drive-server.git $GOPATH/src/decipher.com
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


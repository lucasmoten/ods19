# Large File Uploader

This is a project to make a simple Go uploader that can deal
with the largest files, by using multipart mime protocol
correctly (almost nothing does).  This means that buffering
in memory is bounded.

It is possible to have a hybrid that keeps a reasonable amount of
data in memory without ever writing it to disk, and flushes
to disk when a session is going to begin to use unfair amounts
of memory


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
      oduploader/
```


We use ~/gocode as my $GOPATH.
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
  * AWS_ACCESS_KEY
  * AWS_SECRET_ACCESS_KEY

Cryptotest Browser (deprecated):

* use a consistent master key to launch it:
  - masterkey=djklerwjkl23 go run uploader.go
* https://localhost:6445/upload   (pick some file, like foo.txt)

# Checking out and building

You should be able to build the source like this.

```
$ git clone ssh://git@gitlab.363-283.io:2252/rob.fielding/oduploader.git $GOPATH/src/decipher.com
$ cd $GOPATH/src/decipher.com
$ ./build
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
go test ./... -v
```

Only run short tests by specifying `-short=true`:

```
go test ./... -short=true -v
```

Hooray for automated tests!

#Automated Uploading and Downloading

By default, cmd/autopilot will look in $AUTOPILOT_HOME which defaults to ~/autopilot if it is not specified.
It will then look through uploadCache_testn directories and randomly upload files.
It does not yet implement randomly downloading files (which means getting directory listings for random users).
It will also need to specify random clearances on files at some point, using data from rmt.zip that we got from Jon.
```
Robs-MacBook-Pro:docker rfielding$ ls ~/autopilot/
downloadCachetest_0	downloadCachetest_4	downloadCachetest_8	uploadCachetest_2	uploadCachetest_6
downloadCachetest_1	downloadCachetest_5	downloadCachetest_9	uploadCachetest_3	uploadCachetest_7
downloadCachetest_2	downloadCachetest_6	uploadCachetest_0	uploadCachetest_4	uploadCachetest_8
downloadCachetest_3	downloadCachetest_7	uploadCachetest_1	uploadCachetest_5	uploadCachetest_9
Robs-MacBook-Pro:docker rfielding$ 
```

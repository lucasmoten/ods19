#Large File Uploader

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
Aws packages:  github.com/aws-sdk-go is retrieved with
go get (rather than a manual clone).

The other code (Java, etc) should be found at $OD_ROOT, which
is the directory into which we check out the object-drive project.
Having these conventions is essential, because we have references
across repositories to built artifacts and static files.

Browser:

* use a consistent master key to launch it:
  - masterkey=djklerwjkl23 go run uploader.go
* https://localhost:6445/upload   (pick some file, like foo.txt)


# Cross compiling

From the root directory, run:

```
$ ./scripts/cross-compile.sh
```

The tar files for multiple system binaries should be available in the
`/release` directory.


# Cross compiling

From the root directory, run:

```
$ ./scripts/cross-compile.sh
```

The tar files for multiple system binaries should be available in the
`/release` directory.

TODO:

Because this is might be an auxillary service to a different service,
we should probably behave like Amazon and require signed URLs,
or signed cookies that give permission to do things such as perform
PUT/GET operations within a short timeframe.

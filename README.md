#Large File Uploader

This is a project to make a simple Go uploader that can deal
with the largest files, by using multipart mime protocol
correctly (almost nothing does).  This means that buffering
in memory is bounded.

It is possible to have a hybrid that keeps a reasonable amount of
data in memory without ever writing it to disk, and flushes
to disk when a session is going to begin to use unfair amounts
of memory



Browser:

* run ./gencerts so that the SSL server can launch
* run ./generateuser $USER to make a client cert signed by the server cert
* go run uploader.go
* by default it uses /tmp/uploader, a directory that should exist
* https://localhost:6060/upload   (pick some file, like foo.txt)
* https://localhost:6060/download/foo.txt  (assuming you uploaded it)


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

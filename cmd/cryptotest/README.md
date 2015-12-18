# Building

This will generate Thrift artifacts

```bash
  ./build
```

After that, when in development, you can run uploader by passing in a password to scramble uploader data:

```bash
masterkey=D1sPassW3rD go run uploader.go
```

If you navigate a browser to:

[http://localhost:6443/upload]

You will have a place to upload files.  They are individually encrypted with random keys, with a masterkey
that you need to remember to download these files again.
If you run multiple instances of it, or want to change the location of the buckets it writes data into,
there are options for just about everything available:

```bash
masterkey=D1sPassW3rD go run uploader.go -tcpPort 8444 -homeBucket bucket8444
```


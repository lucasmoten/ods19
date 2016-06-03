
# odutil

A tool for putting source code (or anything) into AWS S3.


# Uploading

Uploading is the default command.

```
./odutil  -input test-mpeg_512kb.mp4 -bucket odrive-builds
```

# Downloading

Downloading requires that you specify the `-cmd` flag

```
./odutil -cmd download -input test-mpeg_512kb.mp4 -bucket odrive-builds
```



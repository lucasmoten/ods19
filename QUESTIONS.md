
# Code Review and Questions re: Rob's Prototype

## Main routine

* First we make a server with `makeServer`
  * Make a backend and set it on the server/uploader
  * ensure "partition" exists. This is the "bucket".
  * set the listing address/port on the server
  * spin off a `statsRoutine`, pass in channels of required statistics
* back in `RunIt`, read cert files, create cert pool, and set up TLS config
* listen and serve with the server

* A POST to `/upload` calls serveHTTPUploadPOST, which gets a multipartReader from the request and starts looping
* Pull classification out of request, checks authorization (fakes this right now)
* calls `serveHTTPUploadPOSTDrain`
  * creates new statistic
  * gets a "write handle"
  * gets DN (obfuscated)
  * ensures partition exists (faked by making a directory)
  * creates a key + initialization vector pair with `createKeyIVPair`
    * make a key (a `[]byte`) of randomness
    * make an iv (a `[]byte`) of randomness and length of aes.BlockSize
    * 0 out last four bytes of iv
  


## aws.go

* Can `awsGetReadHandle` return a `ReadCloser` instead? https://golang.org/pkg/io/#ReadCloser
* Can `awsGetWriteHandle` return a `WriteCloser`?
* Should `awsEnsurePartitionExists` be moved to the initializer function, `NewAWSBackend`?
* Why all the calls to `drainFileToS3` in `drainToS3`?

## general.go

* Can the `Backend` type be an interface? Should the function sigs it contains be interfaces?


## web.go

* a `NewServer` function is a more idiomatic name than `makeServer`
* We will need to design for pushing stats to Prometheus (framework)

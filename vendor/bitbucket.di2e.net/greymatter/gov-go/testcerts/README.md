# Test Certificates

This `testcerts` package contains certificates, packaged as gocode,
which can be easily imported into existing packages for testing
purposes.

These certs will only be useful to applications that talk to SOME of our 
government clients. If that doesn't apply to your code, do not use these.

To re-bundle this certificates, after an addition or other modification,
run the following:

```bash
go-bindata -o testcerts/bindata.go -pkg testcerts testcerts/
```


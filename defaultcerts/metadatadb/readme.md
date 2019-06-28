# metadata Certificate files

These files can be referenced by Object Drive Server in development when connecting to MySQL. They were generated self-signed and got baked into the metadatadb docker image. 

The common name (CN) on the subject is specifically set to match the fully-qualified-domain-name for the metadata database container and depends on setting /etc/hosts entries when leveraging the security checks.

## id/server-cert.pem
```
Issuer: C = US, ST = VA, L = Arlington, O = Decipher, OU = Object-Drive, CN = Certificate Authority
Validity
    Not Before: Jan 29 21:17:52 2016 GMT
    Not After : Jan 26 21:17:52 2026 GMT
Subject: C = US, ST = VA, L = Arlington, O = Decipher, OU = Object-Drive, CN = fqdn.for.metadatadb.local
```
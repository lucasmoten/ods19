
FORMAT: 1A

# Object Drive

<table style="width:100%;border:0px;padding:0px;border-spacing:0;border-collapse:collapse;font-family:Helvetica;font-size:10pt;vertical-align:center;"><tbody><tr><td style="padding:0px;font-size:10pt;">Version</td><td style="padding:0px;font-size:10pt;">--Version--</td><td style="width:20%;font-size:8pt;"> </td><td style="padding:0px;font-size:10pt;">Build</td><td style="padding:0px;font-size:10pt;">--BuildNumber--</td><td style="width:20%;font-size:8pt;"></td><td style="padding:0px;font-size:10pt;">Date</td><td style="padding:0px;font-size:10pt;">--BuildDate--</td></tr></tbody></table>

# Group Navigation

## Table of Contents

+ [Service Overview](../../)
+ [RESTful API documentation](rest.html)
+ [Emitted Events documentation](events.html)
+ [Environment](environment.html)
+ [Changelog](changelog.html)
+ [BoringCrypto](boringcrypto.html)

# Group Boring Crypto

This document provides insight into the integration of Boring Crypto in this service that took place in late 2018.

## Background

As a service used for government purposes, the Federal Information Processing Standards (FIPS)
apply. Of key consideration is the use of cryptographic algorithms.  

Object Drive is written in the Go programming language which was designed by engineers at Google in 2007 with a 1.0 release in 2012. In 2017, Google submitted the __BoringCrypto__ core of BoringSSL through the National Institute of Standards & Technology (NIST) [Cryptographic
Module Validation Program](https://csrc.nist.gov/Projects/cryptographic-module-validation-program/Standards). 

* [dev.boringcrypto branch readme](https://go.googlesource.com/go/+/refs/heads/dev.boringcrypto.go1.12/README.boringcrypto.md)
* [JIRA Issue for Object Drive and Boring Crypto](https://jira.di2e.net/browse/DIMEODS-1144)
* [Go+BoringCrypto](https://go.googlesource.com/go/+/refs/heads/dev.boringcrypto.go1.12/misc/boring/)

Releases of Object Drive starting with 1.0.18 are built using a version of Go that includes the BoringCrypto module to satisfy FIPS 140-2.  This is signified in the same way that the Go does with a suffix of `b` followed by release number (e.g. b4 represented as 1.0.18b4).


## Using Go+BoringCrypto

Go+BoringCrypto modifies the crypto package of Go programming language to check whether a flag is enabled, and if so, runs a block of code that references the validated module in place of the native code.

The latest versions of Go+BoringCrypto are acquired from the S3 go-boringcrypto bucket whose contents can be listed at https://go-boringcrypto.storage.googleapis.com/

As of this writing, the current version of Go+BoringCrypto used by the project is go1.12.6b4

Building projects with Go+BoringCrypto is performed the same way as that when using Go Native.

### Important caveats

- You must *not* enable `pure` mode, since cgo must be enabled. To ensure that binaries are linked with BoringCrypto, you can set `pure = "off"` on all relevant `go_binary` rules.
- The build must be GOOS=linux, GOARCH=amd64.
- The build must have cgo enabled.
- The android build tag must not be specified.
- The cmd_go_bootstrap build tag must not be specified.

### Verification

The version string reported by `runtime.Version` does not indicate that BoringCrypto
was actually used for the build. For example, linux/386 and non-cgo linux/amd64 binaries
may report a version of `go1.8.3b2` but not be using BoringCrypto.

To check whether a given binary is using BoringCrypto, run `go tool nm` on it and check
that it has symbols named `*_Cfunc__goboringcrypto_*`.

### GLIBC Dependency

Because CGO must be enabled, this places a dependency on the GLIBC libraries. The Go+BoringCrypto releases were compiled on a system that requires GLIBC 2.14 or higher.  For older systems like Centos 6 that come with GLIBC 2.12, you may not be able to upgrade GLIBC through regular processes. Instead, the following has been confirmed to work with the docker images that are built in this project that test RPM installations that check for the dependency

```bash
wget http://ftp.redsleeve.org/pub/steam/glibc-2.15-60.el6.x86_64.rpm
wget http://ftp.redsleeve.org/pub/steam/glibc-common-2.15-60.el6.x86_64.rpm
wget http://ftp.redsleeve.org/pub/steam/glibc-devel-2.15-60.el6.x86_64.rpm
wget http://ftp.redsleeve.org/pub/steam/glibc-headers-2.15-60.el6.x86_64.rpm
rpm -Uvh glibc-2.15-60.el6.x86_64.rpm glibc-common-2.15-60.el6.x86_64.rpm glibc-devel-2.15-60.el6.x86_64.rpm glibc-headers-2.15-60.el6.x86_64.rpm
```

## Script From Scratch

The steps taken to install the tools when building everything from scratch
follows the steps that are outlined in the [PDF](https://csrc.nist.gov/CSRC/media/projects/cryptographic-module-validation-program/documents/security-policies/140sp2964.pdf) prepared by Google Inc. on
July 18, 2017 for BoringCrypto FIPS 140-2 Security Policy, Software Version
24e5886c0edfc409c8083d10f9f1120111efd6f5

The following script is an adaptation of that guidance

```bash
mkdir -p boringcryptotools
cd boringcryptotools

# dependencies
sudo apt-get install cmake xz-utils re2c ninja-build

# clang 4.0.0
wget http://releases.llvm.org/4.0.0/hans-gpg-key.asc
gpg --import hans-gpg-key.asc
wget http://releases.llvm.org/4.0.0/llvm-4.0.0.src.tar.xz
wget http://releases.llvm.org/4.0.0/llvm-4.0.0.src.tar.xz.sig
gpg --verify llvm-4.0.0.src.tar.xz.sig llvm-4.0.0.src.tar.xz
tar -xf llvm-4.0.0.src.tar.xz
mkdir -p llvm-4.0.0.src/tools/clang
wget http://releases.llvm.org/4.0.0/cfe-4.0.0.src.tar.xz
wget http://releases.llvm.org/4.0.0/cfe-4.0.0.src.tar.xz.sig
gpg --verify cfe-4.0.0.src.tar.xz.sig cfe-4.0.0.src.tar.xz
tar -xf cfe-4.0.0.src.tar.xz
mv -t llvm-4.0.0.src/tools/clang cfe-4.0.0.src/*
mv -t llvm-4.0.0.src/tools/clang cfe-4.0.0.src/.*
rm -rf cfe-4.0.0.src
mkdir -p mybuilddir
cd mybuilddir
cmake ../llvm-4.0.0.src
cmake --build .
cd bin
./clang -v
./clang++ -v
cd ..
cd ..

# go dev.boringcrypto.go1.12 
#    (once downloaded, boringcrypto bits built in src/crypto/internal/boring/build/build.sh)
#    this wgets the same boringssl as below and checks the hash
#    however.... the runtime.version ends up being set from the commit hash like so:   
#        devel +2e2a04a605 Mon Sep 24 21:19:42 2018 -0400
#    which is from the commit info. 
#    https://go.googlesource.com/go/+/dev.boringcrypto/ is authoritative for source
#      Within src/misc/boring/build.release is how it packages and publishes to the
#      go-boringcrypto.storage.googleapis.com location
git clone --single-branch -b dev.boringcrypto.go1.12 git@github.com:golang/go.git
cd go/src
./all.bash
which go
go version
cd ../bin 
./go version
cd ..
cd ..

# ninja 1.7.2
wget https://github.com/ninja-build/ninja/archive/v1.7.2.tar.gz
tar -xf v1.7.2.tar.gz
cd ninja-1.7.2
./configure.py --bootstrap
./ninja --version
cp ninja ../mybuilddir/bin
cd ..

# toolchain
cd mybuilddir/bin
#printf "set(CMAKE_C_COMPILER \""`pwd`"/clang\")\nset(CMAKE_CXX_COMPILER \""`pwd`"/clang++\")\nset(CMAKE_MAKE_PROGRAM \""`pwd`"/ninja\")\n" >> ${HOME}/toolchain
printf "set(CMAKE_C_COMPILER \""`pwd`"/clang\")\nset(CMAKE_CXX_COMPILER \""`pwd`"/clang++\")\n" >> ${HOME}/toolchain
chmod +x ${HOME}/toolchain
${HOME}/toolchain
cd ..
cd ..

# boringssl on its own
wget https://commondatastorage.googleapis.com/chromium-boringssl-docs/fips/boringssl-24e5886c0edfc409c8083d10f9f1120111efd6f5.tar.xz
sha256sum boringssl-24e5886c0edfc409c8083d10f9f1120111efd6f5.tar.xz | head -c 64
if [ "$(sha256sum boringssl-24e5886c0edfc409c8083d10f9f1120111efd6f5.tar.xz | head -C 64)" = "15a65d676eeae27618e231183a1ce9804fc9c91bcc3abf5f6ca35216c02bf4da" ]; then
    echo "boringssl sha256sum matched expected value"
else
    echo "ERROR: Hash of boringssl does not match expected value"
    exit 1
fi
tar -xf boringssl-24e5886c0edfc409c8083d10f9f1120111efd6f5.tar.xz
cd boringssl
mkdir build && cd build && cmake -GNinja -DCMAKE_TOOLCHAIN_FILE=${HOME}/toolchain -DFIPS=1 -DCMAKE_BUILD_TYPE=Release ..
ninja
ninja run_tests
```


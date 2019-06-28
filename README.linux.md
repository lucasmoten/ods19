# Developers Enhancing Object Drive -- Linux Setup

## Install Go

Make sure `go` is installed

At the time of writing, the latest available version was 1.12.6

Go may be downloaded from https://golang.org/dl/

Go with Boring-Crypto https://go-boringcrypto.storage.googleapis.com/

```bash
wget https://go-boringcrypto.storage.googleapis.com/go1.12.6b4.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.12.6b4.linux-amd64.tar.gz
```

## Set GOPATH

Set up your "go path". Go is generally centered around a single workspace for *everything*.

This is usually at `~/go`. It is then broken down into `~/go/src/` for all of the source code which is where all of the work actually happens, `~/go/bin/` for the global binaries, and `~/go/pkg/` for global dependencies.

You will need to set up an environment variable called `GOPATH` to point to this directory and it will save you headaches later if you the binaries directory to your path.

In your `~/.bash_profile` add:

```bash
# setup Go
export GOPATH="$HOME/go"
export PATH=$PATH:$GOPATH/bin
```

In `$GOPATH`, the `bin` and `pkg` directories mostly handle themselves. However, `src` is highly structured by where the code comes from. 

So if the code was downloaded from bitbucket.di2e.net, it would go into the `$GOPATH/src/bitbucket.di2e.net/` directory

If it came from golang.org it would go in the `$GOPATH/src/golang.org/` directory. 

This structure makes it easier for dependency managers like `govender` or `dep` to manage the packages.

So the overall setup of the go workspace would look something like this:
```bash
$GOPATH                  <-- You set this environment variable (usually ~/go/)
├── bin                  <-- Executables go here
├── pkg                  <-- Object files (*.a) go here
└── src                  <-- Source files go here. This is where the work happens
    └── bitbucket.di2e.net  <-- Folder containing source code from the BitBucket DI2E repos
        └── dime            <-- Folder containing source code from the dime orginization
    └── github.com          <-- Folder containing source code from the GitHub repos
        └── deciphernow     <-- Folder containing source code from the deciphernow orginization
    └── golang.org          <-- Folder containing source code from golang.org
```

## Install Additional Tooling

The sections below will provide guidance for installing each of the following

- protobuf compiler
- openssl bindings
- python
- pip
- docker
- docker-compose
- aws cli
- go-bindata
- govendor


### The protobuf compiler

Almost all of the Decipher microservices use [protobuf](https://developers.google.com/protocol-buffers/) which is *a language-neutral, platform-neutral extensible mechanism for serializing structured data*. 

Download the installer by going to https://github.com/google/protobuf/releases and choosing the release that suits your needs.

Once installed, modify your path to include the binaries from protoc.

An example is 

```bash
PATH=$PATH:~/Developer/protoc-3.3.0-osx-x86_64/bin
```

### Openssl

This project depends on OpenSSL, and binds to C code (uses CGO).

This means you may need to set the `PKG_CONFIG_PATH` variable. This can vary by distribution.

Object Drive uses the openssl C bindings, so add it to the correct environment variable.

```bash
export PKG_CONFIG_PATH="/usr/lib/x86_64-linux-gnu/pkgconfig"
```

### Python

The scripts depend on python 2.7 or 3.4+ 

Using Ubuntu, this may be installed in Linux via

```bash
sudo apt-get update
sudo apt-get install python2.7 
sudo apt-get install python-pip
```

### Dot and Graphviz

Building documentation requires dot to be installed. 

Install in Ubuntu Linux via

```bash
sudo apt-get update
sudo apt-get install graphviz
```

### Docker and Docker Compose

Building documentation and container images requires docker to be installed. 

Follow the guidance for your platform from the [Docker page](https://docs.docker.com/install/linux/docker-ce/ubuntu/#install-using-the-repository)

In addition, install docker-compose in Ubuntu Linux via

```bash
sudo apt-get update
sudo apt-get install docker
sudo apt-get install docker-compose
```

To use the docker commands without sudo, add yourself to the docker group and apply the changes

```bash
sudo groupadd docker
#sudo gpasswd -a $USER docker
sudo usermod -aG docker $USER
newgrp docker
```

### AWS CLI

Use the following commands to install the AWS CLI

This depends on python

```bash
pip install awscli --upgrade --user
```

### Go-BinData

Object Drive leverages this capability to package multiple script files for data migrations into binary form.

```bash
env GIT_TERMINAL_PROMPT=1 go get -u github.com/jteeuwen/go-bindata/...
```

### Govendor

While in your `GOPATH` run:

```bash
go get github.com/kardianos/govendor
go install github.com/kardianos/govendor
```

This will install govendor using the go commands so it should be automatically done.

## Clone this repository

All dependent Go code is relative to the **GOPATH**.

Create the the directory **$GOPATH/src/bitbucket.di2e.net/dime**
and clone this project there. 

This will allow imports to resolve correctly.

```bash
cd $GOPATH/src/
mkdir bitbucket.di2e.net/
cd bitbucket.di2e.net/
mkdir dime
cd dime
git clone ssh://git@bitbucket.di2e.net:7999/dime/object-drive-server.git
```

Most of the work is being done on the `develop` branch so that is probably what you will want to use that code:

```bash
cd object-drive-server
git checkout develop
```

If making changes to source code, follow these guidelines

```bash
git fetch
get checkout develop
get rebase origin/develop
get checkout -b some-new-branch-name
```

In short, dont make changes directly to the develop branch


Then you should be able to run the following within a go program

```go
import "bitbucket.di2e.net/dime/object-drive-server/somepackage"
```

## Configure `/etc/hosts`

Once we get everything going, we will be running locally so we need to provide name addressing from the host to the containers.

The easiest thing to do is to add everything to `/etc/hosts` and have all of the container names aliased to localhost.

`aac dias kafka metadatadb redis twl-server-generic2 zk odrive proxier` will need to be added to `localhost`.

Here is an example of what the `/etc/hosts` file may look like:

    ##
    # Host Database
    #
    # localhost is used to configure the loopback interface
    # when the system is booting.  Do not change this entry.
    ##
    127.0.0.1	localhost      aac dias kafka metadatadb redis twl-server-generic2 zk odrive proxier
    255.255.255.255	broadcasthost
    ::1             localhost


The `127.0.0.1	localhost` line is key.

## PKI Certificates

X509 PKI Certificates provide for a single-sign on authentication between
user web browser and the server or intermediary.

Sample test certificates are included in the project codebase which
may be installed after cloning the project.

If you want to use the Drive App locally, or access the Drive App or other applications in the development environments, you'll need to install at least one certificate into your browser.

Certificate 1 is the lowest classification level and 10 is the highest where 0 maps to 10.

### Simplified Mapping of the Certificates

| File | Name | Clearance | Groups |
| --- | --- | --- | --- |
| test_1.p12 | tester01 | U | DCTC/ODrive, DCTC/ODrive_G2 |
| test_2.p12 | tester02 | S? | DCTC/ODrive, DCTC/ODrive_G2 |
| test_3.p12 | tester03 | S? | DCTC/ODrive, DCTC/ODrive_G2 |
| test_4.p12 | tester04 | TS | DCTC/ODrive, DCTC/ODrive_G2 |
| test_5.p12 | tester05 | TS/SI/TK | DCTC/ODrive, DCTC/ODrive_G2 |
| test_6.p12 | tester06 | TS/SI/TK | DCTC/ODrive, DCTC/ODrive_G1 |
| test_7.p12 | tester07 | TS/SI/TK/HCS | DCTC/ODrive, DCTC/ODrive_G1 |
| test_8.p12 | tester08 | TS/SI/TK/G/HCS | DCTC/ODrive, DCTC/ODrive_G1 |
| test_9.p12 | tester09 | TS/SI/TK/G/HCS | DCTC/ODrive, DCTC/ODrive_G1 |
| test_0.p12 | tester10 | TS/SI/TK/G/HCS | DCTC/ODrive, DCTC/ODrive_G1 |

Additional clearance attributes make up the profile and users may be members of additional groups. The unit tests defined in this project assume that users are in the groups as defined above (Users 1-5 are in ODrive G2, Users 6-10 are in ODrive G1, and all 10 users in ODrive)

What is actually granted depends on the dias system in use.

We reference the dias simulator, and profiles can be found here: https://bitbucket.di2e.net/projects/DIME/repos/dias-simulator/browse/client/users

### Installing Certificates in Browser 

You will need the certificates found at

`$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/clients/`. 

The password for all of the `.p12` certification files is `password`. 

## Setup Environment Variables

More details on all of the variables can be found in [docs/environment.md](docs/environment.md).

A minimal set of environment variables and docker configuration are found in [docker/docker-compose-minimal.yml](docker/docker-compose-minimal.yml).  

These are listed below. These exact values can be used when running the containers locally for development purposes. This setup depends upon defaults, and does not leverage AWS for permanent storage, or database instance as it depends on running alongside the referenced metadatadb database container. Event publishing to Kafka is not enabled when using this configuration.  If you are developing a consumer service, refer to the full stack docker-compose files in this project.

It may be most convenient to define these variables in ~/.bashrc or ~/.bash_profile and reference from docker-compose files

```bash
OD_AAC_CA=/go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-aac/trust/client.trust.pem
OD_AAC_CERT=/go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-aac/id/client.cert.pem
OD_AAC_KEY=/go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-aac/id/client.key.pem
OD_AAC_CN=twl-server-generic2
OD_AAC_INSECURE_SKIP_VERIFY=true
OD_PEER_CN=twl-server-generic2
OD_AWS_S3_BUCKET=
OD_DB_CA=/go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-mysql/trust
OD_DB_CERT=/go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-mysql/id/client-cert.pem
OD_DB_CONN_PARAMS=parseTime=true&collation=utf8_unicode_ci&readTimeout=30s
OD_DB_HOST=metadatadb
OD_DB_KEY=/go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-mysql/id/client-key.pem
OD_DB_PASSWORD=dbPassword
OD_DB_PORT=3306
OD_DB_SCHEMA=metadatadb
OD_DB_USERNAME=dbuser
OD_ENCRYPT_MASTERKEY=0
OD_EVENT_PUBLISH_FAILURE_ACTIONS=
OD_EVENT_PUBLISH_SUCCESS_ACTIONS=
OD_SERVER_ACL_WHITELIST1=cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us
OD_SERVER_CA=/go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/server/trust.pem
OD_SERVER_CERT=/go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/server/server.cert.pem
OD_SERVER_CIPHERS=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_RSA_WITH_AES_128_CBC_SHA
OD_SERVER_KEY=/go/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/server/server.key.pem
OD_ZK_AAC=/cte/service/aac/1.2/thrift
```


## Build from Source

Once tools are setup, repositories cloned, and environment variables defined, you should be able to build the code.

A convenient python build script has been created named `odb` (Object Drive Builder).

For a full clean and build, you can run

```bash
./odb --clean --build
```

This will build binaries and docker containers for the project and dependencies.

Note on vendoring. Sometimes the correct version of the source packages gets messed up when building so you may need to delete all of the directories under `vendor/` and re-run `govendor sync` to `get go build ./...` to build with a consistent source tree.

## Running with Docker

Start the docker containers by going into the `docker/` directory and starting the docker-compose file:

``` bash
cd docker/
docker-compose up -d
```

This may take time as it downloads images for the first time.

## Install Drive App

This is optional, and not critical to the success of Object Drive.

While the version of the Drive App available is a few versions behind, if you have access to read from the S3 `decipherers` bucket referenced, you'll be able to download and install a tarball for the user interface by performing the following

``` bash
./installui
```

This should download all of the css and html that are needed to make the UI and run everything. The UI should be visible at: https://localhost:8080/apps/drive/home .

At the time of this writing, this will download version 1.2.10 of the Drive App from January 15, 2019.  
    
It is suitable for viewing files, and may work for searches that hit the the elasticsearch indexes.

## Run Tests

### Upload and download a file using the UI
1. Go to https://localhost:8080/apps/drive/home. Choose a certificate you previously installed in your browser when prompted
1. Click `Upload` on the top right of the screen.
1. Select a file to upload (it really could be anything).
1. select the classification level of the file using the green button on the left of the screen. 
Then select a classification,
1. Click `Upload Files` on the upper right.

You should see a file appear in the console. Now try to download it:
1. Select the file.
1. In the bar at the top click the cloud with a down arrow.
1. Should start a download of the file. Look at it to make sure it isn't corrupted or anything.

If that worked, now we know Object Drive is working! 

### Commandline tests 

Tests can be run locally if the suite of containers defined in 
**docker/docker-compose.yml**
are built and running.  Run the following from the root of the project

``` bash
go test ./... -count 1 -timeout 300m
```
This will take a few minutes to run. 

The APISample.html will be modified when running tests. You SHOULD run tests and commit this
file when opening a pull request so that it can be bundled in documentation as samples since
the current CI/CD solution does not run tests.

The default timeout for go tests is 5 minutes so we told it to take 300 minutes here.


If you are lucky and nothing is currenty broken, you should see this as output:
```?   	bitbucket.di2e.net/dime/object-drive-server/amazon	[no test files]
ok  	bitbucket.di2e.net/dime/object-drive-server/auth	10.801s
ok  	bitbucket.di2e.net/dime/object-drive-server/autoscale	0.026s
ok  	bitbucket.di2e.net/dime/object-drive-server/ciphertext	0.037s
ok  	bitbucket.di2e.net/dime/object-drive-server/client	2.000s
?   	bitbucket.di2e.net/dime/object-drive-server/cmd/obfuscate	[no test files]
?   	bitbucket.di2e.net/dime/object-drive-server/cmd/odrive	[no test files]
?   	bitbucket.di2e.net/dime/object-drive-server/cmd/odrive-database	[no test files]
ok  	bitbucket.di2e.net/dime/object-drive-server/cmd/odrive-test-cli	0.023s
ok  	bitbucket.di2e.net/dime/object-drive-server/cmd/odutil	0.224s [no tests to run]
ok  	bitbucket.di2e.net/dime/object-drive-server/config	0.154s
ok  	bitbucket.di2e.net/dime/object-drive-server/crypto	0.034s
ok  	bitbucket.di2e.net/dime/object-drive-server/dao	7.740s
?   	bitbucket.di2e.net/dime/object-drive-server/events	[no test files]
ok  	bitbucket.di2e.net/dime/object-drive-server/integration	0.101s
ok  	bitbucket.di2e.net/dime/object-drive-server/mapping	0.031s
ok  	bitbucket.di2e.net/dime/object-drive-server/metadata/models	0.030s
?   	bitbucket.di2e.net/dime/object-drive-server/metadata/models/acm	[no test files]
ok  	bitbucket.di2e.net/dime/object-drive-server/performance	2.340s
ok  	bitbucket.di2e.net/dime/object-drive-server/protocol	0.023s
ok  	bitbucket.di2e.net/dime/object-drive-server/server	633.861s
?   	bitbucket.di2e.net/dime/object-drive-server/services/aac	[no test files]
ok  	bitbucket.di2e.net/dime/object-drive-server/services/audit	0.009s
ok  	bitbucket.di2e.net/dime/object-drive-server/services/kafka	0.020s
ok  	bitbucket.di2e.net/dime/object-drive-server/services/zookeeper	0.033s
?   	bitbucket.di2e.net/dime/object-drive-server/ssl	[no test files]
ok  	bitbucket.di2e.net/dime/object-drive-server/util	0.016s
?   	bitbucket.di2e.net/dime/object-drive-server/utils	[no test files]
```


The following are convenience variables that work in conjunction with running integration tests against the object drive service directly instead of through an edge/gateway. 

| Name | Description | 
| --- | --- | 
| OD_EXTERNAL_HOST | Allows for overriding the host name used for go tests when checking server integration tests.  <br />__`Default: proxier`__ |
| OD_EXTERNAL_PORT | Allows for overriding the port used for go tests when checking server integration tests direct to Object Drive. <br />__`Default: 8080`__ |

# Other Tools

Binaries for the main server are built under **cmd/odrive**.

## odutil

Another tool is compiled under **cmd/odutil**. Currently it can upload and
download files from S3. AWS credentials are taken from the environment.

Upload

```
odutil -cmd upload -input somefile.txt -bucket decipher-tools -key some/path/somefile.txt
```

Download

```
odutil -cmd download -input somefile.txt -bucket decipher-tools -key some/path/somefile.txt
```

## Documentation

Generating current docs (no longer checked in):

```
./makedocs
```

## Client

For writing a client to listen on changes and to pull content in response, see this link.

[client](client/README.md)

## RPM Creation

Convenience scripts support assembling an RPM. This is done via Jenkins CI process, but may also be
kicked off locally by running the following in the root folder

```
./makerpm
```

This will also build necessary docker images that encapsulate the build if they dont exist already.
Documentation and binaries will be built inside the container.

An RPM will be built, owned by root with the following naming pattern
`object-drive-{major}.{minor}-{version}-{buildnumber}.{YYYY}{MM}{DD}.x86_64.rpm`

For example: object-drive-1.1-v1.1.0-SNAPSHOT.20190225.x86_64.rpm

The makerpm script supports passing arguments to specify the version, build number, and tag version
used for the package and filename.  For example, to make an RPM as version 1.1.0 and build 2600, 
and tag you can call

```
./makerpm 1.1.0 2600 v1.1.0b4
```

When no arguments are given, the version number is taken by parsing the top most release entry
in the changelog.md file, and the buildnumber is read from the BUILDNUMBER file which defaults to
SNAPSHOT in local environment.

If you want to review the metadata for the RPM

```
rpm -qip *.rpm
```

And if you want to list its file contents

```
rpm -qil *.rpm
```
# Object Drive Server

# API Documentation

API documentation for the Object Drive service may be reviewed at the root of an instantiated object-drive server,
previewed [here](./docs/home.md), or accessed from this [live instance on MEME](https://meme.363-283.io/services/object-drive/1.0/)


# Configuration

Step by step information for starting Object Drive from complete scratch.

## 1. Set up go
1. Make sure `go` is installed, for macs with homebrew it is as simple as `brew install go` or you can download the installer for your platform from https://golang.org/dl/.

2. Set up your "go path". Go is generally centered around a single workspace for *everything*.
 This is usually at `~/go`.
 It is then broken down into `~/go/src/` for all of the source code which is where all of the work actually happens, `~/go/bin/` for the global binaries, and `~/go/pkg/` for global dependencies.
 You will need to set up an environment variable called `GOPATH` to point to this directory and it will save you headaches later if you the binaries directory to your path.
 In your `~/.bash_profile` add:

    ```bash
    # setup Go
    export GOPATH="$HOME/go"
    export PATH=$PATH:$GOPATH/bin
    ```

3. In `$GOPATH`, the `bin` and `pkg` directories mostly handle themselves. However, `src` is highly structured by where the code comes from. So if the code was downloaded from GitHub.com, it would go into the `$GOPATH/src/github.com/` directory, if it came from golang.org it would go in the `$GOPATH/src/golang.org/` directory. This structure makes it easier for dependency managers like `govender` or `dep` (both of which I have encountered in deciphernow repos) to manage the packages.

    So the overall setup of the go workspace would look something like this:
    ```
    $GOPATH                  <-- You set this environment variable (usually ~/go/)
    ├── bin                  <-- Executables go here
    ├── pkg                  <-- Object files (*.a) go here
    └── src                  <-- Source files go here. This is where the work happens
        └── github.com       <-- Folder containing source code from the GitHub repos
            └── deciphernow  <-- Folder containing source code from the deciphernow orginization
        └── golang.org       <-- Folder containing source code from golang.org
    ```

## 2 Install/Set up Dependent Things
#### The protobuf compiler
1. Almost all of the Decipher microservices use [protobuf](https://developers.google.com/protocol-buffers/) which is *a language-neutral, platform-neutral extensible mechanism for serializing structured data*. On macs with homebrew this is: 
    ```brew install protobuf```
    or you can download the installer for your platform by going to https://github.com/google/protobuf/releases and choosing your release that suits your needs.

2. If needed (homebrew may do this for you), add this to your path, I needed to do: 

    `PATH=$PATH:~/Developer/protoc-3.3.0-osx-x86_64/bin`

#### Openssl

This project depends on OpenSSL, and binds to C code (uses CGO).
This means you may need to set the `PKG_CONFIG_PATH` variable. This can vary by distribution.

1. Apple has deprecated use of OpenSSL in favor of its own TLS and crypto libraries, so install it.
On macs with homebrew use `brew install openssl`.

2. Object Drive uses the openssl C bindings, so add it to the correct environment variable.
If using a mac use `export PKG_CONFIG_PATH="$(brew --prefix openssl)/lib/pkgconfig"` or on linux it may be: `export PKG_CONFIG_PATH="/usr/lib/x86_64-linux-gnu/pkgconfig"`

Detailed here: https://gitlab.363-283.io/cte/object-drive/wikis/object-drive-environment-variables

See also the example docker-compose file **.ci/docker-compose.yml** for example environment variables.
Note that some vars are not set directly inline, because they contain secrets (e.g. AWS vars).


#### Python

The scripts depend on python 2. 

Install in Ubuntu linux via
```
sudo apt-get update
sudo apt-get install python2.7 python-pip
```

#### Dot and Graphviz

Building documentation requires dot to be installed. 

Install in Ubuntu Linux via
```
sudo apt-get update
sudo apt-get install graphviz
```

#### Docker and Docker Compose

Building documentation and container images requires docker to be installed. 

Follow the guidance for your platform from the [Docker page](https://docs.docker.com/install/linux/docker-ce/ubuntu/#install-using-the-repository)

In addition, install docker-compose in Ubuntu Linux via
```
sudo apt-get install docker-compose
```

To use the docker commands without sudo, add yourself to the docker group and apply the changes
```
sudo groupadd docker
#sudo gpasswd -a $USER docker
sudo usermod -aG docker $USER
newgrp docker
```

#### AWS CLI

Use the following commands to install the AWS CLI

```
pip install awscli --upgrade --user
```

#### Go-BinData

Object Drive leverages this capability to package multiple script files for data migrations into binary form.

```
env GIT_TERMINAL_PROMPT=1 go get -u github.com/jteeuwen/go-bindata/...
```
#### Maven
If planning to do full clean builds, you may need maven to support dependent projects using java.
```
sudo apt-get install maven
```

#### Development Environment
Developing on this project requires maven, docker, and nodejs configurations.
Also, a separate build "root" directory must be specified by setting the `OD_ROOT`
environment variable.
The build script will check out and build other dependencies
there.
Consider this a volatile directory. This can be anywhere, but a fine choice is `$HOME/my_code/od_root/`

1. Create the directory `$HOME/my_code/od_root/`.
2. Then added `export OD_ROOT=$HOME/my_code/od_root` to my `.bash_profile`.


#### Dependent Projects
If you are developing changes with AAC and DIAS, you will need to retrieve those from a private gitlab repository at gitlab.363-283.io. The project names are

* AAC is at cte/cte-security-service
  * A docker image is available at: `docker.363-283.io/cte/cte-security-service:1.2.2-SNAPSHOT`
* Redis is used by AAC
  * A docker image is available at: `docker.363-283.io/docker/backend:redis-3.2.2`
* DIAS is at bedrock/dias-simulator
  * A docker image is available at: `deciphernow/dias:latest`

Additional external projects referenced by the docker-compose file for testing full event to indexing, and user access

* Finder is a facade around elastic search.
  * A docker image is available at: `docker.363-283.io/bedrock/finder-service:1.0.0-SNAPSHOT`
* Elastic Search gives some json document search capability
  * A docker image is available at: `docker.363-283.io/docker/backend:elasticsearch-1.7.2`
* ODrive Indexer listens for events on kafka, and saves to Elastic Search
  * A docker image is available at: `docker.363-283.io/bedrock/object-drive-indexer-service:1.0.0-SNAPSHOT`
* User Service
  * A docker image is available at: `docker.363-283.io/chimera/user-service:1.0.1-SNAPSHOT`
* Postgres is used by the user service for storing data
  * A docker image is available at: `docker.363-283.io/docker/backend:postgres-9.4`
  

If you have gitlab.363-283.io credentials, you can clone these projects into OD_ROOT as follows
```
cd $OD_ROOT
git clone ssh://git@gitlab.363-283.io:2252/cte/cte-security-service.git
git clone ssh://git@gitlab.363-283.io:2252/bedrock/dias-simulator.git
```

When these are present, they can be built when calling `./odb --build`, which will allow you to edit configuration in your dias-simulator instance to alter user permissions for testing.

#### Update `/etc/hosts`
Once we get everything going, we will be running locally so we need to provide name addressing from the host to the containers.
The easiest thing to do is to add everything to `/etc/hosts` and have all of the container names aliased to localhost.
`aac consumers01 dias dockervm gatekeeper jobs01 kafka metadatadb nginx01 packager python01 redis salt service01 twl-server-generic2 zk zookeeper proxier odrive` will need to be added to `localhost`.
Here is an example of what the `/etc/hosts` file may look like:

```
##
# Host Database
#
# localhost is used to configure the loopback interface
# when the system is booting.  Do not change this entry.
##
127.0.0.1	localhost      aac consumers01 dias dockervm gatekeeper jobs01 kafka metadatadb nginx01 packager python01 redis salt service01 twl-server-generic2 zk zookeeper proxier odrive
255.255.255.255	broadcasthost
::1             localhost
```

The `127.0.0.1	localhost` line is key.


#### Install `govendor`
While in your `GOPATH` run:
``` bash
go get github.com/kardianos/govendor
go install github.com/kardianos/govendor
```
This will install govendor using the go commands so it should be automatically done.

#### Install PKI Certificates in Browser
This needs to be done **after** you have downloaded the code in step 4, but it is **VERY** important!
As you might have guessed, we care about the security level of the documents that get uploaded to Object Drive so we have certificates telling AAC what classification level the certificate grants access to, among other security attributes.
This dictates which documents you can view and share.
We need to install at least one (probably best to do all) into your browser. Certificate 1 is the lowest classification level and 10 is the highest where 0 maps to 10.
A simplified mapping of the certificates looks like this:

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

What is actually granted depends on the dias system in use.
We reference the dias simulator, and profiles can be found here: https://gitlab.363-283.io/bedrock/dias-simulator/blob/master/client/users

On a mac, this is handled using your keychain. If you use chrome follow the *Importing your Certificate into Chrome* section of https://www.comodo.com/support/products/authentication_certs/setup/mac_chrome.php for installing the certificates. 
You will need the certificates found in `$GOPATH/src/github.com/deciphernow/object-drive-server/defaultcerts/clients/`. The password for all of the `.p12` certification files is `password`. 

## 3. Set up environment for Object Drive Server
There are a **lot** of environment variables that need to be set up for Object Drive to function properly.
More detailed on all of the variables can be found in [docs/environment.md](docs/environment.md).

1. To start, get AWS credentials I talked to Rob Fielding (@rfielding) and Lucas Moten (@lucasmoten), what is needed is the `aws_access_key_id` and `aws_secret_access_key` which they will provide.

2. Having already set the variables for everything in the previous section, here are all of the other environment variables that I needed to get Object Drive working, keep in mind I have already defined `GOPATH`:
    ```bash
    # local Object Drive things
    export OD_ENCRYPT_MASTERKEY=hi
    export OD_AWS_ACCESS_KEY_ID="your key"
    export OD_AWS_SECRET_ACCESS_KEY="your key"

    export OD_AAC_CA=$GOPATH/src/github.com/deciphernow/object-drive-server/defaultcerts/client-aac/trust/client.trust.pem
    export OD_AAC_CERT=$GOPATH/src/github.com/deciphernow/object-drive-server/defaultcerts/client-aac/id/client.cert.pem
    export OD_AAC_CN=twl-server-generic2
    export OD_AAC_KEY=$GOPATH/src/github.com/deciphernow/object-drive-server/defaultcerts/client-aac/id/client.key.pem
    export OD_AAC_ZK_ADDRS=zk:2181
    export OD_AWS_REGION=us-east-1
    export OD_AWS_S3_BUCKET=decipherers
    export OD_AWS_S3_ENDPOINT=s3.amazonaws.com
    export OD_AWS_S3_FETCH_MB=16
    export OD_CACHE_EVICTAGE=300
    export OD_CACHE_HIGHWATERMARK=0.75
    export OD_CACHE_LOWWATERMARK=0.50
    export OD_CACHE_PARTITION=[name no space]
    export OD_CACHE_WALKSLEEP=30
    export OD_CERTPATH=$GOPATH/src/github.com/deciphernow/object-drive-server/defaultcerts
    export OD_DB_CA=/go/src/github.com/deciphernow/object-drive-server/defaultcerts/client-mysql/trust
    export OD_DB_CERT=/go/src/github.com/deciphernow/object-drive-server/defaultcerts/client-mysql/id/client-cert.pem
    export OD_DB_CONN_PARAMS='parseTime=true&collation=utf8_unicode_ci&readTimeout=30s'
    export OD_DB_HOST=metadatadb
    export OD_DB_KEY=/go/src/github.com/deciphernow/object-drive-server/defaultcerts/client-mysql/id/client-key.pem
    export OD_DB_MAXIDLECONNS=5
    export OD_DB_MAXOPENCONNS=10
    export OD_DB_PASSWORD=dbPassword
    export OD_DB_PORT=3306
    export OD_DB_SCHEMA=metadatadb
    export OD_DB_USERNAME=dbuser
    export OD_EVENT_PUBLISH_FAILURE_ACTIONS=disabled
    export OD_EVENT_PUBLISH_SUCCESS_ACTIONS=create,delete,undelete,update
    export OD_EVENT_ZK_ADDRS=zk:2181
    export OD_LOG_LEVEL=0
    export OD_PEER_CN=twl-server-generic2
    export OD_PEER_SIGNIFIER=P2P
    export OD_SERVER_BASEPATH=/services/object-drive/1.0
    export OD_SERVER_CA=$GOPATH/src/github.com/deciphernow/object-drive-server/defaultcerts/server-web/trust/server.trust.pem
    export OD_SERVER_CERT=$GOPATH/src/github.com/deciphernow/object-drive-server/defaultcerts/server-web/id/server.cert.pem
    export OD_SERVER_KEY=$GOPATH/src/github.com/deciphernow/object-drive-server/defaultcerts/server-web/id/server.key.pem
    export OD_SERVER_PORT=4430
    export OD_ZK_AAC=/cte/service/aac/1.0/thrift
    export OD_ZK_ANNOUNCE=/services/object-drive/1.0
    export OD_ZK_TIMEOUT=5
    export OD_ZK_URL=zk:2181
    ```


## 4. Clone this repository

All dependent Go code is relative to the **GOPATH**.
Create the the directory **$GOPATH/src/github.com/deciphernow**
and clone this project there. This will allow imports like this to resolve correctly.

``` bash
cd $GOPATH/src/
mkdir github.com/
cd github.com/
mkdir deciphernow
cd deciphernow
git clone https://github.com/DecipherNow/object-drive-server.git
```

Most of the work is being done on the `develop` branch so that is probably what you will want to use that code:
``` bash
cd object-drive-server
git checkout develop
```

Then you should be able to run the following within a go program

```go
import "github.com/deciphernow/object-drive-server/somepackage"
```

## 5. Build and start the Object Drive Source code
1. We should be able to move on to building the code!
    There is already a nice python build script set up for us called `odb`.
    This will download all of the images we need and set all of the services running and perform vendoring.
    So use `odb` to build the project, NOTE this takes a while to run and will download a lot of images:

    ```bash
    ./odb --build
    ```

    `odb` is a python script that builds binaries and docker containers for this
    project and its dependencies. It also inspects your build environment, and
    notifies you when tools are missing.

    Note on vendoring. Sometimes the correct version of the source packages gets messed up when building so you may need to delete all of the directories under `vendor/` and re-run `govendor sync` to `get go build ./...` to build with a consistent source tree.

2. Start the docker containers by going into the `docker/` directory and starting the docker-compose file:
    ``` bash
    cd docker/
    docker-compose up -d
    ```
    This will also take a while as it will donwload more images and start everything.

3. Once all of the images are up and running, start up the UI using the `installui` script in the `docker/` directory:
    ``` bash
    ./installui
    ```
    This should download all of the css and html that are needed to make the UI and run everything. The UI should be visible at: https://localhost:8080/apps/drive/home .


## 6. Run Tests
##### 1. Upload and download a file using the UI
1. Go to https://localhost:8080/apps/drive/home.
If this is the first time you are using it you should be prompted for a certificate to use. select a certificate, test9 or test10 is usually a good one to use.
2. Click `Upload` on the top right of the screen.
3. Select a file to upload (it really could be anything).
4. select the classification level of the file using the green button on the left of the screen. 
Then select a classification,
5. Click `Upload Files` on the upper right.

You should see a file appear in the console. Now try to download it:
1. Select the file.
2. In the bar at the top click the cloud with a down arrow.
3. Should start a download of the file. Look at it to make sure it isn't corrupted or anything.

If that worked, now we know Object Drive is working! 

##### 2. Commandline tests 
Tests can be run locally if the suite of containers defined in **docker/docker-compose.yml**
are built and running. Run `go test ./...` from the root of this project.

Assuming that everything is up and running in all of the previous steps, we can run the testing suite on the code to make sure that it is actually going properly.
So in the root directory of `object-drive-server` run:
``` bash
go test ./... -timeout 3000m
```
This will take a long time to run. 
The default timeout for go tests is 5 minutes so we told it to take 300 minutes here.


If you are lucky and nothing is currenty broken, you should see this as output:
```?   	github.com/deciphernow/object-drive-server/amazon	[no test files]
ok  	github.com/deciphernow/object-drive-server/auth	10.801s
ok  	github.com/deciphernow/object-drive-server/autoscale	0.026s
ok  	github.com/deciphernow/object-drive-server/ciphertext	0.037s
ok  	github.com/deciphernow/object-drive-server/client	2.000s
?   	github.com/deciphernow/object-drive-server/cmd/obfuscate	[no test files]
?   	github.com/deciphernow/object-drive-server/cmd/odrive	[no test files]
?   	github.com/deciphernow/object-drive-server/cmd/odrive-database	[no test files]
ok  	github.com/deciphernow/object-drive-server/cmd/odrive-test-cli	0.023s
ok  	github.com/deciphernow/object-drive-server/cmd/odutil	0.224s [no tests to run]
ok  	github.com/deciphernow/object-drive-server/config	0.154s
ok  	github.com/deciphernow/object-drive-server/crypto	0.034s
ok  	github.com/deciphernow/object-drive-server/dao	7.740s
?   	github.com/deciphernow/object-drive-server/events	[no test files]
ok  	github.com/deciphernow/object-drive-server/integration	0.101s
ok  	github.com/deciphernow/object-drive-server/mapping	0.031s
ok  	github.com/deciphernow/object-drive-server/metadata/models	0.030s
?   	github.com/deciphernow/object-drive-server/metadata/models/acm	[no test files]
ok  	github.com/deciphernow/object-drive-server/performance	2.340s
ok  	github.com/deciphernow/object-drive-server/protocol	0.023s
ok  	github.com/deciphernow/object-drive-server/server	633.861s
?   	github.com/deciphernow/object-drive-server/services/aac	[no test files]
ok  	github.com/deciphernow/object-drive-server/services/audit	0.009s
ok  	github.com/deciphernow/object-drive-server/services/kafka	0.020s
ok  	github.com/deciphernow/object-drive-server/services/zookeeper	0.033s
?   	github.com/deciphernow/object-drive-server/ssl	[no test files]
ok  	github.com/deciphernow/object-drive-server/util	0.016s
?   	github.com/deciphernow/object-drive-server/utils	[no test files]
```



# Other Configuration

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

Generating current docs (no longer checked in):

```
./makedocs
```

Making an rpm (will build docs as well):

```
cd $GOPATH/src/github.com/deciphernow/object-drive-server
#make an rpm as version 1.0.9 and call it build number 2600.  It will be in current directory when done
./makerpm 1.0.9 2600
```

## Client

For writing a client to listen on changes and to pull content in response, see this link.

[client](client/README.md)

# Object Drive Server

Object Drive provides for secure storage and high performance retrieval of hierarchical folder organization of objects that are named, owned and managed by users and their groups. Access control is facilitated by integration of the User AO and Object ACM model through AAC policy guidance. File streams are encrypted in transit and at rest.

# API Documentation

API documentation for the Object Drive service may be reviewed at the root of an instantiated object-drive server,
previewed [here](./docs/home.md), or accessed from this [live instance on MEME](https://meme.dime.di2e.net/services/object-drive/1.0/)

# Developers Using Object Drive

To integrate another service or application with Object Drive, we recommend using Docker. Images for the 51th release of the service and database are available here:

* docker-dime.di2e.net/dime/object-drive-server:1.0.21b4
* docker-dime.di2e.net/dime/object-drive-metadatadb:1.0.21b4

Those wanting to experiment with the most recent changes can use these images

* docker-dime.di2e.net/dime/object-drive-server:1.0.22b4-release-candidate
* docker-dime.di2e.net/dime/object-drive-metadatadb:1.0.22b4-release-candidate

There is no 'latest' tagged images unless you build them from the source herein. Any such tags are wholely unsupported and any requests for such will be redirected to the above image tags


The [docker folder](./docker/README.md) of this project has a series of compose files for different testing use cases.  The [minimal-docker-compose.yml](./docker/minimal-docker-compose.yml) is a good starting point.


# Developers Enhancing Object Drive

Step by step information for starting Object Drive from complete scratch.

* [Guidance for Linux Environments](README.linux.md)
* [Guidance for Mac Environments](README.mac.md)



# Docker-Compose Configuration Files

This folder contains different docker compose configuration files

The images referenced for Object Drive are assumed to have been built by the developer by first running the following in the root of the project.

```
cd $GOPATH/src/bitbucket.di2e.net/dime/object-drive-server
./odb --clean --build
```

## docker-compose.yml

This is the default file that will be used if not specifying any of the other files.  It will bring up a basic stack suitable for also using the Drive UI application, but without full indexing and search support.

This will use the latest tagged images for odrive-bc and metadatadb. These are snapshot images that you the developer will have available from running the `odb` command above.

To start
```
docker-compose up -d
```

After starting, you can install the Drive UI
```
./installui
```

To stop and remove containers
```
docker-compose kill && docker-compose rm -f
```

## fullstack-docker-compose.yml

This configuration file can be used to bring up a full object drive environment with kafka, an indexer, elastic search, user service and more to support all functionality when installing the Drive UI

This will use the latest tagged images for odrive-bc and metadatadb. These are snapshot images that you the developer will have available from running the `odb` command above.

To start
```
docker-compose -f fullstack-docker-compose.yml up -d
```

After starting, you can install the Drive UI
```
./installui
```

To stop and remove containers
```
docker-compose -f fullstack-docker-compose.yml kill && docker-compose -f fullstack-docker-compose.yml rm -f
```

## minimal-docker-compose.yml

This configuration file is handy to provide others integrating with the Object Drive service.  All images references are tagged and retrievable from the DI2E docker repository.  It will not include the latest changes to the develop branch as it references tagged releases.  Neither the Drive UI, nor other applications will work with this unless their specific dependencies are also installed into the docker network.

To start
```
docker-compose -f minimal-docker-compose.yml up -d
```

To stop and remove containers
```
docker-compose -f minimal-docker-compose.yml kill && docker-compose -f minimal-docker-compose.yml rm -f
```

# Helpful Docker and Docker Compose commands

If you want to delete all local docker images and containers, and start from a clean base, do:

```
docker rm $(docker ps -a -q)
docker rmi $(docker images -f "dangling=true" -q)
```

# Docker Container Interfaces

All docker container folders have an interface for creating and running images,
even before docker-compose is used:

* cleanit
* makeimage (may depend on cleanit)

The top level will use these to make sure that images are build in the standard way, so that they can be launched with compose.

# Troubleshooting

----

If you run `docker-compose up -d` and receive a message like

```
ERROR: No such image: <HASH_IDENTIFIER>...
```

Try running `docker-compose rm` and then `docker-compose up -d` again.


----

If you get the error `Cannot connect to the Docker daemon. ...` you can try the following
command, replacing `decipher-dev` with the name of your docker-machine VM instance.

```
$ eval "$(docker-machine env decipher-dev)"
```

----

### Unicode error when running docker-compose logs

The underlying code supporting `docker-compose` is Python. The `logs` command sometimes fails 
to intelligently parse and forward the stdout stream from running containers. Please note
that this does not necessarily mean that your container is dead. Run `docker-compose ps` to 
view running containers.

----

_add more helpful Docker tips here..._

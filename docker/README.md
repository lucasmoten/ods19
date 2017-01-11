
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
* runimage (depends on makeimage)

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

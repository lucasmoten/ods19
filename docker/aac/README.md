# Docker Containers For Environment Dependencies

Pass in the root where built (ie: maven) cte-security-service lives:

* BUILT_HOME=~/gitlab/chimera ./cleanit
* BUILT_HOME=~/gitlab/chimera ./makeimage
* BUILT_HOME=~/gitlab/chimera ./runimage

Each target calls the previous one (prerequisites).  This lets us make
docker containers of the latest items.


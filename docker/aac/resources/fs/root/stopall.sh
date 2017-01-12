#!/bin/bash

(
/etc/init.d/cte-aac-service-server-1.0.0 stop
/opt/zookeeper/bin/zkServer.sh stop
killall redis-server
)

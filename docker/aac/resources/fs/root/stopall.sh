#!/bin/bash

(
/etc/init.d/aac-1.1.3 stop
/opt/zookeeper/bin/zkServer.sh stop
killall redis-server
)

#!/bin/bash

(
/etc/init.d/aac-1.1.4 stop
/opt/zookeeper/bin/zkServer.sh stop
killall redis-server
)

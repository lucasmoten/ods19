#!/bin/bash

(
APP=cte-aac-service-server-1.0.0
/usr/sbin/redis-server /etc/redis.conf 2>&1 &
/opt/zookeeper/bin/zkServer.sh start
/etc/init.d/$APP start
)

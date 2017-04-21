#!/bin/bash

(
APP=aac-1.1.4
/usr/sbin/redis-server /etc/redis.conf 2>&1 &
/opt/zookeeper/bin/zkServer.sh start
/etc/init.d/$APP start
)

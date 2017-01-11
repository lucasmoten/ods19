#!/bin/bash

export PID_FILE=/opt/bedrock/redis/redis_bedrock.pid
export REDIS_HOME=/opt/bedrock/redis

case "$1" in
    start)
        echo "Starting Redis"
        cd ${REDIS_HOME}/bin
	./redis-server ${REDIS_HOME}/redis.conf
        echo $! > $PID_FILE
    ;;
    stop)
        echo "Shutting down Redis"
        awk '{print $0}' $PID_FILE | xargs kill
        rm -rf $PID_FILE
    ;;
    status)
    ;;
    restart)
     stop
     start
    ;;
    *)
    echo "Usage: $0 {start|stop|restart}"
    exit 1
    ;;
esac

exit 0

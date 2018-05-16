#!/bin/bash

ROOT=/tmp/x86_64/terminal
BINARY=`cd ..`
echo "Automatically create exmaple folder under /tmp with $ROOT"
mkdir -p $ROOT
mkdir -p $ROOT/bin
mkdir -p $ROOT/lib
cp -n pid $ROOT/bin
cp -n gitpid.so $ROOT/lib
echo "checking if there is memcached instance running on your os..."
MEM_PID=`ps -aux|grep memcached|grep -v "grep"|awk '{print $2}'`
if [ $MEM_PID != "" ];then
  kill -9 $MEM_PID
fi
LD_PRELOAD=$BINARY/libevent.so ./$BINARY/memcached
./$BINARY/lpmx init
./$BINARY/lpmx run -c $BINARY/setting.yml -s $ROOT

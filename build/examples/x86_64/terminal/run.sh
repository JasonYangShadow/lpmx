#!/bin/bash

if [ -f "/usr/bin/uname" ] || [ -f "/bin/uname" ]; then
  ARCH_OS=`uname -o`
  ARCH_PLAT=`uname -m`
else
  echo "your os doesn't have uname, it may not be compatible with this script"
  exit 1
fi

if [ $ARCH_OS = "GNU/Linux" ];then
  ROOT=/tmp/x86_64/terminal
  BUILD_ROOT=`cd .././../`
  mkdir -p $ROOT
  cp -n $BUILD_ROOT/linux/$ARCH_PLAT ./
  mkdir -p $ROOT/bin
  mkdir -p $ROOT/lib
  cp -n pid $ROOT/bin
  cp -n gitpid.so $ROOT/lib
  chmod 755 lpmx
  chmod 755 memcached  
  MEM_PID=`ps -aux|grep memcached|grep -v "grep"|awk '{print $2}'`
  if [ $MEM_PID != "" ];then
    kill -9 $MEM_PID
  fi
  LD_PRELOAD=./libevent.so ./memcached
  ./lpmx init
  ./lpmx run -c ./setting.yml -s $ROOT
else
  echo "your os is not supported"
  exit 1
fi

#!/bin/bash

ROOT=/tmp/x86_64/rpc
BINARY="$(dirname $(dirname $(dirname `pwd`)))"
CURRENT=`pwd`
LOG=$ROOT/log

echo "cleanup"
rm -rf $ROOT/.lpmx
rm -rf $BINARY/.lpmxsys
killall lpmx

echo "Automatically create exmaple folder under /tmp with $ROOT"
mkdir -p $ROOT
cp -n loop1 loop2 $ROOT
echo "checking if there is memcached instance running on your os..."
MEM_PID=`ps -aux|grep memcached|grep -v "grep"|awk '{print $2}'`
if [ -n "$MEM_PID" ];then
  echo "memcached instance with pid $MEM_PID will be killed"
  kill -9 $MEM_PID
fi
echo "restarting memcached server"
export LD_PRELOAD=$BINARY/libevent.so
cd $BINARY
./memcached -d
NEW_MEM_PID=`ps -aux|grep memcached|grep -v "grep"|awk '{print $2}'`
if [ -n "$NEW_MEM_PID" ];then
  echo "memcached instance is restarted with new pid $NEW_MEM_PID"
else
  echo "restarting memcached instace encountered error"
  exit 1
fi
if [ -f "readme" ];then
  cat readme
fi
./lpmx init
./lpmx run -c $CURRENT/setting.yml -s $ROOT -p > $LOG &
LPMXID=`ps -aux|grep lpmx|grep -v "grep"|awk '{print $2}'`
if [ -n "$LPMXID" ];then
  echo "lpmx container started in passive mode(rpc mode) in background with pid
  $LPMXID"
else
  echo "starting lpmx container encountered error"
  exit 1
fi
echo "you might want to use './lpmx list' to check the container info"
./lpmx list
RPC=`./lpmx list`
while read -r line; do
  CID=$(echo $line|awk '{print $1}')
  CROOT=$(echo $line|awk '{print $2}')
  CSTATUS=$(echo $line|awk '{print $3}')
  CRPC=$(echo $line|awk '{print $4}')
  echo $CRPC $CID $CSTATUS
  if [ -n $CRPC ] && [ ! -z $CID ] && [ $CSTATUS = "RUNNING" ]; then
    echo "the follwing command will be executed to trigger remote command via
    rpc"
    CMD1="./lpmx rpc exec -i localhost -p $CRPC $ROOT/loop1" 
    CMD2="./lpmx rpc exec -i localhost -p $CRPC $ROOT/loop2" 
    echo "$CMD1"
    eval $CMD1
    echo "$CMD2"
    eval $CMD2
  fi
done <<< "$RPC"

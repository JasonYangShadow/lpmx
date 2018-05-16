#!/bin/bash

SRC="https://raw.githubusercontent.com/JasonYangShadow/lpmx/master/build"
if [ -f "/usr/bin/uname" ] || [ -f "/bin/uname" ]; then
    ARCH_OS=`uname -o`
    ARCH_PLAT=`uname -m`
else
    echo "your os doesn't have uname, it may not be compatible with this
    installment script"
    exit 1
fi
if [ ! -f "/usr/bin/wget" ];then
    echo "wget dees not exist in your os, please install wget"
    exit 1
fi

get_binary(){
  if [ $1 = "GNU/Linux" ];then
    echo "installment script will create folder named lpmx in current directory"
    ROOT=lpmx
    mkdir -p $ROOT
    wget $3/$2/libevent.so -P $ROOT 
    wget $3/$2/libfakechroot.so -P $ROOT 
    wget $3/$2/lpmx -P $ROOT 
    wget $3/$2/memcached -P $ROOT 
    wget $3/$2/patchelf -P $ROOT 
    chmod 755 $ROOT/lpmx $ROOT/memcached $ROOT/patchelf
  fi
}

install(){
  get_binary $ARCH_OS $ARCH_PLAT $SRC/linux
}

get_terminal(){
  if [ -d "lpmx" ];then
    ROOT=lpmx/examples/$1/terminal
    mkdir -p $ROOT
    wget $2/exmaples/$1/terminal/getpid.so -P $ROOT
    wget $2/exmaples/$1/terminal/pid -P $ROOT
    wget $2/exmaples/$1/terminal/readme.md -P $ROOT
    wget $2/exmaples/$1/terminal/setting.yml -P $ROOT
    wget $2/exmaples/$1/terminal/run.sh -P $ROOT
    chmod 755 $ROOT/pid $ROOT/run.sh
  else
   echo "sorry, i can't find lpmx directory, seems the installment encountered
   errors!"
  exit 1 
  fi
}

get_rpc(){
  if [ -d "lpmx" ];then
    ROOT=lpmx/examples/$1/rpc
    mkdir -p $ROOT
    wget $2/examples/$1/rpc/readme.md -P $ROOT
    wget $2/examples/$1/rpc/run.sh -P $ROOT
    wget $2/examples/$1/rpc/loop1 -P $ROOT
    wget $2/examples/$1/rpc/loop2 -P $ROOT
    wget $2/examples/$1/rpc/setting.yml -P $ROOT
    chmod 755 $ROOT/loop1 $ROOT/loop2 $ROOT/run.sh
  else
   echo "sorry, i can't find lpmx directory, seems the installment encountered
   errors!"
  exit 1 
  fi
}

download_example(){
  get_terminal $ARCH_PLAT $SRC
  get_rpc $ARCH_PLAT $SRC
}

install
dowload_exmaple

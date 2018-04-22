#!/bin/bash

echo "-------------------------------------------------------------------------------------"
echo " Please make sure that your os has the following apps(git,autoconf,make,libtool,g++,libmemcached-dev zlib1g-dev msgpack-c cmake) "
echo "-------------------------------------------------------------------------------------"

FOLDER=/tmp/lpmx_test
ELFDIR=/tmp/patchelf
CHROOTDIR=/tmp/fakechroot

if [ -d "$FOLDER" ];then
    rm -rf "$FOLDER"
fi

mkdir -p "$FOLDER"
mkdir -p "$FOLDER/bin"
mkdir -p "$FOLDER/lib"

if [ ! -d "$ELFDIR" ];then
    git clone https://github.com/JasonYangShadow/patchelf /tmp/patchelf
    cd /tmp/patchelf
    ./bootstrap.sh
    ./configure
    make
fi

if [ ! -d "$CHROOTDIR" ];then
    git clone https://github.com/JasonYangShadow/fakechroot /tmp/fakechroot
    cd /tmp/fakechroot
    ./autogen.sh
    ./configure
    make
fi

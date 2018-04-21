#!/bin/bash

echo "-------------------------------------------------------------------------------------"
echo " Please make sure that your os has the following apps(git,autoconf,make,libtool,g++,libmemcached-dev zlib1g-dev msgpack-c cmake) "
echo "-------------------------------------------------------------------------------------"

FOLDER=/tmp/lpmx_test
ROOTDIR=/tmp/lpmx_root
ELFDIR=/tmp/patchelf
CHROOTDIR=/tmp/fakechroot

if [ -d "$FOLDER" ];then
    rm -rf "$FOLDER"
fi

mkdir -p "$FOLDER"
mkdir -p "$FOLDER/bin"
mkdir -p "$FOLDER/lib"
mkdir -p "$ROOTDIR"

if [ ! -f "$ROOTDIR/setting.yml" ];then
    touch "$ROOTDIR/setting.yml"
    echo "RootDir: /tmp/lpmx_root" > "$ROOTDIR/setting.yml"
fi

if [ ! -d "$ELFDIR" ];then
    git clone https://github.com/JasonYangShadow/patchelf /tmp/patchelf
    cd /tmp/patchelf
    ./bootstrap.sh
    ./configure
    make
    cp src/patchelf "$ROOTDIR"
fi

if [ ! -d "$CHROOTDIR" ];then
    git clone https://github.com/JasonYangShadow/fakechroot /tmp/fakechroot
    cd /tmp/fakechroot
    ./autogen.sh
    ./configure
    make
    cp src/.libs/libfakechroot.so "$ROOTDIR"
fi

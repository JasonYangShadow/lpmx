#!/bin/bash

SRC="https://github.com/JasonYangShadow/lpmx/tree/master/build"
if [ -f "/usr/bin/uname" ];then
    ARCH=`uname -m`
    mkdir -p lpmx
    if [ -f "/usr/bin/wget" ];then
        cd lpmx
        wget "$SRC/linux/$ARCH/lpmx-linux-$ARCH"
        wget "$SRC/libfakechroot.so"
        wget "$SRC/patchelf"
        chmod 755 lpmx
    else
        echo "please install wget"
    fi
    
fi

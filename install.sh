#!/bin/bash

SRC="https://raw.githubusercontent.com/JasonYangShadow/lpmx/master/build/linux/"
ARCH=`uname -m`
mkdir -p lpmx
if [ -f "/usr/bin/wget" ];then
    cd lpmx
    wget "$SRC/$ARCH/lpmx-linux-$ARCH"
    wget "$SRC/$ARCH/libfakechroot.so"
    wget "$SRC/$ARCH/patchelf"
    chmod 755 lpmx-linux-$ARCH
    chmod 755 patchelf
else
    echo "please install wget"
fi
    

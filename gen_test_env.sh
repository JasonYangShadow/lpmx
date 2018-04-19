#!/bin/bash

FOLDER=/tmp/lpmx_test
ROOTDIR=/tmp/lpmx_root

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

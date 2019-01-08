#!/bin/bash

for GOOS in linux; do
    mkdir -p build/$GOOS
#    for GOARCH in i686 x86_64; do
     for GOARCH in x86_64; do
        mkdir -p build/$GOOS/$GOARCH
        if [ $GOARCH = "i686" ];then
          env GOOS=$GOOS GOARCH="386" go build -v -o build/$GOOS/$GOARCH/lpmx
        fi
        if [ $GOARCH = "x86_64" ];then
          env GOOS=$GOOS GOARCH="amd64" go build -v -o build/$GOOS/$GOARCH/lpmx
        fi
     done
done

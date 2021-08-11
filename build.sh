#!/bin/bash
for GOOS in linux; do
    mkdir -p build/$GOOS
#    for GOARCH in i686 x86_64; do
     for GOARCH in x86_64; do
        mkdir -p build/$GOOS/$GOARCH
        if [ $GOARCH = "x86_64" ];then
          env GOOS=$GOOS GOARCH="amd64" GO111MODULE=off go build -v -o build/$GOOS/$GOARCH/Linux-x86_64-lpmx
        #generate log first
          if [ -x "$(command -v chglog)" ];then
              chglog init
          fi
        #generate deb/rpm pacakges
          if [ -x "$(command -v nfpm)" ];then
              nfpm pkg --packager deb --target build/$GOOS/$GOARCH/Linux-x86_64-lpmx.deb
              nfpm pkg --packager rpm --target build/$GOOS/$GOARCH/Linux-x86_64-lpmx.rpm
          fi
        fi
     done
done

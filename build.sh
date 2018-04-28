#!/bin/bash

for GOOS in linux; do
    mkdir -p build/$GOOS
    for GOARCH in i386 x86_64; do
        mkdir -p build/$GOOS/$GOARCH
        go build -v -o build/$GOOS/$GOARCH/lpmx-$GOOS-$GOARCH github.com/jasonyangshadow/lpmx
     done
done

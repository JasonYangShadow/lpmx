#!/bin/bash

echo "making 64bit tar ball"
A64="lpmx_x86_64.tar.gz"
if [ -f $A64 ];then
  rm -rf $A64
fi
tar -czf lpmx_x86_64.tar.gz examples/x86_64 linux/x86_64

echo "making 32bit tar ball"
A32="lpmx_i686.tar.gz"
if [ -f $A32 ];then
  rm -rf $A32
fi
tar -czf lpmx_i686.tar.gz examples/i686 linux/i686


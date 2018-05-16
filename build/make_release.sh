#!/bin/bash

echo "making 64bit tar ball"
tar -czf lpmx_x86_64.tar.gz examples/x86_64 linux/x86_64

echo "making 32bit tar ball"
tar -czf lpmx_i686.tar.gz examples/i686 linux/i686


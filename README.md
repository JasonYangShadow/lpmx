# lpmx
-------
lpmx is rootless container other than local package manager. 
It employs the LD_PRELOAD mechanism and ELF header patch to implement both elf modification and system calls interception.

Therefore, this project contains customized [fakechroot branch](https://github.com/JasonYangShadow/fakechroot) and [elfpatcher](https://github.com/JasonYangShadow/patchelf). 

# Compile from source code 
------
1. Make sure golang is installed on your os
2. go get -v github.com/jasonyangshadow/lpmx
3. cd $GOPATH/src/github.com/jasonyangshadow/lpmx
4. ./build.sh
5. You will locate the 32bit/64bit binary under build/linux/x86_64/lpmx or build/linux/i386/lpmx 

# How to use it
-------
1. Make sure you have memcached installed on your os and start it with "memcached -d". As lpmx currently depends on the memcached to exchange privilges information.
2. ./lpmx init  -> initialize the basic system folder for lpmx ( otherwize, any commands executed following will report error and notify you should execute initialize command firstly)
3. ./lpmx run -c config_file_path -s target_container_root_folder -> creates and starts running container based on configuration file and target folder via terminal. 
4. ./lpmx run -c config_file_path -s target_container_root_folder -p -> creates and starts running container in passive mode, which will result in opening rpc port to receive commands and no interaction terminal is triggered. 

# Other commands
------
```
lpmx rootless container

Usage:
  lpmx [command]

Available Commands:
  destroy     destroy the registered container
  help        Help about any command
  init        init the lpmx itself
  list        list the containers in lpmx system
  resume      resume the registered container
  rpc         exec command remotely
  run         run container based on specific directory
  set         set environment variables for container

Flags:
  -h, --help   help for lpmx

Use "lpmx [command] --help" for more information about a command.
```

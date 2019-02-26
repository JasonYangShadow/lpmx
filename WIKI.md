# All Commands
```
lpmx rootless container

Usage:
  lpmx [command]

Available Commands:
  destroy     destroy the registered container
  docker      docker command
  expose      expose program inside container
  get         get settings from memcache server
  help        Help about any command
  init        init the lpmx itself
  list        list the containers in lpmx system
  resume      resume the registered container
  set         set environment variables for container
  uninstall   uninstall lpmx completely
  version     show the version of LPMX

Flags:
  -h, --help   help for lpmx

Use "lpmx [command] --help" for more information about a command.
```

### 1. 'lpmx init' command is used for initializing the basic system folder of lpmx, it stores information of containers and other maintaince information(Users should call this command before using lpmx)
```
init command is the basic command of lpmx, which is used for initializing lpmx system

Usage:
  lpmx init [flags]

Flags:
  -d, --dependency string   dependency tar ball(optional)
  -h, --help                help for init
  -r, --reset               initialize by force(optional)
```
'lpmx init' supports offline initialization with dependency tar ball, i.e, 'lpmx init -d dependency.tar.gz', initializing lpmx in offline mode. 

### 2. 'lpmx docker' command is our newly added command targeting support docker images on DockerHub, by using this command, users can search, download, create and package container based on docker images.
```
docker command is the advanced comand of lpmx, which is used for executing docker related commands

Usage:
  lpmx docker [command]

Available Commands:
  add         add the local docker image to system
  commit      commit docker container
  create      initialize the local docker images
  delete      delete the local docker images
  download    download the docker images from docker hub
  list        list local docker images
  package     package the docker images from docker hub for offline usage
  reset       reset local docker base layers
  search      search the docker images from docker hub

Flags:
  -h, --help   help for docker

Use "lpmx docker [command] --help" for more information about a command.
```
### Introduction of lpmx docker sub-command
- 'lpmx docker add' -> add local image packaged viaucommand  'lpmx docker package'to  lpmx 
- 'lpmx docker commit' -> commit container to image (similar to 'docker commit')
- 'lpmx docker create' -> create contaienr based on downloaded docker images from dockerhub
- 'lpmx docker delete' -> delete local docker images
- 'lpmx docker download' -> download docker images from docker hub
- 'lpmx docker list' -> list all downloaded docker images
- 'lpmx docker package' -> package local images to tar balls
- 'lpmx docker reset' -> reset downloaded images(re-extraction)
- 'lpmx docker search' -> search images on docker hub

#### Examples for docker sub-command:
> (command) lpmx docker search ubuntu  
> (result)  
> Name: ubuntu, Available Tags: [10.04 12.04.5 12.04 12.10 13.04 13.10 14.04.1 14.04.2 14.04.3 14.04.4 14.04.5 14.04 14.10 15.04 15.10 16.04 16.10 17.04 17.10 18.04 18.10 19.04 artful-20170511.1 artful-20170601 artful-20170619 artful-20170716 artful-20170728 artful-20170826 artful-20170916 artful-20171006 artful-20171019 artful-20171116 artful-20180112 artful-20180123 artful-20180227 artful-20180412 artful-20180417 artful-20180524 artful-20180706 artful bionic-20171114 bionic-20171214 bionic-20171220 bionic-20180125 bionic-20180224 bionic-20180410 bionic-20180426 bionic-20180526 bionic-20180710 bionic-20180724.1 bionic-20180821 bionic-20181018 bionic-20181112 bionic cosmic-20180605 cosmic-20180716 cosmic-20180725 cosmic-20180821 cosmic-20180905 cosmic-20181018 cosmic-20181114 cosmic devel disco-20181112 disco latest lucid precise-20150212 precise-20150228.11 precise-20150320 precise-20150427 precise-20150528 precise-20150612 precise-20150626 precise-20150729 precise-20150813 precise-20150924 precise-20151020 precise-20151028 precise-20151208 precise-20160108 precise-20160217 precise-20160225 precise-20160303 precise-20160311 precise-20160318 precise-20160330 precise-20160425 precise-20160503 precise-20160526 precise-20160624 precise-20160707 precise-20160819 precise-20160923.1 precise-20161102 precise-20161123 precise-20161209 precise-20170214 precise-20170331 precise quantal raring rolling saucy trusty-20150218.1 trusty-20150228.11 trusty-20150320 trusty-20150427 trusty-20150528 trusty-20150612 trusty-20150630 trusty-20150730 trusty-20150806 trusty-20150814 trusty-20151001 trusty-20151009 trusty-20151021 trusty-20151028 trusty-20151208 trusty-20151218 trusty-20160119 trusty-20160217 trusty-20160226 trusty-20160302 trusty-20160315 trusty-20160317 trusty-20160323 trusty-20160405 trusty-20160412 trusty-20160424 trusty-20160503.1 trusty-20160526 trusty-20160624 trusty-20160711 trusty-20160802 trusty-20160819 trusty-20160914 trusty-20160923.1 trusty-20161006 trusty-20161101 trusty-20161123 trusty-20161214 trusty-20170119 trusty-20170214 trusty-20170330 trusty-20170602 trusty-20170620 trusty-20170719 trusty-20170728 trusty-20170817 trusty-20171117 trusty-20171207 trusty-20180112 trusty-20180123 trusty-20180302 trusty-20180412 trusty-20180420 trusty-20180531 trusty-20180712 trusty-20180807 trusty-20180929 trusty-20181115 trusty utopic-20150211 utopic-20150228.11 utopic-20150319 utopic-20150418 utopic-20150427 utopic-20150528 utopic-20150612 utopic-20150625 utopic vivid-20150218 vivid-20150309 vivid-20150319.1 vivid-20150421 vivid-20150427 vivid-20150528 vivid-20150611 vivid-20150802 vivid-20150813 vivid-20150930 vivid-20151021 vivid-20151106 vivid-20151111 vivid-20151208 vivid-20160122 vivid wily-20150528.1 wily-20150611 wily-20150708 wily-20150731 wily-20150807 wily-20150818 wily-20150829 wily-20151006 wily-20151009 wily-20151019 wily-20151208 wily-20160121 wily-20160217 wily-20160302 wily-20160316 wily-20160329 wily-20160424 wily-20160503 wily-20160526 wily-20160602 wily-20160706 wily xenial-20151218.1 xenial-20160119.1 xenial-20160125 xenial-20160217.2 xenial-20160226 xenial-20160303.1 xenial-20160314.4 xenial-20160317 xenial-20160331.1 xenial-20160422 xenial-20160503 xenial-20160525 xenial-20160629 xenial-20160706 xenial-20160713 xenial-20160809 xenial-20160818 xenial-20160914 xenial-20160923.1 xenial-20161010 xenial-20161114 xenial-20161121 xenial-20161213 xenial-20170119 xenial-20170214 xenial-20170410 xenial-20170417.1 xenial-20170510 xenial-20170517.1 xenial-20170619 xenial-20170710 xenial-20170802 xenial-20170915 xenial-20171006 xenial-20171114 xenial-20171201 xenial-20180112.1 xenial-20180123 xenial-20180228 xenial-20180412 xenial-20180417 xenial-20180525 xenial-20180705 xenial-20180726 xenial-20180808 xenial-20181005 xenial-20181113 xenial yakkety-20160708 yakkety-20160717 yakkety-20160806.1 yakkety-20160826 yakkety-20160919 yakkety-20160923.1 yakkety-20161013 yakkety-20161104 yakkety-20161121 yakkety-20161213 yakkety-20170104 yakkety-20170224 yakkety-20170327 yakkety-20170517.1 yakkety-20170619 yakkety-20170704 yakkety zesty-20161129.1 zesty-20161212 zesty-20170118 zesty-20170224 zesty-20170411 zesty-20170517.1 zesty-20170619 zesty-20170703 zesty-20170913 zesty-20170915 zesty-20171114 zesty-20171122 zesty]  
> (Note) result contains target name and available tags for this name. If name does not exist in docker hub, then error occurs.  

> (command) lpmx docker download ubuntu:16.10  
> (result)   
> Downloading file with type: application/vnd.docker.image.rootfs.diff.tar.gzip, size: 42786408, destination: /home/test/app/.docker/ubuntu/16.10/image/dca7be20e546564ad2c985dae3c8b0a259454f5637e98b59a3ca6509432ccd01
> Downloading file with type: application/vnd.docker.image.rootfs.diff.tar.gzip, size: 816, destination: /home/test/app/.docker/ubuntu/16.10/image/40bca54f5968c2bdb0d8516e6c2ca4d8f181326a06ff6efee8b4f5e1a36826b8
> Downloading file with type: application/vnd.docker.image.rootfs.diff.tar.gzip, size: 515, destination: /home/test/app/.docker/ubuntu/16.10/image/61464f23390e7d30cddfd10a22f27ae6f8f69cc4c1662af2c775f9d657266016
> Downloading file with type: application/vnd.docker.image.rootfs.diff.tar.gzip, size: 854, destination: /home/test/app/.docker/ubuntu/16.10/image/d99f0bcd5dc8b557254a1a18c6b78866b9bf460ab1bf2c73cc6aca210408dc67
> Downloading file with type: application/vnd.docker.image.rootfs.diff.tar.gzip, size: 163, destination: /home/test/app/.docker/ubuntu/16.10/image/120db6f90955814bab93a8ca1f19cbcad473fc22833f52f4d29d066135fd10b6
> INFO[0029] DONE  

> (command) lpmx docker list  
> (result)   
> Name  
> ubuntu:14.04  
> ubuntu:16.10  

> (command) lpmx docker create ubuntu:16.10  
> (result) root@ubuntu:/#  
> (Note) lpmx will automatically create and open bash shell.  

> (command) lpmx docker delete ubuntu:16.10  
> (result) INFO[0000] DONE  
> (Note) docker image is removed locally  

**NOTE**: users can always resume exited container by 'lpmx resume' command. For the id of container, users can always use 'lpmx list' command to show info of all registered containers.  

### 3. 'lpmx list' command is used for listing the information of all the registered containers, including containerid, container rpc port(NA for no rpc port)
```
list command is the basic command of lpmx, which is used for listing all the containers registered

Usage:
  lpmx list [flags]

Flags:
  -h, --help   help for list
```

### 4. 'lpmx resume' command is used for resuming stopped container, you need to use this command with container id argument
```
resume command is the basic command of lpmx, which is used for resuming the registered container via id

Usage:
  lpmx resume [flags]

Flags:
  -h, --help   help for resume
```

### 5. 'lpmx destroy' command is used for destroying container, i.e deleting configuration files
```
destroy command is the basic command of lpmx, which is used for destroying the registered container via id

Usage:
  lpmx destroy [flags]

Flags:
  -h, --help   help for destroy

Example:
  ./lpmx destroy containerid
```

### 6. 'lpmx expose' command is used for exposing applications inside containers to host, i.e, users can directly call apps inside containers from host OS. 
```
expose command is the advanced command of lpmx, which is used for exposing binaries inside containers to host

Usage:
  lpmx expose [flags]

Flags:
  -h, --help          help for expose
  -i, --id string     required
  -n, --name string   required
```

### 7. 'lpmx get' command is used for getting app settings from memcache server, values are set by using 'lpmx set' command.
```
get command is the basic command of lpmx, which is used for getting settings from cache server

Usage:
  lpmx get [flags]

Flags:
  -h, --help          help for get
  -i, --id string     required
  -n, --name string   required
```

### 8. 'lpmx set' command is used for setting environment variables for containers dynamically
```
set command is an advanced comand of lpmx, which is used for setting environment variables of running containers, you should clearly know what you want before using this command, it will reduce the performance heavily

Usage:
  lpmx set [flags]

Flags:
  -h, --help           help for set
  -i, --id string      required(container id, you can get the id by command 'lpmx list')
  -n, --name string    required(should be the name of libc 'system calls wrapper')
  -t, --type string    required('add_map','remove_map')
  -v, --value string   required(value(file1:replace_file1;file2:repalce_file2;))
``` 

### 9. Compile fakechroot and its dependencies from scratch
Precompiled fakechroot libraries are listed in this [repository](https://github.com/JasonYangShadow/LPMXSettingRepository) for different distros. However, not all distros are listed here, if you need to compile all dependencies for lpmx for your distro. Please refer this bash [script](https://github.com/JasonYangShadow/fakechroot/tree/master/shellbuild). 

Some common dependencies are required such as 
- git
- wget
- autoconf(>2.64)
- automake
- make
- gcc
- g++
- cmake
- libtool
- fakeroot
- libssl-dev
- memcached

for the case that libmemcached-dev and msgpack-c are not available on host, one may have to compile these two libraries manually. 

Package all dependencies into tar ball and initialize lpmx with it.

### About YAML configuration file
```
lpmx receive yml configuration file while creating containers

some configurations can be put in as your requirements:

**NOTE** users may not need to modify configuration files themselves in principle unless they clearly know what they want. Most items in configuration file are set and configurated by LPMX itself, any new value will overrite default one. Therefore, if any errors occur after modification, please empty configuration file and recreate containers.
```
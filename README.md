![lpmx logo](./lpmx_small.PNG)

# LPMX [![Build Status](https://travis-ci.com/JasonYangShadow/lpmx.svg?branch=master)](https://travis-ci.com/JasonYangShadow/lpmx) [![Gitter](https://badges.gitter.im/lpmx_container/community.svg)](https://gitter.im/lpmx_container/community?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge) [![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=JasonYangShadow_lpmx&metric=alert_status)](https://sonarcloud.io/dashboard?id=JasonYangShadow_lpmx) [![Coverage](https://sonarcloud.io/api/project_badges/measure?project=JasonYangShadow_lpmx&metric=coverage)](https://sonarcloud.io/dashboard?id=JasonYangShadow_lpmx) [![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=JasonYangShadow_lpmx&metric=sqale_rating)](https://sonarcloud.io/dashboard?id=JasonYangShadow_lpmx) [![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=JasonYangShadow_lpmx&metric=reliability_rating)](https://sonarcloud.io/dashboard?id=JasonYangShadow_lpmx) 
LPMX, i.e, Local Package Manager X, is a pure rootless and composable process sandbox system providing chroot-like environment. It allows users to create Docker images based containers and install packages without root/sudo privilege required. 

[![LPMX DEMO](http://img.youtube.com/vi/_1XOLa1cKX4/0.jpg)](http://www.youtube.com/watch?v=_1XOLa1cKX4 "LPMX simple demo")

# Features
1. **Pure rootless**, root privilege(root/sudo) is not required in any stages. It runs completely inside user space, which is especially suitable for creating and running software in restricted environment such as Linux cluster, grid infrastructure, batch system and etc, where root privilege is not approved.
2. **Understanding docker meta-data(Limited distros)**, LPMX could create containers via Docker images available on docker hub. Currently ubuntu and centos series are supported.
3. **Fake union file system(Fake Unionfs)**, LPMX implements its own simple rootless union file system, creating a union mount for different layers containing directories and files from different locations and forming a single coherent file system. Unlike other existing implementations, i.e, fuse, overlay and etc, our Fake Unionfs does not require any pre-installation and bring modifications to host OS.
4. **Composability**, traditional container systems(Docker, Singularity, Podman) do not provide efficient communication channels for applications running on host and containers. For example, 'app A' running inside container could not directly make a call to 'app B' running on host OS. However, LPMX is designed to provide this feature, which makes communication among applications running in different runtime environments become possible.
5. **Dynamic management of environmental variables**, LPMX allows end-users to set environment variables dynamically without stopping containers, all settings come into effect immediately.
6. **Designed for restricted runtime environment**, LPMX is designed for running containers in restricted runtime environments, such as root privilege is not approved or complete off-line usage. LPMX supports complete off-line initialization and deployment, which is especially suitable for scientific computing infrastructure.

# Examples
- Download and run newer ubuntu distros(ubuntu 14.04/16.04/18.04) on host with old OS, such as centos 6, then install and run newer software inside.
- Package and deliver current created environment to another machine.
- Run programs inside container, and then make a direct call to other programs on host.
- Install packages inside container and expose them to host.
- Open a file inside container through one unique file path, while can be mapped to any paths on host. 

# Quick Run
1. check out [release page](https://github.com/JasonYangShadow/lpmx/releases)
2. chmod a+x Linux-x86_64-lpmx && ./Linux-x86_64-lpmx init

For bash users, 'source ~/.bashrc' will add lpmx folder to PATH env

# Compile LPMX 
1. Make sure golang and [dep](https://github.com/golang/dep) are installed on your OS
2. go get -v github.com/jasonyangshadow/lpmx
3. cd $GOPATH/src/github.com/jasonyangshadow/lpmx
4. ./build.sh

If there are any dependencies issues, try to execute 'dep ensure' inside project folder and then add vendor subfolder into $GOROOT var.

# Compile Fakechroot
LPMX uses [customized fakechroot](https://github.com/jasonyangshadow/fakechroot) for trapping glibc functions(open, mkdir, symlink and etc), if you want to compile fakechroot, please refer this [Wiki](https://github.com/JasonYangShadow/lpmx/wiki#9-compile-fakechroot-and-its-dependencies-from-scratch)

# Attention

**Only several host OS and Docker images are supported currently**

For supported host OS information, please refer this repository -> [https://github.com/JasonYangShadow/LPMXSettingRepository](https://github.com/JasonYangShadow/LPMXSettingRepository)

Basically, LPMX supports centos/redhat (5.7, 6.7, 7) and ubuntu (12.04, 14.04, 16.04, 18.04, 19.04) as host OS. For containerized system, currently LPMX only supports running ubuntu (14.04/16.04).  

**Busybox is not support!**

For more information please refer project's wiki page. 

# Related Projects

- [Fakechroot](https://github.com/JasonYangShadow/fakechroot)
- [LPM](https://lpm.bio/)
- [udocker](https://github.com/indigo-dc/udocker)

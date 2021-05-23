![lpmx logo](./lpmx_small.PNG)

# LPMX [![Build Status](https://travis-ci.com/JasonYangShadow/lpmx.svg?branch=master)](https://travis-ci.com/JasonYangShadow/lpmx) [![Gitter](https://badges.gitter.im/lpmx_container/community.svg)](https://gitter.im/lpmx_container/community?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge) [![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=JasonYangShadow_lpmx&metric=sqale_rating)](https://sonarcloud.io/dashboard?id=JasonYangShadow_lpmx) [![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=JasonYangShadow_lpmx&metric=reliability_rating)](https://sonarcloud.io/dashboard?id=JasonYangShadow_lpmx) 
LPMX, i.e, Local Package Manager X, is a **pure rootless composable** container system It allows users to create container runtimes based on Docker or Singularity (Experimental) images and install packages inside without root/sudo privilege required. 

# Features
1. **Pure Rootless**, root privilege is not required at any stage, including installation, launching containers, creation of images. Users can install software inside the container without root privileges and do not need to give in to running a read-only container.
2. **Composability**, traditional container systems do not provide the composability feature, which allows executables to directly call executables in different environments, such as on the host or in the other containers. LPMX supports the composability feature. Users can easily compose existing containers and inject apps from different environments into the current runtime as if they are installed locally. Imagine that you can containerize the Canu assembler inside a container and still allows it to submit jobs via the host job submission command, e.g. qsub.
3. **Userspace Union File System(UUFS)**, LPMX implements its own simple userspance union file system to support loading layers extracted from Docker images (or other layered file system). Unlike existing implementations such as [fuse-overlayfs](https://github.com/containers/fuse-overlayfs), UUFS does not require neither newer Linux kernels nor preinstalled libraries, it purely runs in userland. The UUFS is designed to support sharing base layers among different containers so that storage space and network traffic are saved, while container launch speed is largely accelerated.
4. **Understanding existing container image meta-data(Limited distros, Alpine is not supported)**, LPMX can create containers on Docker images available on the docker hub. Currently ubuntu and centos series are supported. Besides, the latest release also has experimental support for the Singularity image.
5. **Dynamic management of environmental variables**, LPMX allows end-users to set environment variables dynamically without stopping containers, all settings come into effect immediately.
6. **Designed for restricted runtime environment**, LPMX is designed for running containers in restricted runtime environments, such as root privilege is not approved or complete off-line usage. LPMX supports complete off-line initialization and deployment, which is especially suitable for scientific computing infrastructure.
7. **Easy to access GPGPU resource**, LPMX provides end-users an easy way to access the host GPGPU resource. An example is here [https://github.com/JasonYangShadow/lpmx/wiki/GPGPU](https://github.com/JasonYangShadow/lpmx/wiki/GPGPU)

# Composability Feature
Genome analysis tools are often difficult to install due to their complex dependencies and conflicts. 
Container virtualization systems such as Dockera and Singularity can help researchers install tools by isolating tools. However, they lack **composability**, an easy way to integrate multiple tools in different containers or multiple tools in a container and a host, which was an obstacle to benefit from container systems in research. An example is that tools that require distributed computing are not straightforward to be containerized. Another example is that a pipeline container integrating different tools or versions is difficult to build from existing containers.

![composability](figures/composability.jpg)

The below video shows how to dynamically inject applications inside other LPMX containers into a current running LPMX container, you can see that even though applications, e.g. bwa, samtools, are not installed inside the currently running container, you can still inject them easily if they are already created via LPMX. This will greatly help integrate existing containers without repeated creation.

[![IMAGE ALT TEXT HERE](https://img.youtube.com/vi/kf94-rmOFYA/0.jpg)](https://www.youtube.com/watch?v=kf94-rmOFYAE)

And a gif showing injecting an exposed samtool into another container

![samtool](https://user-images.githubusercontent.com/2051711/100324168-301dd600-300a-11eb-9170-5457613b0db4.gif)

# Quick Run
1. check out [release page](https://github.com/JasonYangShadow/lpmx/releases)
2. chmod a+x Linux-x86_64-lpmx && ./Linux-x86_64-lpmx init

That's it!

For all other command details, please check [wiki](https://github.com/JasonYangShadow/lpmx/wiki)

Below is a basic demo of using LPMX:

[![LPMX DEMO](http://img.youtube.com/vi/_1XOLa1cKX4/0.jpg)](http://www.youtube.com/watch?v=_1XOLa1cKX4 "LPMX simple demo")

# Limitations
1. Only Linux(x86-64) systems are supported. (**Windows/Mac OS** are not supported)
2. **NON-GLIBC** based distros(For host OS and container images) are not supported, because our fakechroot only wraps functions inside GNU C Library(glibc), so both host OS and container images should be Glibc-based.
3. Only several host OS are supported currently in this [repository](https://github.com/JasonYangShadow/LPMXSettingRepository)(Ubuntu 12.04/14.04/16.04/18.04/19.04, Centos 5.11/6/6.7/7), we compiled fakechroot against common Linux distros, but still there might be incompatability issues among different glibc versions. Common container image types are supported, such as Ubuntu and Centos.

# Online Tutorial Session
If you are interested in LPMX and want an online tutorial session, please fill in this [Online Tutorial Request Form](https://forms.gle/6tUYdMmMSo6nDv916), I will contact you. (English will be used).:w

# Related Projects

- [Fakechroot](https://github.com/JasonYangShadow/fakechroot)
- [LPM](https://lpm.bio/)
- [udocker](https://github.com/indigo-dc/udocker)

# Acknowledgement
Computations were partially performed on the NIG supercomputer at ROIS National Institute of Genetics.

https://gc.hgc.jp

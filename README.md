![lpmx logo](./lpmx_small.PNG)

# LPMX [![Build Status](https://travis-ci.com/JasonYangShadow/lpmx.svg?branch=master)](https://travis-ci.com/JasonYangShadow/lpmx) [![Gitter](https://badges.gitter.im/lpmx_container/community.svg)](https://gitter.im/lpmx_container/community?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge) [![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=JasonYangShadow_lpmx&metric=sqale_rating)](https://sonarcloud.io/dashboard?id=JasonYangShadow_lpmx) [![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=JasonYangShadow_lpmx&metric=reliability_rating)](https://sonarcloud.io/dashboard?id=JasonYangShadow_lpmx) 
LPMX, i.e, Local Package Manager X, is a **pure rootless composable** container system. It helps researchers run genome analysis tools via existing Docker or Singularity (Experimental) images without root/sudo privilege required. Besides, researchers can benefit composability feature, e.g. call host qsub command to submit a job inside container.  

# Why should I use containers?
In the bioinformatics world, [Bioconda](https://bioconda.github.io) is the best tool for setting up genome analysis tools, but conflicting tools(requiring conflicting dependencies, e.g. Python2 & Python3) inside a genomics analysis pipeline can not be set up successfully due to the fact that conflicting dependencies can not be installed inside a single namespace. For example, [Manta](https://github.com/Illumina/manta) stills requires Python2, installing a pipeline consisting of Manta and other Python3 based tools will corrupt. Container virtualization technologies can solve this problem by isolating each tool into a container. [Singularity](https://sylabs.io/singularity/) gains so much attractiveness recently, Singularity is used widely in High-Performance Computing infrastructures and helps accelerate genomics research. 

# Why should I use LPMX?
Traditional container systems lack **composability**, which prevents tools running in different environments(host & containers) from communicating. For example, if we use Singularity to containerize the Canu tool, Canu will not be able to call host commands, e.g. call qsub to submit jobs. It can be solved only if we do modifications to either the Canu tool itself or the container engine, such as adding a network communication adapter, which is usually unachievable. 

LPMX provides both isolation and composability. LPMX can isolate tools like Singularity does but also allow tools to make direct calls to other tools at the same time. 

Besides, you can directly use existing Docker and Singularity images with LPMX without root privilege, which is safe and convenient. You can also install software inside the container as you commonly do.

# Features
1. **Pure Rootless**, root privilege is not required at any stage, including installation, launching containers, creation of images. It is suitable for Linux clusters, where users do not have root permission.
2. **Composability**, existing container systems do not allow users to compose existing containers. LPMX has composability feature. Imagine that you can containerize the Canu assembler inside a container and still allows it to submit jobs via the host job submission command, e.g. qsub.
3. **Userspace Union File System(UUFS)**, LPMX implements its own simple userspance union file system to support loading layers extracted from Docker images (or other layered file system). Unlike existing implementations such as [fuse-overlayfs](https://github.com/containers/fuse-overlayfs), UUFS does not require neither newer Linux kernels nor preinstalled libraries, it purely runs in userland. The UUFS is designed to support sharing base layers among different containers so that storage space and network traffic are saved, while container launch speed is largely accelerated.
4. **Understanding existing container image meta-data(Limited distros, Alpine is not supported)**, LPMX can create containers via Docker images available on the docker hub. Currently Ubuntu and CentOS series are supported. Besides, the latest release also has experimental support for the Singularity image.
5. **Designed for restricted runtime environment**, LPMX is designed for running containers in restricted runtime environments, such as root privilege is not approved or complete off-line usage. LPMX supports complete off-line initialization and deployment, which is especially suitable for scientific computing infrastructure.
6. **Easy to access GPGPU resource**, LPMX provides end-users an easy way to access the host GPGPU resource. An example is here [https://github.com/JasonYangShadow/lpmx/wiki/GPGPU](https://github.com/JasonYangShadow/lpmx/wiki/GPGPU)

# Quick Run
```
wget -O lpmx https://github.com/JasonYangShadow/lpmx/blob/master/build/linux/x86_64/Linux-x86_64-lpmx?raw=true

chmod a+x lpmx && ./lpmx init

#download common Linux distro from Docker hub
./lpmx docker download ubuntu:16.04

#echo hello world
./lpmx docker fastrun ubuntu:16.04 "echo 'hello world'"

#download common genomic analysis tools from Docker hub
./lpmx docker download evolbioinfo/minimap2:v2.17

#run minimap2
./lpmx docker fastrun evolbioinfo/minimap2:v2.17 "minimap2"

#run executables inside host/other containers from current container
#mapping host installed /usr/bin/vim into container /bin/vim, so that container can also run vim as if it exists, the /usr/bin/vim can also any exposed tools by LPMX
./lpmx docker fastrun -m /usr/bin/vim=/bin/vim evolbioinfo/minimap2:v2.17 "vim"
```

That's it!

For all other command details, please check [wiki](https://github.com/JasonYangShadow/lpmx/wiki)



# Composability Feature
Genome analysis tools are often difficult to install due to their complex dependencies and conflicts. 
Container virtualization systems such as Dockera and Singularity can help researchers install tools by isolating tools. However, they lack **composability**, an easy way to integrate multiple tools in different containers or multiple tools in a container and a host, which was an obstacle to benefit from container systems in research. An example is that tools that require distributed computing are not straightforward to be containerized. Another example is that a pipeline container integrating different tools or versions is difficult to build from existing containers.

![composability](figures/composability.jpg)

The below video shows how to dynamically inject applications inside other LPMX containers into a current running LPMX container, you can see that even though applications, e.g. bwa, samtools, are not installed inside the currently running container, you can still inject them easily if they are already created via LPMX. This will greatly help integrate existing containers without repeated creation.

[![IMAGE ALT TEXT HERE](https://img.youtube.com/vi/kf94-rmOFYA/0.jpg)](https://www.youtube.com/watch?v=kf94-rmOFYAE)

And a gif showing injecting an exposed samtool into another container

![samtool](https://user-images.githubusercontent.com/2051711/100324168-301dd600-300a-11eb-9170-5457613b0db4.gif)

Below is a basic demo of using LPMX:

[![LPMX DEMO](http://img.youtube.com/vi/_1XOLa1cKX4/0.jpg)](http://www.youtube.com/watch?v=_1XOLa1cKX4 "LPMX simple demo")

# Common commands
1. List existing containers with their container ids, status and other info(also with name filter)
   ```
   ./lpmx list -n name
   ```
2. Download Docker image from Docker Hub
   ```
   ./lpmx docker download ubuntu:16.04
   ```
3. Create container with binding volumes via Docker image
   ```
   ./lpmx docker create -v /host_path:/container_path -n name ubuntu:16.04
   ```
4. Delete container
   ```
   ./lpmx destroy container_id(which can be found by calling list command #1)
   ```
5. Resume container
   ```
   ./lpmx resume container_id(which can be found by calling list command #1)
   ```

# Limitations
1. Only Linux(x86-64) systems are supported. (**Windows/Mac OS** are not supported)
2. **NON-GLIBC** based distros(For host OS and container images) are not supported, because our fakechroot only wraps functions inside GNU C Library(glibc), so both host OS and container images should be Glibc-based. For example, LPMX does not support Alpine Linux
3. User can not do privileged manipulations inside containers, such as but not limited to:
   - open privileged ports (range below 1024)
   - mount file systems
   - use su command inside containers
   - change host name, system time and etc.  
4. Executables statically linked do not work properly inside containers. Recompiling them withshared libraries is a recommended workaround. Alternatively, users can install such staticallylinked executables on host and call it from inside container by exposing them by LPMX, if needed.
5. Some commands, e.g ps command, will not work as expected inside containers due to the lackof inter-process communication namespace isolation; a customized ps command wrapper cando the trick.
6. LPMX does not work with a root account; end-users should use non-privileged accounts.
7. Setuid/setgid executables do not work inside LPMX containers because LD_PRELOAD is disabled by Linux for such executables.
8. When executables uses a system call that does not exist in the host kernel, LPMX cannotexecute them. This is the common limitation of container systems.
9. **(We need supports from community!)** Only several host OS are supported currently in this [repository](https://github.com/JasonYangShadow/LPMXSettingRepository)(Ubuntu 12.04/14.04/16.04/18.04/19.04, Centos 5.11/6/6.7/7), we compiled fakechroot against common Linux distros, but still there might be incompatability issues among different glibc versions. Common container image types are supported, such as Ubuntu and CentOS. 

# Online Tutorial Session
If you are interested in LPMX and want an online tutorial session, please fill in this [Online Tutorial Request Form](https://forms.gle/6tUYdMmMSo6nDv916), I will contact you. (English will be used).:w

# Related Projects

- [Fakechroot](https://github.com/JasonYangShadow/fakechroot)
- [LPM](https://lpm.bio/)
- [udocker](https://github.com/indigo-dc/udocker)
- [Singularity](https://sylabs.io/singularity)

# Acknowledgements
- Computations were partially performed on the NIG supercomputer at ROIS National Institute of Genetics. https://gc.hgc.jp
- Supported by SHIROKANE super computing system in Human Genome Center, The Institute of Medical Science, The University of Tokyo. https://www.at.hgc.jp/
- Thanks to Department of Computational Biology and Medical Sciences, The University of Tokyo. http://www.cbms.k.u-tokyo.ac.jp/english/index.html


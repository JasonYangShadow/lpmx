![lpmx logo](./lpmx_small.PNG)
# LPMX 
LPMX, i.e, Local Package Manager X edition, is a pure rootless and composable container system.

# Feature
1. **Pure rootless**, root privilege(root/sudo) is not required in any stage. It runs completely inside user space, which is especially suiable for running containers in restricted environment such as Linux cluster, grid infrastructure, batch system and etc, where root privilege is not approved.
2. **Understanding docker metadata**, LPMX could create containers via docker images available on docker hub.
3. **Fake union file system(Fake Unionfs)**, LPMX implements its own simple rootless union file system, creating a union mount for different layers containing directories and files from differnt locations and forming a single coherent file system. Unlike existing implementations, i.e, fuse, overlay and etc, Fake Unionfs does not need pre-installation and modification on host OS.
4. **Composability**, traditonal container systems do not provide efficient communication channels for applications running on host and containers. For example, 'app A' running inside container could not directly make a call to 'app B' running on host OS. However, LPMX is designed to provide this feature, which makes communication among applications running in different runtime environments become possible.
5. **Dynamic management of environmental variables**, LPMX allows end-users to set environment variables dynamically without stopping containers, all settings come into effect immediately.
  
# Quick run
1. check out [release page](https://github.com/JasonYangShadow/lpmx/releases)
2. chmod a+x lpmx && ./lpmx init

For bash users, 'source ~/.bashrc' will add lpmx folder to PATH env

# Compile from source code 
1. Make sure golang and [dep](https://github.com/golang/dep) are installed on your OS
2. go get -v github.com/jasonyangshadow/lpmx
3. cd $GOPATH/src/github.com/jasonyangshadow/lpmx
4. ./build.sh

If there are any dependencies issues, try to execute 'dep ensure' inside project folder

# How to use it
- Download and Initialize
![Init](figures/Init.gif)
- Search docker images and Download
![DownloadImage](figures/DownloadImage.gif)
- Create containers and Management
![ContainerManagement](figures/ContainerManagement.gif)  
- Directly make a call to host application inside container
![CallToHost](figures/CallToHost.gif)
- Directly make a call to container application from host
![CallToContainer](figures/CallToContainer.gif)
- Dynamically manage environment variables to achieve advanced function

For creating containers based on cutomized content:  
1. lpmx run -c config_file_path -s target_container_root_folder -> creates and starts running container based on configuration file and target folder via terminal. 
2. lpmx run -c config_file_path -s target_container_root_folder -p -> creates and starts running container in passive mode, which will result in opening rpc port to receive commands and no interaction terminal is triggered. 

# Related projects
- [fakechroot](https://github.com/JasonYangShadow/fakechroot)
- [lpm](https://lpm.bio/)

## [Wiki](https://github.com/JasonYangShadow/lpmx/wiki) for commands details
#!/bin/bash

get(){
  if [ $1 = "GNU/Linux" ];then
    echo "installment script will create folder named lpmx in current directory"
    mkdir -p lpmx
    cd lpmx
    wget --recursive --no-parent "$3/$2"
    echo "Done! Your system is $1 and its arch is $2"
  fi
}

install(){
  SRC="https://raw.githubusercontent.com/JasonYangShadow/lpmx/master/build/linux"
  if [ -f "/usr/bin/uname" ] || [ -f "/bin/uname" ]; then
    ARCH_OS=`uname -o`
    ARCH_PLAT=`uname -m`
  else
    echo "your os doesn't have uname, it may not be compatible with this
    installment script"
    exit 1
  fi

  if [ ! -f "/usr/bin/wget" ];then
    echo "wget dees not exist in your os, please install wget"
    exit 1
  fi

  get $ARCH_OS $ARCH_PLAT $SRC
}

uninstall(){
  if [ -d "lpmx" ];then
    rm -rf lpmx
    echo "Done! You may need to delete the test folder inside /tmp manually"
    exit 0
  fi
}

test_terminal(){
  if [ -f "/usr/bin/uname" ] || [ -f "/bin/uname" ];then
    ARCH_PLAT=`uname -m`
    ./build/examples/$ARCH_PLAT/terminal/run.sh
  else
    echo "your os doesn't have uname, it may not be comaptible iwth this
    installment script"
    exit 1
  fi
}

test_rpc(){
  if [ -f "/usr/bin/uname" ] || [ -f "/bin/uname" ];then
    ARCH_PLAT=`uname -m`
    ./build/examples/$ARCH_PLAT/rpc/run.sh
  else
    echo "your os doesn't have uname, it may not be comaptible iwth this
    installment script"
    exit 1
  fi
}

main(){
  while [ "$1" != "" ];do
    case $1 in
      -t | --type )
        shift
        type=$1
        ;;
      * )
        echo "wrong input type, please use -t or --type"
        exit 1
        ;;
      esac
      shift
  done


  case $type in
    install )
      install
      ;;
    uninstall )
      uninstall
      ;;
    term )
      test_terminal
      ;;
    rpc )
      test_rpc
      ;;
    * )
      echo "please input either 'install', 'uninstall', 'term','rpc'"
      exit 1
      ;;
    esac
}

main $@

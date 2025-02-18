#!/bin/bash
# @pkg_manager: apt
# @dependencies: apt-utils
# @author: PanelBase Team
# @version: 1.0.0
# @description: 安裝指定的套件

if [ -z "$1" ]; then
    echo "請提供要安裝的套件名稱"
    exit 1
fi

apt install -y $1

#!/bin/bash
# @commands: create
# @pkg_manager: apk, apt, opkg, pacman, yum, zypper, dnf
# @dependencies: null
# @author: PanelBase Team
# @version: 1.0.0
# @description: Create specified files

[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source ~/utilkit.sh

file="*#ARG_1#*"
path="*#ARG_2#*"

if [ -n "$file" ]; then
    ADD -f $file
fi
if [ -n "$path" ]; then
    ADD -d $path
fi

#!/bin/bash
# @commands: remove
# @pkg_manager: apk, apt, opkg, pacman, yum, zypper, dnf
# @dependencies: null
# @author: PanelBase Team
# @version: 1.0.0
# @description: Remove specified packages

[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source ~/utilkit.sh
DEL *#ARG_1#*
#!/bin/bash
# @commands: clean
# @pkg_manager: apk, apt, opkg, pacman, yum, zypper, dnf
# @dependencies: null
# @author: PanelBase Team
# @version: 1.0.0
# @description: Clean system cache

[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source ~/utilkit.sh
SYS_CLEAN
#!/bin/bash
# @commands: create
# @pkg_manager: apk, apt, opkg, pacman, yum, zypper, dnf
# @dependencies: null
# @author: PanelBase Team
# @version: 1.0.0
# @description: Create specified files

[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source ~/utilkit.sh
if [ -n "*#PATH#*" ]; then
	ADD -d *#PATH#*
fi
if [ -n "*#FILE#*" ]; then
	ADD -f *#FILE#*
fi

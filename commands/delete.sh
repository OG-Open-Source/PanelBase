#!/bin/bash
# @commands: delete
# @pkg_manager: apk, apt, opkg, pacman, yum, zypper, dnf
# @dependencies: null
# @author: PanelBase Team
# @version: 1.0.0
# @description: Delete specified files

[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source ~/utilkit.sh
if [ -n "*#PATH#*" ]; then
	DEL -d *#PATH#*
fi
if [ -n "*#FILE#*" ]; then
	DEL -f *#FILE#*
fi

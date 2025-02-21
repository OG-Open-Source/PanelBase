#!/bin/bash
# @commands: run
# @pkg_manager: apk, apt, opkg, pacman, yum, zypper, dnf
# @dependencies: null
# @author: PanelBase Team
# @version: 1.0.0
# @description: Run specified command

command="*#ARG_1#*"

if [ -n "$command" ]; then
	eval "$command"
fi

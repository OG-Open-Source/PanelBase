#!/bin/bash
# @script: run
# @pkg_managers: apk, apt, opkg, pacman, yum, zypper, dnf
# @dependencies: null
# @authors: PanelBase Team
# @version: 1.0.0
# @description: Run specified command

command="*#ARG_1#*"

if [ -n "$command" ]; then
	eval '$command'
fi
#!/bin/bash
# @pkg_manager: apt
# @dependencies: apt-utils, curl
# @author: PanelBase Team
# @version: 1.0.0
# @description: 清理系統套件快取

apt autoclean
apt autoremove

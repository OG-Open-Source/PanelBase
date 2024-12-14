#!/bin/bash

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 檢查是否為 root 用戶
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}請使用 root 權限運行此腳本${NC}"
    exit 1
fi

# 停止服務
echo -e "${YELLOW}停止 Lighttpd 服務...${NC}"
systemctl stop lighttpd
systemctl disable lighttpd

# 移除安裝目錄
INSTALL_DIR="/opt/panelbase"
echo -e "${YELLOW}移除安裝目錄...${NC}"
rm -rf $INSTALL_DIR

# 詢問是否要移除 Lighttpd
read -p "是否要移除 Lighttpd？(y/n) " remove_lighttpd
if [ "$remove_lighttpd" = "y" ] || [ "$remove_lighttpd" = "Y" ]; then
    # 檢測系統類型
    if [ -f /etc/debian_version ]; then
        apt-get remove -y lighttpd
        apt-get autoremove -y
    elif [ -f /etc/redhat-release ]; then
        yum remove -y lighttpd
        yum autoremove -y
    fi
    echo -e "${GREEN}Lighttpd 已移除${NC}"
fi

echo -e "${GREEN}解除安裝完成！${NC}" 
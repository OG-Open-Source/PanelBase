#!/bin/bash

[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source ~/utilkit.sh

Authors="OGATA Open-Source"
Scripts="panelbase_setup.sh"
Version="Beta218"
License="Apache License 2.0"

REPO_URL="https://raw.githubusercontent.com/OG-Open-Source/PanelBase/refs/heads/main"

CLR1="\033[0;31m"
CLR2="\033[0;32m"
CLR3="\033[0;33m"
CLR4="\033[0;34m"
CLR5="\033[0;35m"
CLR6="\033[0;36m"
CLR7="\033[0;37m"
CLR8="\033[0;96m"
CLR9="\033[0;97m"
CLR0="\033[0m"

CHECK_ROOT

# 下載檔案函數
download_file() {
    local file_path="$1"
    local target_path="$2"
    text "${CLR2}下載 ${file_path}...${CLR0}"
    curl -sSLO "${REPO_URL}/${file_path}"
    if [ $? -eq 0 ]; then
        mkdir -p "$(dirname "${target_path}")"
        mv "$(basename ${file_path})" "${target_path}"
        chmod +x "${target_path}"
        text "${CLR2}成功下載 ${file_path}${CLR0}"
    else
        error "下載 ${file_path} 失敗"
        exit 1
    fi
}

text "${CLR2}開始安裝必要套件...${CLR0}"

# 檢測包管理器並安裝必要套件
if command -v apt-get &>/dev/null; then
    apt-get update
    apt-get install -y lighttpd xxd openssl curl
elif command -v yum &>/dev/null; then
    yum update -y
    yum install -y lighttpd vim-common openssl curl
elif command -v pacman &>/dev/null; then
    pacman -Syu --noconfirm
    pacman -S --noconfirm lighttpd vim openssl curl
else
    error "不支援的系統，請手動安裝 lighttpd"
    exit 1
fi

# 創建必要目錄
text "${CLR2}創建必要目錄...${CLR0}"
mkdir -p /var/www/panelbase/cgi-bin
mkdir -p /tmp/sessions
mkdir -p /tmp/resets
mkdir -p /etc/lighttpd/certs
chmod 700 /tmp/sessions /tmp/resets

# 生成 SSL 證書
text "${CLR2}生成 SSL 證書...${CLR0}"
DOMAIN=${1:-$(hostname)}
openssl req -x509 -newkey rsa:4096 -keyout /etc/lighttpd/certs/server.key -out /etc/lighttpd/certs/server.crt -days 365 -nodes -subj "/CN=${DOMAIN}"
cat /etc/lighttpd/certs/server.key /etc/lighttpd/certs/server.crt >/etc/lighttpd/certs/server.pem
chmod 600 /etc/lighttpd/certs/server.pem

# 下載並安裝 CGI 腳本
text "${CLR2}下載並安裝 CGI 腳本...${CLR0}"
download_file "cgi-bin/auth.cgi" "/var/www/panelbase/cgi-bin/auth.cgi"

# 配置 lighttpd
text "${CLR2}配置 lighttpd...${CLR0}"

# 下載配置文件
text "${CLR2}下載 lighttpd 配置文件...${CLR0}"
mkdir -p /etc/lighttpd/conf.d
download_file "config/lighttpd/conf.d/10-cgi.conf" "/etc/lighttpd/conf.d/10-cgi.conf"
download_file "config/lighttpd/conf.d/20-rewrite.conf" "/etc/lighttpd/conf.d/20-rewrite.conf"

# 確保 conf.d 目錄被包含
if ! grep -q "conf.d" /etc/lighttpd/lighttpd.conf; then
    echo 'include "conf.d/*.conf"' >>/etc/lighttpd/lighttpd.conf
fi

# 重啟 lighttpd
text "${CLR2}重啟 lighttpd 服務...${CLR0}"
if command -v systemctl &>/dev/null; then
    systemctl enable lighttpd
    systemctl restart lighttpd
else
    service lighttpd restart
fi

# 檢查服務狀態
if command -v systemctl &>/dev/null; then
    if systemctl is-active --quiet lighttpd; then
        text "${CLR2}安裝完成！${CLR0}"
        text "${CLR3}CGI 腳本可通過以下 URL 訪問：${CLR0}"
        text "https://${DOMAIN}/panelbase/s/login"
        text "https://${DOMAIN}/panelbase/s/check"
        text "https://${DOMAIN}/panelbase/s/logout"
        text "https://${DOMAIN}/panelbase/s/validate"
    else
        error "lighttpd 服務啟動失敗，請檢查日誌"
        exit 1
    fi
else
    text "${CLR2}安裝完成！${CLR0}"
    text "${CLR3}請手動檢查 lighttpd 服務狀態${CLR0}"
fi

# 顯示測試命令
text "\n${CLR3}測試命令示例：${CLR0}"
text "curl -k 'https://localhost/panelbase/s/login?redirect=/dashboard' -d 'username=admin&password=password'"
text "curl -k 'https://localhost/panelbase/s/check' -H 'Cookie: SESSION_ID=your_session_id'"
text "curl -k 'https://localhost/panelbase/s/validate' -H 'Cookie: SESSION_ID=your_session_id'"
text "curl -k 'https://localhost/panelbase/s/logout'"

text "\n${CLR3}注意：${CLR0}"
text "1. 使用 -k 參數是因為使用自簽名證書"
text "2. 在生產環境中，建議使用正式的 SSL 證書"
text "3. 默認使用自簽名證書，如需使用正式證書，請替換 /etc/lighttpd/certs/server.pem"

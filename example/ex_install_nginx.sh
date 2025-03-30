#!/bin/bash
# @command install_nginx
# @pkg_managers apt, yum, brew, apk
# @dependencies wget, curl
# @authors PanelBase Team
# @version 1.0.0
# @description 安裝並配置Nginx網頁伺服器
# @source_link https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main/example/ex_install_nginx.sh
# @execute_dir /tmp
# @allow_users admin

# 檢查運行用戶是否為root
if [ "$(id -u)" -ne 0 ] && [ "$(uname)" != "Darwin" ]; then
	echo "錯誤: 此腳本需要root權限運行"
	exit 1
fi

# 檢查系統類型並安裝Nginx
if [ -f /etc/debian_version ]; then
	# Debian/Ubuntu系統
	echo "檢測到Debian/Ubuntu系統，使用apt安裝Nginx..."
	apt update
	apt install -y nginx
elif [ -f /etc/redhat-release ]; then
	# CentOS/RHEL系統
	echo "檢測到CentOS/RHEL系統，使用yum安裝Nginx..."
	yum install -y epel-release
	yum install -y nginx
elif [ -f /etc/alpine-release ]; then
	# Alpine Linux
	echo "檢測到Alpine Linux系統，使用apk安裝Nginx..."
	apk update
	apk add nginx
elif [ "$(uname)" == "Darwin" ]; then
	# macOS系統
	echo "檢測到macOS系統，使用Homebrew安裝Nginx..."
	brew install nginx
else
	echo "不支持的操作系統類型"
	exit 1
fi

# 確認Nginx安裝成功
if ! command -v nginx &>/dev/null; then
	echo "錯誤: Nginx安裝失敗"
	exit 1
fi

# 創建網站配置
SITE_NAME="*#ARG_1#*"
DOMAIN_NAME="*#ARG_2#*"
WEB_ROOT="*#ARG_3#*"

# 檢查參數
if [ -z "$SITE_NAME" ] || [ -z "$DOMAIN_NAME" ] || [ -z "$WEB_ROOT" ]; then
	echo "錯誤: 缺少必要參數"
	echo "用法: install_nginx 網站名稱 域名 網站根目錄"
	exit 1
fi

# 創建網站根目錄
mkdir -p "$WEB_ROOT"

# 創建默認網頁
cat >"$WEB_ROOT/index.html" <<EOF
<!DOCTYPE html>
<html>
<head>
  <title>歡迎訪問 $DOMAIN_NAME</title>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
	body {
	  font-family: Arial, sans-serif;
	  line-height: 1.6;
	  margin: 0;
	  padding: 0;
	  color: #333;
	  background-color: #f4f4f4;
	  display: flex;
	  flex-direction: column;
	  justify-content: center;
	  align-items: center;
	  min-height: 100vh;
	  text-align: center;
	}
	.container {
	  width: 80%;
	  max-width: 800px;
	  margin: 0 auto;
	  padding: 20px;
	  background-color: white;
	  border-radius: 8px;
	  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
	}
	h1 {
	  color: #2c3e50;
	}
	.server-info {
	  background-color: #f8f9fa;
	  padding: 15px;
	  border-radius: 4px;
	  margin-top: 20px;
	}
  </style>
</head>
<body>
  <div class="container">
	<h1>歡迎訪問 $DOMAIN_NAME</h1>
	<p>如果您看到此頁面，表示Nginx已成功安裝並運行。</p>
	<div class="server-info">
	  <p>伺服器信息:</p>
	  <p>網站名稱: $SITE_NAME</p>
	  <p>域名: $DOMAIN_NAME</p>
	  <p>根目錄: $WEB_ROOT</p>
	  <p>安裝時間: $(date)</p>
	</div>
  </div>
</body>
</html>
EOF

# 設置適當的權限
chmod 755 "$WEB_ROOT"
chmod 644 "$WEB_ROOT/index.html"
if [ "$(id -u)" -eq 0 ]; then
	chown -R www-data:www-data "$WEB_ROOT" 2>/dev/null ||
		chown -R nginx:nginx "$WEB_ROOT" 2>/dev/null ||
		chown -R apache:apache "$WEB_ROOT" 2>/dev/null || true
fi

# 確定Nginx配置目錄
if [ -d "/etc/nginx/conf.d" ]; then
	NGINX_CONF_DIR="/etc/nginx/conf.d"
elif [ -d "/etc/nginx/sites-available" ]; then
	NGINX_CONF_DIR="/etc/nginx/sites-available"
elif [ -d "/usr/local/etc/nginx/servers" ]; then
	# macOS Homebrew
	NGINX_CONF_DIR="/usr/local/etc/nginx/servers"
else
	echo "無法確定Nginx配置目錄，可能需要手動完成配置"
	NGINX_CONF_DIR="/tmp"
fi

# 創建Nginx站點配置文件
CONFIG_FILE="$NGINX_CONF_DIR/$SITE_NAME.conf"
cat >"$CONFIG_FILE" <<EOF
server {
	listen 80;
	server_name $DOMAIN_NAME;
	root $WEB_ROOT;
	index index.html index.htm index.php;

	access_log /var/log/nginx/${SITE_NAME}_access.log;
	error_log /var/log/nginx/${SITE_NAME}_error.log;

	location / {
		try_files \$uri \$uri/ =404;
	}

	# 添加其他常見設置
	location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
		expires 30d;
	}

	# 禁止訪問隱藏文件
	location ~ /\. {
		deny all;
	}
}
EOF

# 如果使用sites-available/sites-enabled模式，創建符號連接
if [ -d "/etc/nginx/sites-available" ] && [ -d "/etc/nginx/sites-enabled" ]; then
	ln -sf "/etc/nginx/sites-available/$SITE_NAME.conf" "/etc/nginx/sites-enabled/$SITE_NAME.conf"
fi

# 測試Nginx配置
echo "測試Nginx配置..."
if nginx -t; then
	# 重啟Nginx服務
	echo "重啟Nginx服務..."
	if [ -f /etc/debian_version ] || [ -f /etc/redhat-release ]; then
		systemctl restart nginx
	elif [ -f /etc/alpine-release ]; then
		rc-service nginx restart
	elif [ "$(uname)" == "Darwin" ]; then
		brew services restart nginx
	else
		nginx -s reload
	fi

	echo "==============================================="
	echo "Nginx安裝完成!"
	echo "網站名稱: $SITE_NAME"
	echo "域名: $DOMAIN_NAME"
	echo "網站根目錄: $WEB_ROOT"
	echo "Nginx配置文件: $CONFIG_FILE"
	echo "==============================================="
	echo "請確保您的DNS設置指向此服務器，並且防火牆允許80和443端口訪問。"
else
	echo "Nginx配置測試失敗，請檢查配置錯誤"
	exit 1
fi

exit 0

#!/bin/bash

# 顯示橫幅
echo "================================="
echo "=  PanelBase 安裝程序 (Beta15)  ="
echo "================================="

# 檢查是否為 root 用戶
if [ "$EUID" -ne 0 ]; then
	echo "請使用 root 權限運行此腳本"
	exit 1
fi

# 設定用戶名和密碼
read -p "請輸入管理員用戶名（預設：admin）：" ADMIN_USER
ADMIN_USER=${ADMIN_USER:-admin}

while true; do
	read -s -p "請輸入管理員密碼：" ADMIN_PASS
	echo
	read -s -p "請再次輸入密碼：" ADMIN_PASS2
	echo

	if [ "$ADMIN_PASS" = "$ADMIN_PASS2" ]; then
		if [ ${#ADMIN_PASS} -lt 6 ]; then
			echo "密碼長度必須至少為 6 個字符"
		else
			break
		fi
	else
		echo "兩次輸入的密碼不一致，請重新輸入"
	fi
done

# 詢問是否使用自定義 HTML
read -p "是否使用自定義的面板頁面？(y/N) " USE_CUSTOM_HTML
USE_CUSTOM_HTML=${USE_CUSTOM_HTML:-n}

if [[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]]; then
	read -p "請輸入自定義面板頁面的路徑：" CUSTOM_HTML_PATH
	if [ ! -f "$CUSTOM_HTML_PATH" ]; then
		echo "錯誤：找不到指定的文件"
		exit 1
	fi
fi

# 檢查系統類型
if [ -f /etc/os-release ]; then
	. /etc/os-release
	OS=$NAME
else
	echo "無法確定操作系統類型"
	exit 1
fi

# 安裝必要的套件
echo "正在安裝必要的套件..."
case $OS in
	"Ubuntu"|"Debian GNU/Linux")
		apt-get update
		apt-get install -y lighttpd curl
		;;
	"CentOS Linux"|"Red Hat Enterprise Linux")
		yum install -y epel-release
		yum install -y lighttpd curl
		;;
	*)
		echo "不支援的操作系統: $OS"
		exit 1
		;;
esac

# 創建必要的目錄
echo "創建必要的目錄..."
INSTALL_DIR="/opt/panelbase"
mkdir -p $INSTALL_DIR
mkdir -p $INSTALL_DIR/www
mkdir -p $INSTALL_DIR/cgi-bin
mkdir -p $INSTALL_DIR/config
mkdir -p $INSTALL_DIR/logs

# 下載面板文件
echo "下載面板文件..."
BASE_URL="https://raw.githubusercontent.com/OG-Open-Source/PanelBase/main"

# 下載並檢查每個文件
for FILE in "src/cgi-bin/panel.cgi" "src/cgi-bin/auth.cgi" "src/cgi-bin/check_auth.cgi" "www/index.html"; do
	echo "下載 $FILE..."
	HTTP_CODE=$(curl -s -w "%{http_code}" -o "${FILE##*/}" "$BASE_URL/$FILE")
	if [ "$HTTP_CODE" != "200" ]; then
		echo "錯誤：無法下載 $FILE (HTTP 代碼: $HTTP_CODE)"
		exit 1
	fi
done

# 如果不使用自定義 HTML，則下載默認的面板頁面
if [[ ! $USE_CUSTOM_HTML =~ ^[Yy]$ ]]; then
	echo "下載 panel.html..."
	HTTP_CODE=$(curl -s -w "%{http_code}" -o "panel.html" "$BASE_URL/www/panel.html")
	if [ "$HTTP_CODE" != "200" ]; then
		echo "錯誤：無法下載面板頁面 (HTTP 代碼: $HTTP_CODE)"
		exit 1
	fi
fi

# 設置執行權限
chmod +x panel.cgi auth.cgi check_auth.cgi

# 移動文件到正確位置
mv panel.cgi auth.cgi check_auth.cgi $INSTALL_DIR/cgi-bin/
mv index.html $INSTALL_DIR/www/

# 如果使用自定義 HTML，複製自定義頁面
if [[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]]; then
	cp "$CUSTOM_HTML_PATH" "$INSTALL_DIR/www/panel.html"
else
	mv panel.html $INSTALL_DIR/www/
fi

# 配置 lighttpd
echo "配置 lighttpd..."
cat > /etc/lighttpd/lighttpd.conf << EOF
server.modules = (
	"mod_access",
	"mod_alias",
	"mod_compress",
	"mod_redirect",
	"mod_rewrite",
	"mod_cgi"
)

server.document-root = "$INSTALL_DIR/www"
server.port = 8080

# 設置用戶和組
server.username = "www-data"
server.groupname = "www-data"

# 日誌配置
server.errorlog = "$INSTALL_DIR/logs/error.log"
accesslog.filename = "$INSTALL_DIR/logs/access.log"

# 目錄訪問權限
\$HTTP["url"] =~ "^/" {
	dir-listing.activate = "disable"
}

# CGI 配置
cgi.assign = ( ".cgi" => "" )
alias.url = ( "/cgi-bin/" => "$INSTALL_DIR/cgi-bin/" )

# 允許執行 CGI
\$HTTP["url"] =~ "^/cgi-bin/" {
	cgi.assign = ( "" => "" )
}

# MIME 類型
mimetype.assign = (
	".html" => "text/html",
	".css"  => "text/css",
	".js"   => "application/javascript",
	".png"  => "image/png",
	".jpg"  => "image/jpeg",
	".gif"  => "image/gif",
	".svg"  => "image/svg+xml"
)

# URL 重寫規則
\$HTTP["url"] !~ "^/\$" {
	\$HTTP["url"] !~ "^/cgi-bin/(auth|panel)\.cgi" {
		url.rewrite-once = (
			"^/.*" => "/cgi-bin/check_auth.cgi"
		)
	}
}

# 設置默認文件
index-file.names = ( "index.html" )

# 設置文件訪問權限
static-file.exclude-extensions = ( ".cgi" )
EOF

# 創建用戶配置文件
echo "創建用戶配置..."
echo "${ADMIN_USER}:$(echo -n "${ADMIN_PASS}" | md5sum | cut -d' ' -f1)" > $INSTALL_DIR/config/users.conf
touch $INSTALL_DIR/config/sessions.conf

# 設置權限
echo "設置權限..."
# 確保 www-data 用戶存在
if ! id -u www-data >/dev/null 2>&1; then
	useradd -r -s /usr/sbin/nologin www-data
fi

# 設置目錄權限
find $INSTALL_DIR -type d -exec chmod 755 {} \;
find $INSTALL_DIR -type f -exec chmod 644 {} \;

# 設置特殊權限
chmod -R 755 $INSTALL_DIR/cgi-bin
chmod 600 $INSTALL_DIR/config/users.conf
chmod 600 $INSTALL_DIR/config/sessions.conf

# 設置所有權
chown -R www-data:www-data $INSTALL_DIR
chown -R www-data:www-data /etc/lighttpd

# 確保日誌目錄存在且具有正確的權限
mkdir -p /var/log/lighttpd
chown -R www-data:www-data /var/log/lighttpd
chmod 755 /var/log/lighttpd

# 重啟 lighttpd
echo "重啟 lighttpd 服務..."
systemctl restart lighttpd

# 檢查服務是否正常運行
if ! systemctl is-active --quiet lighttpd; then
	echo "警告：lighttpd 服務未能正常啟動"
	echo "請檢查日誌文件：$INSTALL_DIR/logs/error.log"
	exit 1
fi

# 等待服務完全啟動
sleep 2

# 檢查服務是否監聽在指定端口
if ! netstat -tuln | grep -q ":8080 "; then
	echo "警告：服務未能在 8080 端口啟動"
	echo "請檢查是否有其他服務佔用該端口"
	exit 1
fi

# 獲取服務器 IP
SERVER_IP=$(hostname -I | awk '{print $1}')

echo "================================="
echo "安裝完成！"
echo "請訪問 http://${SERVER_IP}:8080"
echo "用戶名：$ADMIN_USER"
echo "請使用您設定的密碼登入"
if [[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]]; then
	echo "已使用自定義面板頁面"
fi
echo "================================="
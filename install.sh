#!/bin/bash

[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source ~/utilkit.sh

Authors="OGATA Open-Source"
Scripts="panelbase-install.sh"
Version="Beta39"
License="Apache License 2.0"

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

CLEAN
text "${CLR3}=================================${CLR0}"
text "${CLR3}=      PanelBase  安裝程序      =${CLR0}$Version"
text "${CLR3}=================================${CLR0}"

CHECK_ROOT

INPUT "請輸入管理員用戶名：" ADMIN_NAME
ADMIN_NAME=${ADMIN_NAME:-admin}

while true; do
	read -s -p "請輸入管理員密碼：" ADMIN_PASS
	ADMIN_PASS=${ADMIN_PASS:-1917159}
	text
	read -s -p "請再次輸入密碼：" ADMIN_PASS2
	ADMIN_PASS2=${ADMIN_PASS2:-1917159}
	text

	if [ "$ADMIN_PASS" = "$ADMIN_PASS2" ]; then
		[ ${#ADMIN_PASS} -lt 6 ] && error "密碼長度必須至少為 6 個字符" || break
	else
		error "兩次輸入的密碼不一致，請重新輸入"
	fi
done

INPUT "是否使用自定義的面板頁面？(y/N) " USE_CUSTOM_HTML
USE_CUSTOM_HTML=${USE_CUSTOM_HTML:-n}

if [[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]]; then
	INPUT "請輸入自定義面板壓縮檔的路徑：" CUSTOM_ARCHIVE_PATH
	[ ! -f "$CUSTOM_ARCHIVE_PATH" ] && { error "找不到指定的壓縮檔"; exit 1; }
	FILE_EXT="${CUSTOM_ARCHIVE_PATH##*.}"
	deps=(unzip tar)
	CHECK_DEPS -a
fi

[ -f /etc/os-release ] && { source /etc/os-release; OS=$NAME; } || { error "無法確定操作系統類型"; exit 1; }

TASK "正在安裝必要的套件" "deps=(curl wget lighttpd); CHECK_DEPS -a;" true

INSTALL_DIR="/opt/panelbase"
TASK "創建必要的目錄" "ADD -d $INSTALL_DIR/{www,cgi-bin,config,logs}" true

text "下載面板文件..."
BASE_URL="https://raw.githubusercontent.com/OG-Open-Source/PanelBase/refs/heads/main"
for FILE in "src/cgi-bin/panel.cgi" "src/cgi-bin/auth.cgi" "src/cgi-bin/check_auth.cgi" "www/index.html"; do
	text "下載 $FILE..."
	HTTP_CODE=$(curl -s -w "%{http_code}" -o "${FILE##*/}" "$BASE_URL/$FILE")
	[ "$HTTP_CODE" != "200" ] && { error "無法下載 $FILE (HTTP 代碼: $HTTP_CODE)"; exit 1; }
done

if [[ ! $USE_CUSTOM_HTML =~ ^[Yy]$ ]]; then
	text "下載 panel.html..."
	HTTP_CODE=$(curl -s -w "%{http_code}" -o "panel.html" "$BASE_URL/www/panel.html")
	[ "$HTTP_CODE" != "200" ] && { error "無法下載面板頁面 (HTTP 代碼: $HTTP_CODE)"; exit 1; }
fi

chmod +x panel.cgi auth.cgi check_auth.cgi

mv panel.cgi auth.cgi check_auth.cgi $INSTALL_DIR/cgi-bin/
mv index.html $INSTALL_DIR/www/

if [[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]]; then
	text "正在處理自定義面板文件..."
	TMP_DIR=$(mktemp -d)
	text "臨時目錄：$TMP_DIR"
	
	case "$FILE_EXT" in
		"zip")
			text "解壓縮 ZIP 文件..."
			unzip -q "$CUSTOM_ARCHIVE_PATH" -d "$TMP_DIR"
			;;
		"tar")
			text "解壓縮 TAR 文件..."
			tar xf "$CUSTOM_ARCHIVE_PATH" -C "$TMP_DIR"
			;;
		"gz"|"tgz")
			text "解壓縮 GZIP 文件..."
			tar xzf "$CUSTOM_ARCHIVE_PATH" -C "$TMP_DIR"
			;;
	esac
	
	text "解壓縮後的文件列表："
	ls -la "$TMP_DIR"
	
	PANEL_HTML=$(find "$TMP_DIR" -name "panel.html" -type f)
	
	if [ -z "$PANEL_HTML" ]; then
		text "${CLR3}錯誤：在壓縮檔中找不到 panel.html 文件${CLR0}"
		text "${CLR3}請確保文件名稱正確（區分大小寫）${CLR0}"
		rm -rf "$TMP_DIR"
		exit 1
	else
		text "找到 panel.html：$PANEL_HTML"
		cp -f "$PANEL_HTML" "$INSTALL_DIR/www/panel.html"
		
		PANEL_DIR=$(dirname "$PANEL_HTML")
		
		for file in "$PANEL_DIR"/*; do
			if [ -f "$file" ] && [ "$(basename "$file")" != "index.html" ] && [ "$(basename "$file")" != "panel.html" ]; then
				cp -f "$file" "$INSTALL_DIR/www/"
			fi
		done
		
		for dir in "$PANEL_DIR"/*; do
			[ -d "$dir" ] && cp -rf "$dir" "$INSTALL_DIR/www/"
		done
	fi
	
	text "安裝目錄文件列表："
	ls -la "$INSTALL_DIR/www/"
	
	rm -rf "$TMP_DIR"
	
	text "${CLR2}自定義面板文件安裝完成${CLR0}"
else
	mv panel.html $INSTALL_DIR/www/
fi

text "配置 lighttpd..."
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

server.username = "www-data"
server.groupname = "www-data"

server.errorlog = "$INSTALL_DIR/logs/error.log"
accesslog.filename = "$INSTALL_DIR/logs/access.log"

\$HTTP["url"] =~ "^/" {
	dir-listing.activate = "disable"
}

cgi.assign = ( ".cgi" => "" )
alias.url = ( "/cgi-bin/" => "$INSTALL_DIR/cgi-bin/" )

\$HTTP["url"] =~ "^/cgi-bin/" {
	cgi.assign = ( "" => "" )
}

mimetype.assign = (
	".html" => "text/html",
	".css"  => "text/css",
	".js"   => "application/javascript",
	".png"  => "image/png",
	".jpg"  => "image/jpeg",
	".gif"  => "image/gif",
	".svg"  => "image/svg+xml"
)

\$HTTP["url"] !~ "^/\$" {
	\$HTTP["url"] !~ "^/cgi-bin/(auth|panel)\.cgi" {
		url.rewrite-once = (
			"^/.*" => "/cgi-bin/check_auth.cgi"
		)
	}
}

index-file.names = ( "index.html" )

static-file.exclude-extensions = ( ".cgi" )
EOF

text "創建用戶配置..."
text "${ADMIN_NAME}:$(echo -n "${ADMIN_PASS}" | md5sum | cut -d' ' -f1)" > $INSTALL_DIR/config/users.conf
touch $INSTALL_DIR/config/sessions.conf

text "設置權限..."
if ! id -u www-data >/dev/null 2>&1; then
	useradd -r -s /usr/sbin/nologin www-data
fi

find $INSTALL_DIR -type d -exec chmod 755 {} \;
find $INSTALL_DIR -type f -exec chmod 644 {} \;

chmod -R 755 $INSTALL_DIR/cgi-bin
chmod 600 $INSTALL_DIR/config/users.conf
chmod 600 $INSTALL_DIR/config/sessions.conf

chown -R www-data:www-data $INSTALL_DIR
chown -R www-data:www-data /etc/lighttpd

mkdir -p /var/log/lighttpd
chown -R www-data:www-data /var/log/lighttpd
chmod 755 /var/log/lighttpd

TASK "重啟 lighttpd 服務" "systemctl restart lighttpd" true

if ! systemctl is-active --quiet lighttpd; then
	text "${CLR3}警告：lighttpd 服務未能正常啟動${CLR0}"
	text "請檢查日誌文件：$INSTALL_DIR/logs/error.log"
	exit 1
fi

sleep 2

if ! netstat -tuln | grep -q ":8080 "; then
	text "${CLR3}警告：服務未能在 8080 端口啟動${CLR0}"
	text "請檢查是否有其他服務佔用該端口"
	exit 1
fi

SERVER_IP=$(hostname -I | awk '{print $1}')

text "================================="
text "安裝完成！"
text "請訪問 http://${SERVER_IP}:8080"
text "用戶：$ADMIN_NAME"
text "密碼：$ADMIN_PASS"
[[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]] && text "已使用自定義面板頁面"
text "================================="
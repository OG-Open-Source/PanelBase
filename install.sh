#!/bin/bash

[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source ~/utilkit.sh

Authors="OGATA Open-Source"
Scripts="panelbase-install.sh"
Version="Beta110"
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

DIR_MODE="755"
WWW_MODE="644"
CGI_MODE="755"
CONFIG_MODE="600"

declare -A FILES=(
	["cgi-bin"]="auth.cgi check_auth.cgi panel.cgi"
	["www"]="index.html 404.html 403.html panel.html"
	["config"]="routes.conf"
)

declare -A FILE_PERMISSIONS=(
	["cgi-bin"]="$CGI_MODE"
	["www"]="$WWW_MODE"
	["config"]="$CONFIG_MODE"
)

CLEAN
text "${CLR3}=================================${CLR0}"
text "${CLR3}=      PanelBase  安裝程序      =${CLR0}$Version"
text "${CLR3}=================================${CLR0}"

CHECK_ROOT

INPUT "請輸入管理員用戶名：" ADMIN_NAME
ADMIN_NAME=${ADMIN_NAME:-admin}

if ! [[ $ADMIN_NAME =~ ^[A-Za-z0-9]+$ ]]; then
	error "用戶名只能包含英文字母和數字"
	exit 1
fi

while true; do
	read -s -p "請輸入管理員密碼：" ADMIN_PASS
	ADMIN_PASS=${ADMIN_PASS:-1917159}
	text
	read -s -p "請再次輸入密碼：" ADMIN_PASS2
	ADMIN_PASS2=${ADMIN_PASS2:-1917159}
	text

	if [ "$ADMIN_PASS" = "$ADMIN_PASS2" ]; then
		if [ ${#ADMIN_PASS} -lt 6 ]; then
			error "密碼長度必須至少為 6 個字符"
			continue
		fi

		if ! [[ $ADMIN_PASS =~ ^[A-Za-z0-9!@$]+$ ]]; then
			error "密碼只能包含英文字母、數字和特殊符號 !@$"
			continue
		fi

		break
	else
		error "兩次輸入的密碼不一致，請重新輸入"
	fi
done

while true; do
	INPUT "請輸入面板端口 (1024-65535，預設: 8080)：" PANEL_PORT
	PANEL_PORT=${PANEL_PORT:-8080}

	if ! [[ $PANEL_PORT =~ ^[0-9]+$ ]]; then
		error "端口必須是數字"
		continue
	fi

	if [ $PANEL_PORT -lt 1024 ] || [ $PANEL_PORT -gt 65535 ]; then
		error "端口必須在 1024-65535 之間"
		continue
	fi

	if netstat -tuln | grep -q ":$PANEL_PORT "; then
		error "端口 $PANEL_PORT 已被占用"
		continue
	fi

	break
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
TMP_DIR=$(mktemp -d)

download_files() {
	local dir="$1"
	local files="$2"
	local target_dir="$INSTALL_DIR/$dir"

	for file in $files; do
		text "下載 $dir/$file..."
		local source_url="$BASE_URL/src/$dir/$file"
		[ "$dir" = "www" ] && source_url="$BASE_URL/$dir/$file"

		if ! curl -sSL -o "$TMP_DIR/$file" "$source_url"; then
			error "無法下載 $file"
			return 1
		fi

		chmod ${FILE_PERMISSIONS[$dir]} "$TMP_DIR/$file"
	done

	mv $TMP_DIR/* "$target_dir/"
	return 0
}

for dir in "${!FILES[@]}"; do
	if [[ "$dir" = "www" ]] && [[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]] && [[ "${FILES[$dir]}" =~ "panel.html" ]]; then
		FILES[$dir]=${FILES[$dir]/panel.html/}
	fi

	if [ -n "${FILES[$dir]}" ]; then
		if ! download_files "$dir" "${FILES[$dir]}"; then
			rm -rf "$TMP_DIR"
			error "安裝失敗"
			exit 1
		fi
	fi
done

rm -rf "$TMP_DIR"

if [[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]]; then
	text "正在處理自定義面板文件..."
	TMP_DIR=$(mktemp -d)
	text "臨時目錄：$TMP_DIR"

	case "${CUSTOM_ARCHIVE_PATH##*.}" in
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

	PANEL_HTML=$(find "$TMP_DIR" -name "panel.html" -type f)

	if [ -z "$PANEL_HTML" ]; then
		error "在壓縮檔中找不到 panel.html 文件"
		error "請確保文件名稱正確（區分大小寫）"
		rm -rf "$TMP_DIR"
		exit 1
	fi

	PANEL_DIR=$(dirname "$PANEL_HTML")
	cp -f "$PANEL_HTML" "$INSTALL_DIR/www/panel.html"

	find "$PANEL_DIR" -type f ! -name "panel.html" ! -name "index.html" -exec cp -f {} "$INSTALL_DIR/www/" \;
	find "$PANEL_DIR" -type d ! -path "$PANEL_DIR" -exec cp -rf {} "$INSTALL_DIR/www/" \;

	rm -rf "$TMP_DIR"
	text "${CLR2}自定義面板文件安裝完成${CLR0}"
fi

text "配置 lighttpd..."
cat > /etc/lighttpd/lighttpd.conf << EOF
server.modules = (
	"mod_access",
	"mod_alias",
	"mod_compress",
	"mod_redirect",
	"mod_rewrite",
	"mod_cgi",
	"mod_accesslog"
)

server.document-root = "$INSTALL_DIR/www"
server.port = $PANEL_PORT

server.username = "www-data"
server.groupname = "www-data"

server.errorlog = "$INSTALL_DIR/logs/error.log"
accesslog.filename = "$INSTALL_DIR/logs/access.log"
accesslog.format = "%h %V %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\""

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
	".jpeg" => "image/jpeg",
	".gif"  => "image/gif",
	".ico"  => "image/x-icon",
	".svg"  => "image/svg+xml",
	".woff" => "font/woff",
	".woff2" => "font/woff2",
	".ttf"  => "font/ttf",
	".eot"  => "application/vnd.ms-fontobject"
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
text "${ADMIN_NAME}:$(echo -n "${ADMIN_PASS}" | md5sum | cut -d' ' -f1)" > $INSTALL_DIR/config/user.conf
touch $INSTALL_DIR/config/sessions.conf

text "創建安全配置..."
cat > $INSTALL_DIR/config/security.conf << EOF
# PanelBase Security Configuration

# Basic Settings
INSTALL_DIR="/opt/panelbase"
DOCUMENT_ROOT="/opt/panelbase/www"

# Session Settings
SESSION_LIFETIME=86400
SESSION_ROTATION_INTERVAL=3600

# Security Restrictions
MAX_LOGIN_ATTEMPTS=5
LOGIN_BLOCK_TIME=300
PASSWORD_MIN_LENGTH=6

# File Access Control
ACCESS_CONTROL_MODE="whitelist"

# Whitelist: Active when ACCESS_CONTROL_MODE="whitelist"
# Format: Space-separated list of file patterns, supports wildcards
WHITELIST_FILES="*.html *.htm"

# Blacklist: Active when ACCESS_CONTROL_MODE="blacklist"
# Format: Space-separated list of file patterns, supports wildcards
BLACKLIST_FILES="*.css *.js *.json *.xml *.txt *.md *.csv *.sql *.sh *.conf"

# Allow access to restricted files when referenced from HTML
ALLOW_HTML_REFERENCE=false

# Security Headers Configuration
SECURITY_HEADERS_CSP="default-src 'self' https://cdnjs.cloudflare.com; \
script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdnjs.cloudflare.com; \
style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; \
img-src 'self' data: https:; \
font-src 'self' https://cdnjs.cloudflare.com; \
frame-ancestors 'none'; \
form-action 'self'; \
base-uri 'self'"

# Cache Control
CACHE_MAX_AGE=31536000

# Logging Configuration
LOG_FILE="/opt/panelbase/logs/auth.log"
ERROR_LOG_FILE="/opt/panelbase/logs/error.log"

# File Permission Settings
CONFIG_FILE_MODE=600
CGI_FILE_MODE=755
WWW_FILE_MODE=644
DIR_MODE=755

# System User Settings
WEB_USER="www-data"
WEB_GROUP="www-data"
EOF

text "設置權限..."
if ! id -u www-data >/dev/null 2>&1; then
	useradd -r -s /usr/sbin/nologin www-data
fi

find "$INSTALL_DIR" -type d -exec chmod "$DIR_MODE" {} \;

for dir in "${!FILE_PERMISSIONS[@]}"; do
	if [ -d "$INSTALL_DIR/$dir" ]; then
		find "$INSTALL_DIR/$dir" -type f -exec chmod "${FILE_PERMISSIONS[$dir]}" {} \;
	fi
done

chmod "$CONFIG_MODE" "$INSTALL_DIR/config/user.conf"
chmod "$CONFIG_MODE" "$INSTALL_DIR/config/sessions.conf"
chmod "$CONFIG_MODE" "$INSTALL_DIR/config/security.conf"

chown -R www-data:www-data "$INSTALL_DIR"
chown -R www-data:www-data /etc/lighttpd

mkdir -p /var/log/lighttpd
chown -R www-data:www-data /var/log/lighttpd
chmod "$DIR_MODE" /var/log/lighttpd

TASK "重啟 lighttpd 服務" "systemctl restart lighttpd" true

if ! systemctl is-active --quiet lighttpd; then
	error "lighttpd 服務未能正常啟動"
	error "請檢查日誌文件：$INSTALL_DIR/logs/error.log"
	exit 1
fi

SERVER_IP=$(hostname -I | awk '{print $1}')

text "================================="
text "安裝完成！"
text "請訪問 http://${SERVER_IP}:$PANEL_PORT"
text "用戶：$ADMIN_NAME"
text "密碼：$ADMIN_PASS"
[[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]] && text "已使用自定義面板頁面"
text "================================="
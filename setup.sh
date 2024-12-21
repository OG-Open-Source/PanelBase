#!/bin/bash

[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source ~/utilkit.sh

Authors="OGATA Open-Source"
Scripts="panelbase-install.sh"
Version="Beta138"
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
	["www"]="index.html 403.html 404.html panel.html favicon.png"
	["config"]="routes.conf security.conf"
)

declare -A FILE_PERMISSIONS=(
	["cgi-bin"]="$CGI_MODE"
	["www"]="$WWW_MODE"
	["config"]="$CONFIG_MODE"
)

declare -A FILE_PATHS=(
	["cgi-bin"]="src/cgi-bin"
	["www"]="www"
	["config"]="config"
)

CLEAN
text "╭────────────────────────────────╮"
text "│  ${CLR3}$Version${CLR0}"
text "│────────────────────────────────╮"
text "│                                │"
text "│       PanelBase 安裝程序       │"
text "│                                │"
text "╰────────────────────────────────╯"

CHECK_ROOT

text "${CLR8}►${CLR0} 基本設置"
text "  ──────────────"
INPUT "  管理員用戶名: " ADMIN_NAME
ADMIN_NAME=${ADMIN_NAME:-admin}

if ! [[ $ADMIN_NAME =~ ^[A-Za-z0-9]+$ ]]; then
	error "  用戶名只能包含英文字母和數字"
	exit 1
fi

while true; do
	read -s -p "  管理員密碼: " ADMIN_PASS
	ADMIN_PASS=${ADMIN_PASS:-1917159}
	echo
	read -s -p "  確認密碼: " ADMIN_PASS_CONFIRM
	ADMIN_PASS_CONFIRM=${ADMIN_PASS_CONFIRM:-1917159}
	echo
	[ "$ADMIN_PASS" = "$ADMIN_PASS_CONFIRM" ] && break
	error "  密碼不匹配，請重試"
done

while true; do
	INPUT "  請輸入面板端口 (1024-65535)：" PANEL_PORT
	PANEL_PORT=${PANEL_PORT:-8080}

	if ! [[ $PANEL_PORT =~ ^[0-9]+$ ]]; then
		error "  端口必須是數字"
		continue
	fi

	if [ $PANEL_PORT -lt 1024 ] || [ $PANEL_PORT -gt 65535 ]; then
		error "  端口必須在 1024-65535 之間"
		continue
	fi

	if netstat -tuln | grep -q ":$PANEL_PORT "; then
		error "  端口 $PANEL_PORT 已被占用"
		continue
	fi

	break
done

INPUT "  是否使用自定義的面板頁面？(y/N) " USE_CUSTOM_HTML
USE_CUSTOM_HTML=${USE_CUSTOM_HTML:-n}

if [[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]]; then
	INPUT "    請輸入自定義面板壓縮檔的路徑：" CUSTOM_ARCHIVE_PATH
	[ ! -f "$CUSTOM_ARCHIVE_PATH" ] && { error "  找不到指定的壓縮檔"; exit 1; }
	FILE_EXT="${CUSTOM_ARCHIVE_PATH##*.}"
	deps=(unzip tar)
	CHECK_DEPS -a
fi

[ -f /etc/os-release ] && { source /etc/os-release; OS=$NAME; } || { error "  無法確定操作系統類型"; exit 1; }

TASK "  正在安裝必要的套件" "deps=(curl wget lighttpd expect); CHECK_DEPS -a;"

INSTALL_DIR="/opt/panelbase"
TASK "  創建必要的目錄" "ADD -d $INSTALL_DIR/{www,cgi-bin,config,logs}"

text "  下載面板文件..."
BASE_URL="https://raw.githubusercontent.com/OG-Open-Source/PanelBase/refs/heads/main"
TMP_DIR=$(mktemp -d)

download_files() {
	local dir="$1"
	local files="$2"
	local target_dir="$INSTALL_DIR/$dir"
	local source_path="${FILE_PATHS[$dir]}"

	for file in $files; do
		text "  下載 $dir/$file..."
		local source_url="$BASE_URL/$source_path/$file"

		if ! curl -sSL -o "$TMP_DIR/$file" "$source_url"; then
			error "  無法下載 $file"
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
			error "  安裝失敗"
			exit 1
		fi
	fi
done

rm -rf "$TMP_DIR"

if [[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]]; then
	text "  正在處理自定義面板文件..."
	TMP_DIR=$(mktemp -d)
	text "  臨時目錄：$TMP_DIR"

	case "${CUSTOM_ARCHIVE_PATH##*.}" in
		"zip")
			text "  解壓縮 ZIP 文件..."
			unzip -q "$CUSTOM_ARCHIVE_PATH" -d "$TMP_DIR"
			;;
		"tar")
			text "  解壓縮 TAR 文件..."
			tar xf "$CUSTOM_ARCHIVE_PATH" -C "$TMP_DIR"
			;;
		"gz"|"tgz")
			text "  解壓縮 GZIP 文件..."
			tar xzf "$CUSTOM_ARCHIVE_PATH" -C "$TMP_DIR"
			;;
	esac

	PANEL_HTML=$(find "$TMP_DIR" -name "panel.html" -type f)

	if [ -z "$PANEL_HTML" ]; then
		error "  在壓縮檔中找不到 panel.html 文件"
		error "  請確保文件名稱正確（區分大小寫）"
		rm -rf "$TMP_DIR"
		exit 1
	fi

	PANEL_DIR=$(dirname "$PANEL_HTML")
	cp -f "$PANEL_HTML" "$INSTALL_DIR/www/panel.html"

	find "$PANEL_DIR" -type f ! -name "panel.html" ! -name "index.html" -exec cp -f {} "$INSTALL_DIR/www/" \;
	find "$PANEL_DIR" -type d ! -path "$PANEL_DIR" -exec cp -rf {} "$INSTALL_DIR/www/" \;

	rm -rf "$TMP_DIR"
	text "  ${CLR2}自定義面板文件安裝完成${CLR0}"
fi

text "  配置 lighttpd..."
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
	setenv.add-response-header = (
		"Cache-Control" => "no-cache",
		"X-Accel-Buffering" => "no"
	)
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

TASK "  創建用戶配置" "echo '${ADMIN_NAME}:$(echo -n "${ADMIN_PASS}" | md5sum | cut -d' ' -f1)' > $INSTALL_DIR/config/user.conf && touch $INSTALL_DIR/config/sessions.conf"

TASK "  設置權限" "if ! id -u www-data >/dev/null 2>&1; then useradd -r -s /usr/sbin/nologin www-data; fi"
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

TASK "  配置 sudo 權限" "cat > /etc/sudoers.d/panelbase << EOF
www-data ALL=(ALL) NOPASSWD: ALL
EOF
chmod 440 /etc/sudoers.d/panelbase"

TASK "  重啟 lighttpd 服務" "systemctl restart lighttpd"

SERVER_IP=$(hostname -I | awk '{print $1}')
text "╭────────────────────────────────╮"
text "│            安裝完成            │"
text "╰────────────────────────────────╯"
text "${CLR8}►${CLR0} 訪問地址: ${GREEN}http://${SERVER_IP}:${PANEL_PORT}${CLR0}"
text "${CLR8}►${CLR0} 登入信息:"
text "  - 用戶: ${GREEN}${ADMIN_NAME}${CLR0}"
text "  - 密碼: ${GREEN}${ADMIN_PASS}${CLR0}"
[[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]] && text "${CLR8}►${CLR0} 使用自定義面板頁面"
#!/bin/bash

[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source ~/utilkit.sh

CHECK_ROOT

# 基礎 URL
BASE_URL="https://raw.githubusercontent.com/OG-Open-Source/PanelBase/refs/heads/main"

# 檢查必要命令
deps=(curl jq)
CHECK_DEPS

# 安裝必要套件
text "正在安裝必要套件..."
ADD lighttpd python3 python3-pip curl jq

# 創建安裝目錄結構
INSTALL_DIR="/opt/panelbase"
TASK "創建安裝目錄結構" "ADD -d $INSTALL_DIR/{config,cgi-bin,www/{templates,assets},logs}"
text "下載必要文件..."

# 下載 CGI 腳本
download_files() {
	target_dir=$1
	shift
	files=("$@")
	for file in "${files[@]}"; do
		text "下載 $file..."
		GET "${BASE_URL}/${file}" $target_dir && chmod 755 $target_dir/$(basename $file)
	done
}

# 下載 CGI 腳本
CGI_FILES=(
	"src/cgi-bin/auth.cgi"
	"src/cgi-bin/api.cgi"
	"src/cgi-bin/example.py"
)
download_files "$INSTALL_DIR/cgi-bin" "${CGI_FILES[@]}"

# 下載模板文件
download_files "$INSTALL_DIR/www/templates" "src/www/templates/panel.md"

# 配置 Lighttpd
LIGHTTPD_CONF="/etc/lighttpd/lighttpd.conf"
text "${CLR3}配置 Lighttpd...${CLR0}"
cat > $LIGHTTPD_CONF << EOL
server.modules = (
	"mod_access",
	"mod_alias",
	"mod_compress",
	"mod_redirect",
	"mod_cgi"
)

server.document-root        = "$INSTALL_DIR/www"
server.upload-dirs          = ( "/var/cache/lighttpd/uploads" )
server.errorlog            = "$INSTALL_DIR/logs/error.log"
server.pid-file            = "/var/run/lighttpd.pid"
server.username            = "www-data"
server.groupname           = "www-data"
server.port                = 8080

index-file.names           = ( "index.html" )
url.access-deny           = ( "~", ".inc" )

# CGI 配置
cgi.assign = (
	".sh"  => "/bin/bash",
	".py"  => "/usr/bin/python3",
	".pl"  => "/usr/bin/perl",
	".rb"  => "/usr/bin/ruby",
	".cgi" => ""
)

# 允許執行所有 CGI 腳本
\$HTTP["url"] =~ "^/cgi-bin/" {
	cgi.assign = (
		""  => ""
	)
}

alias.url = (
	"/cgi-bin/" => "$INSTALL_DIR/cgi-bin/"
)

# MIME 類型設置
mimetype.assign = (
	".html" => "text/html",
	".txt"  => "text/plain",
	".css"  => "text/css",
	".js"   => "application/javascript",
	".json" => "application/json",
	".xml"  => "application/xml"
)

# 設置目錄權限
static-file.exclude-extensions = ( ".py", ".pl", ".rb", ".sh", ".cgi" )
EOL

# 設置權限
text "設置權限..."
chown -R www-data:www-data $INSTALL_DIR
chmod -R 755 $INSTALL_DIR
chmod -R 755 $INSTALL_DIR/cgi-bin

# 啟動服務
text "啟動 Lighttpd 服務..."
systemctl enable lighttpd
systemctl restart lighttpd

# 設置管理員帳號
text "設置管理員帳號..."
INPUT "請輸入管理員用戶名: " admin_user
INPUT "請輸入管理員密碼: " admin_pass
text "\n$admin_user:$(text -n "$admin_pass" | sha256sum | cut -d' ' -f1)" > $INSTALL_DIR/config/admin.conf

text "安裝完成！"
text "請訪問 http://your-server-ip:8080 來訪問面板"

text "安裝資訊："
text "安裝目錄：\t$INSTALL_DIR"
text "CGI 目錄：\t$INSTALL_DIR/cgi-bin"
text "網站根目錄：\t$INSTALL_DIR/www"
text "日誌目錄：\t$INSTALL_DIR/logs"
text "配置文件：\t$LIGHTTPD_CONF"
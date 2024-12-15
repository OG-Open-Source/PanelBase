#!/bin/bash

cd ~ && clear
echo "================================="
echo "=  PanelBase 安裝程序 (Beta33)  ="
echo "================================="

# 檢查是否為 root 用戶
if [ "$EUID" -ne 0 ]; then
	echo "請使用 root 權限運行此腳本"
	exit 1
fi

# 設定用戶名和密碼
read -p "請輸入管理員用戶名：" ADMIN_USER
ADMIN_USER=${ADMIN_USER:-admin}

while true; do
	read -s -p "請輸入管理員密碼：" ADMIN_PASS
	ADMIN_PASS=${ADMIN_PASS:-1917159}
	echo
	read -s -p "請再次輸入密碼：" ADMIN_PASS2
	ADMIN_PASS2=${ADMIN_PASS2:-1917159}
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
	read -p "請輸入自定義面板壓縮檔的路徑：" CUSTOM_ARCHIVE_PATH
	if [ ! -f "$CUSTOM_ARCHIVE_PATH" ]; then
		echo "錯誤：找不到指定的壓縮檔"
		exit 1
	fi

	# 檢查壓縮檔格式
	FILE_EXT="${CUSTOM_ARCHIVE_PATH##*.}"
	case "$FILE_EXT" in
		"zip")
			if ! command -v unzip >/dev/null 2>&1; then
				echo "正在安裝 unzip..."
				case $OS in
					"Ubuntu"|"Debian GNU/Linux")
						apt-get install -y unzip
						;;
					"CentOS Linux"|"Red Hat Enterprise Linux")
						yum install -y unzip
						;;
				esac
			fi
			;;
		"tar"|"gz"|"tgz") ;;
		*)
			echo "錯誤：不支援的壓縮檔格式。請使用 zip、tar 或 tar.gz 格式"
			exit 1
			;;
	esac
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
mkdir -p $INSTALL_DIR/{www,cgi-bin,config,logs,cache,static}

# 下載面板文件
echo "下載面板文件..."
BASE_URL="https://raw.githubusercontent.com/OG-Open-Source/PanelBase/refs/heads/main"
FILES=(
	"src/cgi-bin/panel.cgi"
	"src/cgi-bin/auth.cgi"
	"src/cgi-bin/check_auth.cgi"
	"src/cgi-bin/common.cgi"
	"www/index.html"
	"config/routes.conf"
)

download_file() {
	local file="$1"
	local dest_dir="$INSTALL_DIR/$(dirname $file)"
	local dest_file="$dest_dir/$(basename $file)"
	
	mkdir -p "$dest_dir"
	echo "下載 $file..."
	if ! curl -sSL -o "$dest_file" "$BASE_URL/src/$file"; then
		echo "錯誤：無法下載 $file"
		return 1
	fi
	
	if [ ! -f "$dest_file" ]; then
		echo "錯誤：文件 $dest_file 下載失敗"
		return 1
	fi
	
	# 檢查文件大小
	if [ ! -s "$dest_file" ]; then
		echo "錯誤：文件 $dest_file 為空"
		return 1
	fi
	
	return 0
}

# 下載所有文件
for file in "${FILES[@]}"; do
	if ! download_file "$file"; then
		echo "安裝失敗：無法下載必要文件"
		exit 1
	fi
done

# 如果使用自定義 HTML，解壓縮並複製文件
if [[ $USE_CUSTOM_HTML =~ ^[Yy]$ ]]; then
	echo "正在處理自定義面板文件..."
	TMP_DIR=$(mktemp -d)
	echo "臨時目錄：$TMP_DIR"
	
	case "$FILE_EXT" in
		"zip")
			echo "解壓縮 ZIP 文件..."
			unzip -q "$CUSTOM_ARCHIVE_PATH" -d "$TMP_DIR"
			;;
		"tar")
			echo "解壓縮 TAR 文件..."
			tar xf "$CUSTOM_ARCHIVE_PATH" -C "$TMP_DIR"
			;;
		"gz"|"tgz")
			echo "解壓縮 GZIP 文件..."
			tar xzf "$CUSTOM_ARCHIVE_PATH" -C "$TMP_DIR"
			;;
	esac
	
	# 列出解壓後的文件
	echo "解壓縮後的文件列表："
	ls -la "$TMP_DIR"
	
	# 遞迴搜索 panel.html
	PANEL_HTML=$(find "$TMP_DIR" -name "panel.html" -type f)
	
	if [ -z "$PANEL_HTML" ]; then
		echo "錯誤：在壓縮檔中找不到 panel.html 文件"
		echo "請確保文件名稱正確（區分大小寫）"
		rm -rf "$TMP_DIR"
		exit 1
	else
		echo "找到 panel.html：$PANEL_HTML"
		# 如果 panel.html 不在頂層目錄，將其移動到頂層
		if [ "$(dirname "$PANEL_HTML")" != "$TMP_DIR" ]; then
			echo "移動 panel.html 到頂層目錄..."
			mv "$PANEL_HTML" "$TMP_DIR/"
		fi
	fi
	
	# 如果存在 index.html，先移除它
	if [ -f "$TMP_DIR/index.html" ]; then
		echo "注意：忽略壓縮檔中的 index.html"
		rm "$TMP_DIR/index.html"
	fi
	
	# 複製所有文件到安裝目錄
	echo "複製文件到安裝目錄..."
	cp -rv "$TMP_DIR"/* "$INSTALL_DIR/www/"
	
	# 確認文件複製結果
	echo "安裝目錄文件列表："
	ls -la "$INSTALL_DIR/www/"
	
	# 清理臨時目錄
	rm -rf "$TMP_DIR"
	
	echo "自定義面板文件安裝完成"
else
	echo "下載 panel.html..."
	if ! curl -sSL -o "$INSTALL_DIR/www/panel.html" "$BASE_URL/www/panel.html"; then
		echo "安裝失敗：無法下載面板頁面"
		exit 1
	fi
fi

# 設置執行權限
echo "設置 CGI 腳本權限..."
find "$INSTALL_DIR/cgi-bin" -type f -name "*.cgi" -exec chmod 755 {} \;
find "$INSTALL_DIR/cgi-bin" -type f -name "*.cgi" -exec chown www-data:www-data {} \;

# 設置目錄權限
echo "設置目錄權限..."
find "$INSTALL_DIR" -type d -exec chmod 755 {} \;

# 設置特殊權限
echo "設置特殊權限..."
chmod 755 "$INSTALL_DIR/cgi-bin"
touch "$INSTALL_DIR/config/users.conf"
touch "$INSTALL_DIR/config/sessions.conf"
chmod 600 "$INSTALL_DIR/config/users.conf"
chmod 600 "$INSTALL_DIR/config/sessions.conf"
chmod 777 "$INSTALL_DIR/cache"
chmod 755 "$INSTALL_DIR/logs"

# 設置所有權
echo "設置文件所有權..."
chown -R www-data:www-data "$INSTALL_DIR"
chown -R www-data:www-data /etc/lighttpd

# 確保日誌目錄存在且具有正確的權限
mkdir -p /var/log/lighttpd
chown -R www-data:www-data /var/log/lighttpd
chmod 755 /var/log/lighttpd

# 測試 CGI 腳本
echo "測試 CGI 腳本..."
for script in "$INSTALL_DIR"/cgi-bin/*.cgi; do
	if [ -f "$script" ]; then
		if ! sudo -u www-data bash -n "$script"; then
			echo "錯誤：CGI 腳本 $script 語法檢查失敗"
			exit 1
		fi
	else
		echo "錯誤：找不到 CGI 腳本"
		exit 1
	fi
done

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
echo "${ADMIN_USER}:$(echo -n "${ADMIN_PASS}" | md5sum | cut -d' ' -f1)" > "$INSTALL_DIR/config/users.conf"

# 重啟 lighttpd
echo "重啟 lighttpd 服務..."
systemctl restart lighttpd

# 等待服務啟動
echo "等待服務啟動..."
sleep 2

# 檢查服務狀態
if ! systemctl is-active --quiet lighttpd; then
	echo "錯誤：lighttpd 服務未能正常啟動"
	echo "錯誤日誌："
	tail -n 20 "$INSTALL_DIR/logs/error.log"
	exit 1
fi

# 檢查端口
if ! netstat -tuln | grep -q ":8080 "; then
	echo "錯誤：服務未能在 8080 端口啟動"
	echo "當前監聽的端口："
	netstat -tuln | grep LISTEN
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
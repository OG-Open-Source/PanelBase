#!/bin/bash

[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source ~/utilkit.sh

CHECK_ROOT

# 檢查必要命令
deps=(lighttpd curl jq)
CHECK_DEPS -i

# 創建安裝目錄結構
INSTALL_DIR="/opt/panelbase"
TASK "\n創建安裝目錄結構" "ADD -d $INSTALL_DIR/{config,cgi-bin,www,logs}"
text "下載必要文件..."

# 下載 CGI 腳本
download_files() {
	target_dir=$1
	shift
	files=("$@")
	for file in "${files[@]}"; do
		echo "Downloading $file..."
		curl -sSL "https://raw.githubusercontent.com/OG-Open-Source/PanelBase/refs/heads/main/${file}" -o "$target_dir/$(basename ${file})"
		chmod 755 "$target_dir/$(basename ${file})"
		echo "* File downloaded successfully to $target_dir/$(basename ${file})"
	done
}

# 下載 CGI 腳本
CGI_FILES=(
	"src/cgi-bin/auth.cgi"
	"src/cgi-bin/api.cgi"
	"src/cgi-bin/example.py"
	"src/cgi-bin/check_auth.sh"
)
download_files "$INSTALL_DIR/cgi-bin" "${CGI_FILES[@]}"

# 檢查是否存在自定義的 index.html
if [ ! -f "$INSTALL_DIR/www/index.html" ]; then
    text "創建預設登入頁面..."
    cat > $INSTALL_DIR/www/index.html << 'EOL'
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>管理面板</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 0;
        }
        .login-container {
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
        }
        .login-box {
            background-color: white;
            padding: 2rem;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
            width: 300px;
        }
        .form-group {
            margin-bottom: 1rem;
        }
        label {
            display: block;
            margin-bottom: 0.5rem;
        }
        input {
            width: 100%;
            padding: 0.5rem;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-sizing: border-box;
        }
        button {
            width: 100%;
            padding: 0.75rem;
            background-color: #4CAF50;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background-color: #45a049;
        }
        .error {
            color: red;
            display: none;
            margin-top: 1rem;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="login-box">
            <h2 style="text-align: center; margin-bottom: 2rem;">管理面板登入</h2>
            <form id="loginForm">
                <div class="form-group">
                    <label for="username">用戶名</label>
                    <input type="text" id="username" name="username" required>
                </div>
                <div class="form-group">
                    <label for="password">密碼</label>
                    <input type="password" id="password" name="password" required>
                </div>
                <button type="submit">登入</button>
                <div id="error" class="error">登入失敗，請檢查用戶名和密碼</div>
            </form>
        </div>
    </div>

    <script>
        document.getElementById('loginForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            
            try {
                const response = await fetch('/cgi-bin/auth.cgi', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ username, password })
                });
                
                const data = await response.json();
                
                if (data.status === 'success') {
                    document.cookie = `auth_token=${data.token}; path=/`;
                    window.location.href = '/panel.html';
                } else {
                    document.getElementById('error').style.display = 'block';
                }
            } catch (error) {
                console.error('Error:', error);
                document.getElementById('error').style.display = 'block';
            }
        });
    </script>
</body>
</html>
EOL
else
    text "使用現有的登入頁面..."
fi

# 設置登入頁面權限
chmod 644 $INSTALL_DIR/www/index.html

LIGHTTPD_CONF="/etc/lighttpd/lighttpd.conf"
text "配置 Lighttpd..."
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
TASK "設置權限" "
	chown -R www-data:www-data $INSTALL_DIR
	chmod -R 755 $INSTALL_DIR
	chmod -R 755 $INSTALL_DIR/cgi-bin
	chmod 644 $INSTALL_DIR/www/index.html
"

# 啟動服務
TASK "啟動 Lighttpd 服務" "
	systemctl enable lighttpd
	systemctl restart lighttpd
"

# 設置管理員帳號
text "設置管理員帳號..."
INPUT "請輸入管理員用戶名: " admin_user
INPUT "請輸入管理員密碼: " admin_pass
echo "$admin_user:$(echo -n "$admin_pass" | sha256sum | cut -d' ' -f1)" > $INSTALL_DIR/config/admin.conf

# 測試認證
text "測試認證..."
test_auth=$(curl -s -X POST -H "Content-Type: application/json" \
	-d "{\"username\":\"$admin_user\",\"password\":\"$admin_pass\"}" \
	http://localhost:8080/cgi-bin/auth.cgi)

echo "認證響應: $test_auth" >&2

if [ "$(echo "$test_auth" | jq -r .status)" = "success" ]; then
	text "認證測試成功！"
	
	# 確保配置目錄權限正確
	chown -R www-data:www-data "$INSTALL_DIR/config"
	chmod 755 "$INSTALL_DIR/config"
	chmod 644 "$INSTALL_DIR/config/admin.conf"
	
	# 創建並設置 token.conf 權限
	touch "$INSTALL_DIR/config/token.conf"
	chown www-data:www-data "$INSTALL_DIR/config/token.conf"
	chmod 644 "$INSTALL_DIR/config/token.conf"
	
	# 保存初始 token
	echo "$test_auth" | jq -r .token > "$INSTALL_DIR/config/token.conf"
	
	text "Token 已保存到配置文件"
else
	error "認證測試失敗！"
	text "請檢查 $INSTALL_DIR/logs/auth.log 和 $INSTALL_DIR/logs/error.log 查看詳細信息"
fi

text "安裝完成！"
text "請訪問 http://your-server-ip:8080 來訪問面板"

text "安裝資訊："
text "安裝目錄：\t$INSTALL_DIR"
text "CGI 目錄：\t$INSTALL_DIR/cgi-bin"
text "網站根目錄：\t$INSTALL_DIR/www"
text "日誌目錄：\t$INSTALL_DIR/logs"
text "配置文件：\t$LIGHTTPD_CONF"
#!/bin/bash

# 啟用錯誤追蹤
set -x

# 設置錯誤日誌
exec 2>>/opt/panelbase/logs/auth.log

# 設置 Content-Type
echo "Content-type: application/json"
echo ""

# 檢查是否為驗證 token 的請求
if [ -n "$HTTP_COOKIE" ] && [[ "$HTTP_COOKIE" =~ auth_token=([^;]+) ]]; then
    token="${BASH_REMATCH[1]}"
    stored_token=$(cat /opt/panelbase/config/token.conf 2>/dev/null)
    
    if [ "$token" = "$stored_token" ]; then
        echo '{"status": "success", "message": "token 驗證成功"}'
        exit 0
    else
        echo '{"status": "error", "message": "無效的 token"}'
        exit 1
    fi
fi

# 讀取 POST 數據
read -n $CONTENT_LENGTH POST_DATA
echo "Received POST data: $POST_DATA" >&2

# 解析用戶名和密碼
username=$(echo "$POST_DATA" | grep -o '"username":"[^"]*' | cut -d'"' -f4)
password=$(echo "$POST_DATA" | grep -o '"password":"[^"]*' | cut -d'"' -f4)

echo "Username: $username" >&2
echo "Password length: ${#password}" >&2

# 計算密碼的 SHA-256 雜湊值
password_hash=$(echo -n "$password" | sha256sum | cut -d' ' -f1)
echo "Password hash: $password_hash" >&2

# 讀取儲存的認證信息
stored_auth=$(cat /opt/panelbase/config/admin.conf)
echo "Stored auth: $stored_auth" >&2

stored_username=$(echo "$stored_auth" | cut -d':' -f1)
stored_password_hash=$(echo "$stored_auth" | cut -d':' -f2)

echo "Stored username: $stored_username" >&2
echo "Stored password hash: $stored_password_hash" >&2

# 驗證用戶名和密碼
if [ "$username" = "$stored_username" ] && [ "$password_hash" = "$stored_password_hash" ]; then
    # 生成 token（使用時間戳和隨機數）
    token=$(date +%s%N | sha256sum | cut -d' ' -f1)
    echo "$token" > /opt/panelbase/config/token.conf
    chmod 644 /opt/panelbase/config/token.conf
    chown www-data:www-data /opt/panelbase/config/token.conf
    
    # 返回成功響應，包含 token
    echo "{\"status\": \"success\", \"message\": \"認證成功\", \"token\": \"$token\"}"
else
    # 返回失敗響應
    echo "{\"status\": \"error\", \"message\": \"認證失敗\", \"debug\": {\"username_match\": \"$username = $stored_username\", \"password_match\": \"$password_hash = $stored_password_hash\"}}"
fi 
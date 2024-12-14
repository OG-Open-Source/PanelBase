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
    install_time=$(cat /opt/panelbase/config/install_time.conf 2>/dev/null)
    current_time=$(date +%s)
    
    # 檢查 token 是否存在且有效
    if [ -n "$stored_token" ] && [ -n "$install_time" ]; then
        # 解析 token 中的時間戳和安裝時間
        IFS='.' read -r token_value token_expire token_install <<< "$token"
        
        # 驗證 token
        if [ "$token_value" = "$stored_token" ] && \
           [ "$token_install" = "$install_time" ] && \
           [ "$token_expire" -gt "$current_time" ]; then
            echo '{"status": "success", "message": "token 驗證成功"}'
            exit 0
        fi
    fi
    
    echo '{"status": "error", "message": "無效的 token 或已過期"}'
    exit 1
fi

# 讀取 POST 數據
read -n $CONTENT_LENGTH POST_DATA
echo "Received POST data: $POST_DATA" >&2

# 解析用戶名和密碼
username=$(echo "$POST_DATA" | jq -r '.username // empty')
password=$(echo "$POST_DATA" | jq -r '.password // empty')

if [ -z "$username" ] || [ -z "$password" ]; then
    echo '{"status": "error", "message": "缺少用戶名或密碼"}'
    exit 1
fi

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
    token_value=$(date +%s%N | sha256sum | cut -d' ' -f1)
    current_time=$(date +%s)
    expire_time=$((current_time + 86400))  # 24小時後過期
    install_time=$(cat /opt/panelbase/config/install_time.conf)
    
    # 組合完整的 token（格式：token值.過期時間.安裝時間）
    token="${token_value}.${expire_time}.${install_time}"
    
    # 保存 token 值（不包含過期時間和安裝時間）
    echo "$token_value" > /opt/panelbase/config/token.conf
    chmod 644 /opt/panelbase/config/token.conf
    chown www-data:www-data /opt/panelbase/config/token.conf
    
    echo "{\"status\": \"success\", \"message\": \"認證成功\", \"token\": \"$token\", \"expire\": $expire_time}"
else
    echo "{\"status\": \"error\", \"message\": \"認證失敗\"}"
fi 
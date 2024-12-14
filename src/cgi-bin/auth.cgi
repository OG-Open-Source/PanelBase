#!/bin/bash

# 設置 Content-Type
echo "Content-type: application/json"
echo ""

# 讀取 POST 數據
read -n $CONTENT_LENGTH POST_DATA

# 解析用戶名和密碼
username=$(echo "$POST_DATA" | grep -o '"username":"[^"]*' | cut -d'"' -f4)
password=$(echo "$POST_DATA" | grep -o '"password":"[^"]*' | cut -d'"' -f4)

# 計算密碼的 SHA-256 雜湊值
password_hash=$(echo -n "$password" | sha256sum | cut -d' ' -f1)

# 讀取儲存的認證信息
stored_auth=$(cat /opt/panelbase/config/admin.conf)
stored_username=$(echo "$stored_auth" | cut -d':' -f1)
stored_password_hash=$(echo "$stored_auth" | cut -d':' -f2)

# 驗證用戶名和密碼
if [ "$username" = "$stored_username" ] && [ "$password_hash" = "$stored_password_hash" ]; then
    echo '{"status": "success", "message": "認證成功"}'
else
    echo '{"status": "error", "message": "認證失敗"}'
fi 
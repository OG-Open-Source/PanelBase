#!/bin/bash

# 讀取 POST 數據
read -n $CONTENT_LENGTH POST_DATA

# 解析用戶名和密碼
USERNAME=$(echo "$POST_DATA" | grep -oP 'username=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
PASSWORD=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

# 檢查配置文件是否存在
CONFIG_FILE="/opt/panelbase/config/users.conf"
if [ ! -f "$CONFIG_FILE" ]; then
    # 如果配置文件不存在，創建默認用戶（admin/admin）
    echo "admin:$(echo -n "admin" | md5sum | cut -d' ' -f1)" > "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"
fi

# 檢查用戶名和密碼
STORED_HASH=$(grep "^$USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
INPUT_HASH=$(echo -n "$PASSWORD" | md5sum | cut -d' ' -f1)

if [ "$STORED_HASH" = "$INPUT_HASH" ]; then
    # 生成隨機 token
    TOKEN=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)
    
    # 保存 token（在實際應用中應該使用更安全的存儲方式）
    echo "$TOKEN:$USERNAME:$(date +%s)" >> "/opt/panelbase/config/sessions.conf"
    
    # 設置 cookie
    echo "Content-type: text/plain"
    echo "Set-Cookie: auth_token=$TOKEN; Path=/; HttpOnly; SameSite=Strict"
    echo "Status: 200"
    echo
    echo "OK"
else
    echo "Content-type: text/plain"
    echo "Status: 401"
    echo
    echo "Unauthorized"
fi 
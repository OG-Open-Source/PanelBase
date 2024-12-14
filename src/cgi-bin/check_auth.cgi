#!/bin/bash

# 從 HTTP_COOKIE 中獲取 auth_token
AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')

if [ -z "$AUTH_TOKEN" ]; then
    # 沒有 token，重定向到登入頁面
    echo "Status: 302"
    echo "Location: /"
    echo
    exit 0
fi

# 檢查 token 是否有效
SESSION_FILE="/opt/panelbase/config/sessions.conf"
if [ ! -f "$SESSION_FILE" ]; then
    echo "Status: 302"
    echo "Location: /"
    echo
    exit 0
fi

# 檢查 token 是否存在且未過期（24小時有效期）
CURRENT_TIME=$(date +%s)
VALID_SESSION=$(awk -F: -v token="$AUTH_TOKEN" -v time="$CURRENT_TIME" \
    '$1 == token && (time - $3) < 86400 {print $2}' "$SESSION_FILE")

if [ -z "$VALID_SESSION" ]; then
    echo "Status: 302"
    echo "Location: /"
    echo
    exit 0
fi

# Token 有效，允許訪問
echo "Content-type: text/plain"
echo "Status: 200"
echo
echo "Authorized" 
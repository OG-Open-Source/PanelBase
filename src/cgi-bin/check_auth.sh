#!/bin/bash

# 檢查 cookie 中的 token
if [ -z "$HTTP_COOKIE" ] || ! [[ "$HTTP_COOKIE" =~ auth_token=([^;]+) ]]; then
    echo "Content-type: application/json"
    echo ""
    echo '{"status": "error", "message": "未登入"}'
    exit 1
fi

token="${BASH_REMATCH[1]}"
stored_token=$(cat /opt/panelbase/config/token.conf 2>/dev/null)

if [ -z "$stored_token" ] || [ "$token" != "$stored_token" ]; then
    echo "Content-type: application/json"
    echo ""
    echo '{"status": "error", "message": "無效的 token"}'
    exit 1
fi 
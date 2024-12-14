#!/bin/bash

# 啟用錯誤追蹤
set -x

# 設置錯誤日誌
exec 2>>/opt/panelbase/logs/page.log

# 檢查登入狀態
source $(dirname "$0")/check_auth.sh >/dev/null 2>&1
auth_status=$?

# 獲取請求的頁面路徑
request_uri="${REQUEST_URI:-/index.html}"
request_uri="${request_uri%%\?*}"  # 移除 URL 參數

# 如果是登入頁面，直接顯示
if [ "$request_uri" = "/login.html" ]; then
    echo "Content-type: text/html"
    echo ""
    cat /opt/panelbase/www/login.html
    exit 0
fi

# 如果未登入且不是登入頁面，重定向到登入頁面
if [ $auth_status -ne 0 ]; then
    echo "Status: 302 Found"
    echo "Location: /login.html"
    echo ""
    exit 0
fi

# 構建實際的文件路徑
page_path="/opt/panelbase/www${request_uri}"

# 檢查文件是否存在
if [ ! -f "$page_path" ]; then
    echo "Status: 404 Not Found"
    echo "Content-type: text/plain"
    echo ""
    echo "404 - Page not found"
    exit 0
fi

# 檢查文件類型並設置對應的 Content-Type
case "${page_path##*.}" in
    html) echo "Content-type: text/html" ;;
    css) echo "Content-type: text/css" ;;
    js) echo "Content-type: application/javascript" ;;
    json) echo "Content-type: application/json" ;;
    *) echo "Content-type: text/plain" ;;
esac
echo ""

# 輸出文件內容
cat "$page_path" 
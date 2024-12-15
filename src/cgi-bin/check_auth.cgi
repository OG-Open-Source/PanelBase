#!/bin/bash

# 導入共用函數和認證
source /opt/panelbase/cgi-bin/common.cgi
source /opt/panelbase/cgi-bin/auth.cgi

# 配置
PANEL_ROOT="/opt/panelbase"
STATIC_ROOT="$PANEL_ROOT/static"
HTML_ROOT="$PANEL_ROOT/html"

# 檢查是否為靜態文件請求
is_static_file() {
    local path="$1"
    [[ "$path" =~ \.(css|js|jpg|jpeg|png|gif|ico|svg|woff|woff2|ttf|eot)$ ]]
}

# 獲取文件的 MIME 類型
get_mime_type() {
    local file="$1"
    case "${file##*.}" in
        css)  echo "text/css" ;;
        js)   echo "application/javascript" ;;
        jpg|jpeg) echo "image/jpeg" ;;
        png)  echo "image/png" ;;
        gif)  echo "image/gif" ;;
        ico)  echo "image/x-icon" ;;
        svg)  echo "image/svg+xml" ;;
        woff) echo "font/woff" ;;
        woff2) echo "font/woff2" ;;
        ttf)  echo "font/ttf" ;;
        eot)  echo "application/vnd.ms-fontobject" ;;
        *)    echo "application/octet-stream" ;;
    esac
}

# 處理靜態文件請求
serve_static_file() {
    local path="$1"
    local file_path
    
    # 移除 URL 中的查詢字符串
    path="${path%%\?*}"
    
    # 檢查文件是否存在
    file_path="$STATIC_ROOT$path"
    if [ ! -f "$file_path" ]; then
        send_error 404 "找不到文件"
        return 1
    }
    
    # 設置 MIME 類型
    local mime_type=$(get_mime_type "$file_path")
    echo "Content-type: $mime_type"
    echo
    
    # 輸出文件內容
    cat "$file_path"
    return 0
}

# 處理 HTML 文件請求
serve_html_file() {
    local path="$1"
    local file_path
    
    # 移除 URL 中的查詢字符串
    path="${path%%\?*}"
    
    # 如果路徑為根目錄，預設為 index.html
    if [ "$path" = "/" ]; then
        path="/index.html"
    fi
    
    # 檢查文件是否存在
    file_path="$HTML_ROOT$path"
    if [ ! -f "$file_path" ]; then
        # 如果找不到文件，檢查是否需要認證
        if ! check_auth; then
            file_path="$HTML_ROOT/login.html"
        else
            file_path="$HTML_ROOT/panel.html"
        fi
    fi
    
    # 輸出 HTML 內容
    echo "Content-type: text/html"
    echo
    cat "$file_path"
    return 0
}

# 主程序
main() {
    local path="$REQUEST_URI"
    
    # 檢查是否為靜態文件請求
    if is_static_file "$path"; then
        serve_static_file "$path"
        exit $?
    fi
    
    # 檢查是否為 API 請求
    if echo "$QUERY_STRING" | grep -q "action="; then
        # 轉發到對應的 CGI 處理器
        if echo "$QUERY_STRING" | grep -q "action=\(login\|logout\|get_username\|change_password\|change_username\)"; then
            exec /opt/panelbase/cgi-bin/auth.cgi
        else
            exec /opt/panelbase/cgi-bin/panel.cgi
        fi
        exit $?
    fi
    
    # 處理 HTML 文件請求
    serve_html_file "$path"
    exit $?
}

# 執行主程序
main
#!/bin/bash

# 配置
CACHE_DIR="/opt/panelbase/cache"
CACHE_TIMEOUT=60  # 緩存過期時間（秒）

# 創建緩存目錄
mkdir -p "$CACHE_DIR"
chmod 755 "$CACHE_DIR"

# 緩存相關函數
get_cache_key() {
    local key="$1"
    echo -n "$key" | md5sum | cut -d' ' -f1
}

get_cache() {
    local key="$1"
    local cache_file="$CACHE_DIR/$(get_cache_key "$key")"
    
    if [ -f "$cache_file" ]; then
        local cache_time=$(stat -c %Y "$cache_file")
        local current_time=$(date +%s)
        local age=$((current_time - cache_time))
        
        if [ $age -lt $CACHE_TIMEOUT ]; then
            cat "$cache_file"
            return 0
        fi
    fi
    return 1
}

set_cache() {
    local key="$1"
    local data="$2"
    local cache_file="$CACHE_DIR/$(get_cache_key "$key")"
    echo "$data" > "$cache_file"
    chmod 644 "$cache_file"
}

clear_cache() {
    local key="$1"
    if [ -n "$key" ]; then
        rm -f "$CACHE_DIR/$(get_cache_key "$key")"
    else
        rm -f "$CACHE_DIR"/*
    fi
}

# 參數驗證函數
validate_param() {
    local param_name="$1"
    local param_value="$2"
    local param_type="$3"
    local required="$4"
    local extra_param="$5"
    
    # 檢查必填參數
    if [ "$required" = "true" ] && [ -z "$param_value" ]; then
        echo "錯誤：參數 $param_name 為必填項"
        return 1
    fi
    
    # 如果參數為空且非必填，則通過驗證
    if [ -z "$param_value" ] && [ "$required" != "true" ]; then
        return 0
    fi
    
    case "$param_type" in
        "int")
            if ! [[ "$param_value" =~ ^[0-9]+$ ]]; then
                echo "錯誤：參數 $param_name 必須為整數"
                return 1
            fi
            ;;
        "float")
            if ! [[ "$param_value" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
                echo "錯誤：參數 $param_name 必須為數字"
                return 1
            fi
            ;;
        "email")
            if ! [[ "$param_value" =~ ^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$ ]]; then
                echo "錯誤：參數 $param_name 必須為有效的電子郵件地址"
                return 1
            fi
            ;;
        "date")
            if ! date -d "$param_value" >/dev/null 2>&1; then
                echo "錯誤：參數 $param_name 必須為有效的日期格式"
                return 1
            fi
            ;;
        "enum")
            if ! echo "$extra_param" | grep -q "^$param_value$"; then
                echo "錯誤：參數 $param_name 必須為以下值之一：$extra_param"
                return 1
            fi
            ;;
        "regex")
            if ! [[ "$param_value" =~ $extra_param ]]; then
                echo "錯誤：參數 $param_name 格式不正確"
                return 1
            fi
            ;;
    esac
    return 0
}

# 解析查詢字符串
parse_query_string() {
    local query="$1"
    declare -A params
    
    if [ -n "$query" ]; then
        IFS='&' read -ra pairs <<< "$query"
        for pair in "${pairs[@]}"; do
            IFS='=' read -r key value <<< "$pair"
            # URL 解碼
            value=$(echo -e "${value//%/\\x}")
            # 轉義特殊字符
            value="${value//\'/\\\'}"
            params["$key"]="$value"
        done
    fi
    
    declare -p params
}

# URL 編碼函數
url_encode() {
    local string="$1"
    local length="${#string}"
    local encoded=""
    
    for (( i=0; i<length; i++ )); do
        local c="${string:i:1}"
        case "$c" in
            [a-zA-Z0-9.~_-]) encoded+="$c" ;;
            *) encoded+=$(printf '%%%02X' "'$c") ;;
        esac
    done
    
    echo "$encoded"
}

# URL 解碼函數
url_decode() {
    local encoded="$1"
    echo -e "${encoded//%/\\x}"
}

# JSON 相關函數
json_escape() {
    local string="$1"
    string="${string//\\/\\\\}"
    string="${string//\"/\\\"}"
    string="${string//	/\\t}"
    string="${string//
/\\n}"
    string="${string//\r/\\r}"
    echo "$string"
}

# 錯誤處理函數
send_error() {
    local status="$1"
    local message="$2"
    
    echo "Content-type: application/json"
    echo "Status: $status"
    echo
    echo "{\"error\": \"$(json_escape "$message")\"}"
    exit 1
}

send_success() {
    local data="$1"
    
    echo "Content-type: application/json"
    echo "Status: 200"
    echo
    echo "$data"
}

# 如果是被其他腳本導入，則不執行以下代碼
if [ "${BASH_SOURCE[0]}" != "$0" ]; then
    return 0
fi

# 處理請求
ACTION=$(echo "$QUERY_STRING" | grep -oP 'action=\K[^&]+')

case "$ACTION" in
    "clear_cache")
        clear_cache
        send_success '{"status": "success"}'
        ;;
    *)
        send_error 400 "無效的操作"
        ;;
esac
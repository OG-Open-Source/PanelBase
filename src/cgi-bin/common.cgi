#!/bin/bash

# 配置
CACHE_DIR="/opt/panelbase/cache"
CACHE_TIMEOUT=60  # 緩存過期時間（秒）
ROUTES_FILE="/opt/panelbase/config/routes.conf"

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
            local valid_values="$5"
            if ! echo "$valid_values" | grep -q "^$param_value$"; then
                echo "錯誤：參數 $param_name 必須為以下值之一：$valid_values"
                return 1
            fi
            ;;
        "regex")
            local pattern="$5"
            if ! [[ "$param_value" =~ $pattern ]]; then
                echo "錯誤：參數 $param_name 格式不正確"
                return 1
            fi
            ;;
    esac
    return 0
}

# 路由相關函數
load_routes() {
    if [ ! -f "$ROUTES_FILE" ]; then
        echo "錯誤：找不到路由配置文件"
        return 1
    fi
    source "$ROUTES_FILE"
}

route_request() {
    local path="$1"
    local method="$2"
    local action="$3"
    
    # 載入路由配置
    load_routes
    
    # 檢查路由是否存在
    local route_handler="route_${path}_${method}_${action}"
    if [ "$(type -t $route_handler)" != "function" ]; then
        echo "Content-type: application/json"
        echo "Status: 404"
        echo
        echo '{"error": "Not Found", "message": "找不到指定的路由"}'
        return 1
    fi
    
    # 執行路由處理函數
    $route_handler
}

# 解析查詢字符串
parse_query_string() {
    local query="$1"
    local params=()
    IFS='&' read -ra pairs <<< "$query"
    for pair in "${pairs[@]}"; do
        IFS='=' read -r key value <<< "$pair"
        # URL 解碼
        value=$(echo -e "${value//%/\\x}")
        # 轉義特殊字符
        value="${value//\'/\\\'}"
        params["$key"]="$value"
    done
    declare -p params
}

# 示例用法：
# source common.cgi
# 
# # 使用緩存
# cache_key="system_info"
# if ! data=$(get_cache "$cache_key"); then
#     data=$(get_system_info)
#     set_cache "$cache_key" "$data"
# fi
# 
# # 驗證參數
# validate_param "age" "25" "int" "true"
# validate_param "email" "user@example.com" "email" "true"
# 
# # 路由請求
# route_request "user" "GET" "profile" 
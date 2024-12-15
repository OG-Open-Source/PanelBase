#!/bin/bash

# 導入共用函數和認證
source /opt/panelbase/cgi-bin/common.cgi
source /opt/panelbase/cgi-bin/auth.cgi

# 配置
ROUTES_FILE="/opt/panelbase/config/routes.conf"

# 路由相關函數
load_routes() {
    if [ ! -f "$ROUTES_FILE" ]; then
        send_error 500 "找不到路由配置文件"
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
        send_error 404 "找不到指定的路由"
        return 1
    fi
    
    # 檢查認證
    if ! check_auth; then
        send_unauthorized
        return 1
    fi
    
    # 執行路由處理函數
    $route_handler
}

# 系統資訊相關函數
get_system_info() {
    local cache_key="system_info"
    local data
    
    if ! data=$(get_cache "$cache_key"); then
        # 收集系統資訊
        local cpu_info=$(cat /proc/cpuinfo | grep 'model name' | head -n1 | cut -d: -f2 | xargs)
        local cpu_cores=$(nproc)
        local memory_total=$(free -m | awk '/^Mem:/{print $2}')
        local memory_used=$(free -m | awk '/^Mem:/{print $3}')
        local disk_info=$(df -h / | awk 'NR==2{print $2","$3","$5}')
        local uptime=$(uptime -p)
        local load_average=$(uptime | grep -oP 'load average: \K.*')
        
        # 構建 JSON 響應
        data="{
            \"cpu\": {
                \"model\": \"$(json_escape "$cpu_info")\",
                \"cores\": $cpu_cores
            },
            \"memory\": {
                \"total\": $memory_total,
                \"used\": $memory_used
            },
            \"disk\": {
                \"total\": \"$(echo $disk_info | cut -d, -f1)\",
                \"used\": \"$(echo $disk_info | cut -d, -f2)\",
                \"usage\": \"$(echo $disk_info | cut -d, -f3)\"
            },
            \"uptime\": \"$(json_escape "$uptime")\",
            \"load_average\": \"$(json_escape "$load_average")\"
        }"
        
        # 設置緩存
        set_cache "$cache_key" "$data"
    fi
    
    echo "$data"
}

# 服務控制相關函數
service_action() {
    local service="$1"
    local action="$2"
    
    # 驗證參數
    if ! validate_param "service" "$service" "regex" "true" "^[a-zA-Z0-9_-]+$"; then
        return 1
    fi
    
    if ! validate_param "action" "$action" "enum" "true" "start|stop|restart|status"; then
        return 1
    fi
    
    # 執行服務操作
    local result
    case "$action" in
        "status")
            result=$(systemctl status "$service" 2>&1)
            ;;
        *)
            result=$(systemctl "$action" "$service" 2>&1)
            ;;
    esac
    
    local status=$?
    echo "{\"status\": $status, \"output\": \"$(json_escape "$result")\"}"
}

# 如果是被其他腳本導入，則不執行以下代碼
if [ "${BASH_SOURCE[0]}" != "$0" ]; then
    return 0
fi

# 處理請求
ACTION=$(echo "$QUERY_STRING" | grep -oP 'action=\K[^&]+')

case "$ACTION" in
    "system_info")
        if ! check_auth; then
            send_unauthorized
            exit 1
        fi
        send_success "$(get_system_info)"
        ;;
        
    "service")
        if ! check_auth; then
            send_unauthorized
            exit 1
        fi
        
        read -n $CONTENT_LENGTH POST_DATA
        SERVICE=$(echo "$POST_DATA" | grep -oP 'service=\K[^&]+')
        ACTION=$(echo "$POST_DATA" | grep -oP 'action=\K[^&]+')
        
        RESULT=$(service_action "$SERVICE" "$ACTION")
        if [ $? -eq 0 ]; then
            send_success "$RESULT"
        else
            send_error 400 "$RESULT"
        fi
        ;;
        
    "route")
        PATH_INFO=$(echo "$QUERY_STRING" | grep -oP 'path=\K[^&]+')
        METHOD=$(echo "$QUERY_STRING" | grep -oP 'method=\K[^&]+')
        ROUTE_ACTION=$(echo "$QUERY_STRING" | grep -oP 'route_action=\K[^&]+')
        
        route_request "$PATH_INFO" "$METHOD" "$ROUTE_ACTION"
        ;;
        
    *)
        send_error 400 "無效的操作"
        ;;
esac
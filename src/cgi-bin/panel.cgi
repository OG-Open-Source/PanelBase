#!/bin/bash

# 導入共用函數和認證
source /opt/panelbase/cgi-bin/common.cgi
source /opt/panelbase/cgi-bin/auth.cgi

# 配置
readonly ROUTES_FILE="/opt/panelbase/config/routes.conf"

# 路由相關函數
load_routes() {
    [ ! -f "$ROUTES_FILE" ] && send_error 500 "找不到路由配置文件" && return 1
    source "$ROUTES_FILE"
}

route_request() {
    local path="$1"
    local method="$2"
    local action="$3"
    local route_handler="route_${path}_${method}_${action}"
    
    # 載入路由配置
    load_routes
    
    # 檢查路由是否存在
    [ "$(type -t $route_handler)" != "function" ] && send_error 404 "找不到指定的路由" && return 1
    
    # 檢查認證
    check_auth || send_unauthorized
    
    # 執行路由處理函數
    $route_handler
}

# 系統資訊相關函數
get_system_info() {
    local cache_key="system_info"
    local data cpu_info cpu_cores memory_total memory_used disk_info uptime load_average
    
    data=$(get_cache "$cache_key") && echo "$data" && return 0
    
    # 收集系統資訊
    cpu_info=$(cat /proc/cpuinfo | grep 'model name' | head -n1 | cut -d: -f2 | xargs)
    cpu_cores=$(nproc)
    memory_total=$(free -m | awk '/^Mem:/{print $2}')
    memory_used=$(free -m | awk '/^Mem:/{print $3}')
    disk_info=$(df -h / | awk 'NR==2{print $2","$3","$5}')
    uptime=$(uptime -p)
    load_average=$(uptime | grep -oP 'load average: \K.*')
    
    # 構建 JSON 響應
    data=$(cat << EOF
{
    "cpu": {
        "model": "$(json_escape "$cpu_info")",
        "cores": $cpu_cores
    },
    "memory": {
        "total": $memory_total,
        "used": $memory_used
    },
    "disk": {
        "total": "$(echo $disk_info | cut -d, -f1)",
        "used": "$(echo $disk_info | cut -d, -f2)",
        "usage": "$(echo $disk_info | cut -d, -f3)"
    },
    "uptime": "$(json_escape "$uptime")",
    "load_average": "$(json_escape "$load_average")"
}
EOF
)
    
    # 設置緩存
    set_cache "$cache_key" "$data"
    echo "$data"
}

# 服務控制相關函數
service_action() {
    local service="$1"
    local action="$2"
    local result status
    
    # 驗證參數
    validate_param "service" "$service" "regex" "true" "^[a-zA-Z0-9_-]+$" || return 1
    validate_param "action" "$action" "enum" "true" "start|stop|restart|status" || return 1
    
    # 執行服務操作
    case "$action" in
        "status")
            result=$(systemctl status "$service" 2>&1)
            ;;
        *)
            result=$(systemctl "$action" "$service" 2>&1)
            ;;
    esac
    
    status=$?
    echo "{\"status\": $status, \"output\": \"$(json_escape "$result")\"}"
}

# 如果是被其他腳本導入，則不執行以下代碼
[ "${BASH_SOURCE[0]}" != "$0" ] && return 0

# 處理請求
ACTION=$(echo "$QUERY_STRING" | grep -oP 'action=\K[^&]+')

case "$ACTION" in
    "system_info")
        check_auth || send_unauthorized
        send_success "$(get_system_info)"
        ;;
        
    "service")
        check_auth || send_unauthorized
        
        read -n $CONTENT_LENGTH POST_DATA
        local service action result
        
        service=$(echo "$POST_DATA" | grep -oP 'service=\K[^&]+')
        action=$(echo "$POST_DATA" | grep -oP 'action=\K[^&]+')
        
        result=$(service_action "$service" "$action")
        if [ $? -eq 0 ]; then
            send_success "$result"
        else
            send_error 400 "$result"
        fi
        ;;
        
    "route")
        local path_info method route_action
        
        path_info=$(echo "$QUERY_STRING" | grep -oP 'path=\K[^&]+')
        method=$(echo "$QUERY_STRING" | grep -oP 'method=\K[^&]+')
        route_action=$(echo "$QUERY_STRING" | grep -oP 'route_action=\K[^&]+')
        
        route_request "$path_info" "$method" "$route_action"
        ;;
        
    *)
        send_error 400 "無效的操作"
        ;;
esac
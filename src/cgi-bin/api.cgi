#!/bin/bash

# 設置 Content-Type
echo "Content-type: application/json"
echo ""

# 讀取請求方法和路徑
REQUEST_METHOD="$REQUEST_METHOD"
SCRIPT_NAME="$SCRIPT_NAME"
QUERY_STRING="$QUERY_STRING"

# 如果是 POST 請求，讀取 POST 數據
if [ "$REQUEST_METHOD" = "POST" ]; then
    read -n $CONTENT_LENGTH POST_DATA
fi

# 根據請求路徑處理不同的 API 端點
case "$SCRIPT_NAME" in
    */api/system-info)
        # 獲取系統資訊
        cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}')
        memory_info=$(free -m | grep Mem)
        memory_total=$(echo $memory_info | awk '{print $2}')
        memory_used=$(echo $memory_info | awk '{print $3}')
        memory_usage=$((memory_used * 100 / memory_total))
        
        echo "{
            \"cpu_usage\": \"$cpu_usage%\",
            \"memory_usage\": \"$memory_usage%\",
            \"memory_total\": \"$memory_total MB\",
            \"memory_used\": \"$memory_used MB\"
        }"
        ;;
        
    */api/change-password)
        if [ "$REQUEST_METHOD" = "POST" ]; then
            # 解析 POST 數據
            current_password=$(echo "$POST_DATA" | grep -o '"current_password":"[^"]*' | cut -d'"' -f4)
            new_password=$(echo "$POST_DATA" | grep -o '"new_password":"[^"]*' | cut -d'"' -f4)
            
            # 驗證當前密碼
            current_hash=$(echo -n "$current_password" | sha256sum | cut -d' ' -f1)
            stored_hash=$(cat /opt/panelbase/config/admin.conf | cut -d':' -f2)
            
            if [ "$current_hash" = "$stored_hash" ]; then
                # 更新密碼
                username=$(cat /opt/panelbase/config/admin.conf | cut -d':' -f1)
                new_hash=$(echo -n "$new_password" | sha256sum | cut -d' ' -f1)
                echo "$username:$new_hash" > /opt/panelbase/config/admin.conf
                echo '{"status": "success", "message": "密碼已更新"}'
            else
                echo '{"status": "error", "message": "當前密碼錯誤"}'
            fi
        else
            echo '{"status": "error", "message": "無效的請求方法"}'
        fi
        ;;
        
    */api/logs)
        # 獲取最新的日誌
        tail -n 50 /opt/panelbase/logs/error.log | jq -R -s 'split("\n")'
        ;;
        
    *)
        echo '{"status": "error", "message": "未知的 API 端點"}'
        ;;
esac 
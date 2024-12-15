#!/bin/bash

# 配置文件路徑
readonly CONFIG_FILE="/opt/panelbase/config/users.conf"
readonly SESSION_FILE="/opt/panelbase/config/sessions.conf"

# 檢查認證
check_auth() {
    local auth_token current_ip current_ua current_time result
    
    auth_token=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
    [ -z "$auth_token" ] && return 1
    
    current_ip="$REMOTE_ADDR"
    current_ua="$HTTP_USER_AGENT"
    current_time=$(date +%s)
    
    # 使用單次讀取檢查 session
    result=$(awk -F: -v token="$auth_token" -v time="$current_time" \
                  -v ip="$current_ip" -v ua="$current_ua" \
            'BEGIN { valid=0 }
             $1 == token && (time - $3) < 86400 && $4 == ip && $5 == ua { valid=1; exit }
             END { print valid }' "$SESSION_FILE")
    
    [ "$result" = "1" ] && return 0 || return 1
}

# 檢查認證或密碼
check_auth_or_password() {
    local username="$1"
    local password="$2"
    local stored_hash input_hash
    
    [ -z "$username" ] || [ -z "$password" ] && return 1
    
    stored_hash=$(grep "^$username:" "$CONFIG_FILE" | cut -d: -f2)
    [ -z "$stored_hash" ] && return 1
    
    input_hash=$(echo -n "$password" | md5sum | cut -d' ' -f1)
    [ "$stored_hash" = "$input_hash" ] && return 0 || return 1
}

# 返回未授權錯誤
send_unauthorized() {
    echo "Content-type: application/json"
    echo "Status: 401"
    echo
    echo '{"error": "Unauthorized"}'
    exit 1
}

# 獲取當前用戶名
get_current_username() {
    local auth_token="$1"
    awk -F: -v token="$auth_token" '$1 == token {print $2; exit}' "$SESSION_FILE"
}

# URL 解碼函數
urldecode() {
    local encoded="$1"
    echo -n "$encoded" | sed 's/%40/@/g; s/%2B/+/g; s/%20/ /g'
}

# 如果是被其他腳本導入，則不執行以下代碼
[ "${BASH_SOURCE[0]}" != "$0" ] && return 0

# 處理請求
ACTION=$(echo "$QUERY_STRING" | grep -oP 'action=\K[^&]+')

case "$ACTION" in
    "check_auth")
        if check_auth; then
            echo "Content-type: application/json"
            echo "Status: 200"
            echo
            echo '{"status": "authorized"}'
        else
            send_unauthorized
        fi
        ;;

    "login")
        read -n $CONTENT_LENGTH POST_DATA
        local username password token
        
        username=$(echo "$POST_DATA" | grep -oP 'username=\K[^&]+' | urldecode)
        password=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | urldecode)

        if check_auth_or_password "$username" "$password"; then
            token=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)
            echo "$token:$username:$(date +%s):$REMOTE_ADDR:$HTTP_USER_AGENT" >> "$SESSION_FILE"

            echo "Content-type: application/json"
            echo "Set-Cookie: auth_token=$token; Path=/; HttpOnly; SameSite=Strict; Max-Age=86400"
            echo "Status: 200"
            echo
            echo '0'
        else
            echo "Content-type: application/json"
            echo "Status: 401"
            echo
            echo '1'
        fi
        ;;

    "logout")
        local auth_token
        auth_token=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
        
        if [ -n "$auth_token" ]; then
            sed -i "/^$auth_token:/d" "$SESSION_FILE"
        fi

        echo "Content-type: text/html"
        echo "Set-Cookie: auth_token=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT"
        echo "Status: 302"
        echo "Location: /"
        echo
        ;;

    "get_username")
        if check_auth; then
            local auth_token username
            auth_token=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
            username=$(get_current_username "$auth_token")
            
            echo "Content-type: application/json"
            echo "Status: 200"
            echo
            echo "$username"
        else
            send_unauthorized
        fi
        ;;

    "change_password")
        if ! check_auth; then
            send_unauthorized
            exit 1
        fi

        read -n $CONTENT_LENGTH POST_DATA
        local auth_token username old_password new_password new_hash
        
        auth_token=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
        username=$(get_current_username "$auth_token")
        old_password=$(echo "$POST_DATA" | grep -oP 'old_password=\K[^&]+' | urldecode)
        new_password=$(echo "$POST_DATA" | grep -oP 'new_password=\K[^&]+' | urldecode)

        if check_auth_or_password "$username" "$old_password"; then
            new_hash=$(echo -n "$new_password" | md5sum | cut -d' ' -f1)
            sed -i "s/^$username:.*/$username:$new_hash/" "$CONFIG_FILE"

            echo "Content-type: application/json"
            echo "Status: 200"
            echo
            echo '0'
        else
            echo "Content-type: application/json"
            echo "Status: 401"
            echo
            echo '1'
        fi
        ;;

    "change_username")
        if ! check_auth; then
            send_unauthorized
            exit 1
        fi

        read -n $CONTENT_LENGTH POST_DATA
        local auth_token current_username new_username password
        
        auth_token=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
        current_username=$(get_current_username "$auth_token")
        new_username=$(echo "$POST_DATA" | grep -oP 'new_username=\K[^&]+' | urldecode)
        password=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | urldecode)

        if check_auth_or_password "$current_username" "$password"; then
            if grep -q "^$new_username:" "$CONFIG_FILE"; then
                echo "Content-type: application/json"
                echo "Status: 409"
                echo
                echo '1'
            else
                sed -i "s/^$current_username:/$new_username:/" "$CONFIG_FILE"
                sed -i "s/:$current_username:/:$new_username:/" "$SESSION_FILE"

                echo "Content-type: application/json"
                echo "Status: 200"
                echo
                echo '0'
            fi
        else
            echo "Content-type: application/json"
            echo "Status: 401"
            echo
            echo '2'
        fi
        ;;

    *)
        echo "Content-type: application/json"
        echo "Status: 400"
        echo
        echo '1'
        ;;
esac
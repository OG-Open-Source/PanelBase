#!/bin/bash

# 配置文件路徑
CONFIG_FILE="/opt/panelbase/config/users.conf"
SESSION_FILE="/opt/panelbase/config/sessions.conf"

# 檢查認證
check_auth() {
    local AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
    if [ -z "$AUTH_TOKEN" ]; then
        return 1
    fi
    
    # 檢查 session 是否有效
    local CURRENT_TIME=$(date +%s)
    if ! grep -q "^$AUTH_TOKEN:" "$SESSION_FILE" || \
       [ $(awk -F: -v token="$AUTH_TOKEN" -v time="$CURRENT_TIME" \
           '$1 == token && (time - $3) < 86400 {print 1}' "$SESSION_FILE") != "1" ]; then
        return 1
    fi
    
    return 0
}

# 檢查認證或密碼
check_auth_or_password() {
    local USERNAME="$1"
    local PASSWORD="$2"
    
    if [ -n "$USERNAME" ] && [ -n "$PASSWORD" ]; then
        local STORED_HASH=$(grep "^$USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
        local INPUT_HASH=$(echo -n "$PASSWORD" | md5sum | cut -d' ' -f1)
        
        if [ "$STORED_HASH" = "$INPUT_HASH" ]; then
            return 0
        fi
    fi
    
    return 1
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
    local AUTH_TOKEN="$1"
    awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE"
}

# 如果是被其他腳本導入，則不執行以下代碼
if [ "${BASH_SOURCE[0]}" != "$0" ]; then
    return 0
fi

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
        USERNAME=$(echo "$POST_DATA" | grep -oP 'username=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
        PASSWORD=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

        if check_auth_or_password "$USERNAME" "$PASSWORD"; then
            TOKEN=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)
            echo "$TOKEN:$USERNAME:$(date +%s)" >> "$SESSION_FILE"

            echo "Content-type: application/json"
            echo "Set-Cookie: auth_token=$TOKEN; Path=/; HttpOnly; SameSite=Strict; Max-Age=86400"
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
        AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
        if [ -n "$AUTH_TOKEN" ]; then
            sed -i "/^$AUTH_TOKEN:/d" "$SESSION_FILE"
        fi

        echo "Content-type: text/html"
        echo "Set-Cookie: auth_token=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT"
        echo "Status: 302"
        echo "Location: /"
        echo
        ;;

    "get_username")
        if check_auth; then
            AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
            USERNAME=$(get_current_username "$AUTH_TOKEN")
            echo "Content-type: application/json"
            echo "Status: 200"
            echo
            echo "$USERNAME"
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
        AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
        USERNAME=$(get_current_username "$AUTH_TOKEN")
        OLD_PASSWORD=$(echo "$POST_DATA" | grep -oP 'old_password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
        NEW_PASSWORD=$(echo "$POST_DATA" | grep -oP 'new_password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

        if check_auth_or_password "$USERNAME" "$OLD_PASSWORD"; then
            NEW_HASH=$(echo -n "$NEW_PASSWORD" | md5sum | cut -d' ' -f1)
            sed -i "s/^$USERNAME:.*/$USERNAME:$NEW_HASH/" "$CONFIG_FILE"

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
        AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
        CURRENT_USERNAME=$(get_current_username "$AUTH_TOKEN")
        NEW_USERNAME=$(echo "$POST_DATA" | grep -oP 'new_username=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
        PASSWORD=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

        if check_auth_or_password "$CURRENT_USERNAME" "$PASSWORD"; then
            if grep -q "^$NEW_USERNAME:" "$CONFIG_FILE"; then
                echo "Content-type: application/json"
                echo "Status: 409"
                echo
                echo '1'
            else
                sed -i "s/^$CURRENT_USERNAME:/$NEW_USERNAME:/" "$CONFIG_FILE"
                sed -i "s/:$CURRENT_USERNAME:/:$NEW_USERNAME:/" "$SESSION_FILE"

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
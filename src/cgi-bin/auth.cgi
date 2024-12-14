#!/bin/bash

if [ "$REQUEST_METHOD" = "POST" ]; then
    read -n $CONTENT_LENGTH POST_DATA
fi

CONFIG_FILE="/opt/panelbase/config/users.conf"
SESSION_FILE="/opt/panelbase/config/sessions.conf"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "admin:$(echo -n "admin" | md5sum | cut -d' ' -f1)" > "$CONFIG_FILE"
    chmod 600 "$CONFIG_FILE"
fi

AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')

ACTION=$(echo "$QUERY_STRING" | grep -oP 'action=\K[^&]+')

case "$ACTION" in
    "login")
        USERNAME=$(echo "$POST_DATA" | grep -oP 'username=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
        PASSWORD=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

        STORED_HASH=$(grep "^$USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
        INPUT_HASH=$(echo -n "$PASSWORD" | md5sum | cut -d' ' -f1)

        if [ "$STORED_HASH" = "$INPUT_HASH" ]; then
            TOKEN=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)
            
            echo "$TOKEN:$USERNAME:$(date +%s)" >> "$SESSION_FILE"
            
            EXPIRY=$(($(date +%s) + 86400))
            echo "Content-type: text/plain"
            echo "Set-Cookie: auth_token=$TOKEN; Path=/; HttpOnly; SameSite=Strict; Max-Age=86400; Expires=$(date -u -d "@$EXPIRY" "+%a, %d %b %Y %H:%M:%S GMT")"
            echo "Status: 200"
            echo
            echo "OK"
        else
            echo "Content-type: text/plain"
            echo "Status: 401"
            echo
            echo "Unauthorized"
        fi
        ;;

    "logout")
        echo "Content-type: text/plain"
        echo "Set-Cookie: auth_token=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0"
        echo "Status: 200"
        echo
        echo "OK"
        ;;

    "get_username")
        if [ -n "$AUTH_TOKEN" ]; then
            USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
            echo "Content-type: text/plain"
            echo
            echo "$USERNAME"
        else
            echo "Content-type: text/plain"
            echo "Status: 401"
            echo
            echo "Unauthorized"
        fi
        ;;

    "change_password")
        if [ -n "$AUTH_TOKEN" ]; then
            USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
            OLD_PASSWORD=$(echo "$POST_DATA" | grep -oP 'old_password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
            NEW_PASSWORD=$(echo "$POST_DATA" | grep -oP 'new_password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

            # 驗證舊密碼
            STORED_HASH=$(grep "^$USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
            OLD_HASH=$(echo -n "$OLD_PASSWORD" | md5sum | cut -d' ' -f1)

            if [ "$STORED_HASH" = "$OLD_HASH" ]; then
                # 更新密碼
                NEW_HASH=$(echo -n "$NEW_PASSWORD" | md5sum | cut -d' ' -f1)
                sed -i "s/^$USERNAME:.*/$USERNAME:$NEW_HASH/" "$CONFIG_FILE"
                
                echo "Content-type: text/plain"
                echo "Status: 200"
                echo
                echo "Password updated"
            else
                echo "Content-type: text/plain"
                echo "Status: 401"
                echo
                echo "Invalid old password"
            fi
        else
            echo "Content-type: text/plain"
            echo "Status: 401"
            echo
            echo "Unauthorized"
        fi
        ;;

    "change_username")
        if [ -n "$AUTH_TOKEN" ]; then
            CURRENT_USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
            NEW_USERNAME=$(echo "$POST_DATA" | grep -oP 'new_username=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
            PASSWORD=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

            STORED_HASH=$(grep "^$CURRENT_USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
            INPUT_HASH=$(echo -n "$PASSWORD" | md5sum | cut -d' ' -f1)

            if [ "$STORED_HASH" = "$INPUT_HASH" ]; then
                if grep -q "^$NEW_USERNAME:" "$CONFIG_FILE"; then
                    echo "Content-type: text/plain"
                    echo "Status: 409"
                    echo
                    echo "Username already exists"
                else
                    sed -i "s/^$CURRENT_USERNAME:/$NEW_USERNAME:/" "$CONFIG_FILE"
                    # 更新會話文件中的用戶名
                    sed -i "s/:$CURRENT_USERNAME:/:$NEW_USERNAME:/" "$SESSION_FILE"
                    
                    echo "Content-type: text/plain"
                    echo "Status: 200"
                    echo
                    echo "Username updated"
                fi
            else
                echo "Content-type: text/plain"
                echo "Status: 401"
                echo
                echo "Invalid password"
            fi
        else
            echo "Content-type: text/plain"
            echo "Status: 401"
            echo
            echo "Unauthorized"
        fi
        ;;

    *)
        echo "Content-type: text/plain"
        echo "Status: 400"
        echo
        echo "Unknown action"
        ;;
esac
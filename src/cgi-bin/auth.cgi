#!/bin/bash

if [ "$REQUEST_METHOD" = "POST" ]; then
	read -n $CONTENT_LENGTH POST_DATA
fi

CONFIG_FILE="/opt/panelbase/config/users.conf"
SESSION_FILE="/opt/panelbase/config/sessions.conf"
THEME_FILE="/opt/panelbase/config/themes.conf"

for FILE in "$CONFIG_FILE" "$SESSION_FILE" "$THEME_FILE"; do
	if [ ! -f "$FILE" ]; then
		touch "$FILE"
		chmod 600 "$FILE"
		chown www-data:www-data "$FILE"
	fi
done

AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
ACTION=$(echo "$QUERY_STRING" | grep -oP 'action=\K[^&]+')

create_session() {
	local username="$1"
	local current_time=$(date +%s)

	sed -i "/:[^:]*$username:/d" "$SESSION_FILE"

	local token=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)

	echo "$token:$username:$current_time" >> "$SESSION_FILE"

	local expiry=$((current_time + 86400))
	echo "Set-Cookie: auth_token=$token; Path=/; HttpOnly; SameSite=Strict; Max-Age=86400; Expires=$(date -u -d "@$expiry" "+%a, %d %b %Y %H:%M:%S GMT")"
}

cleanup_sessions() {
	local current_time=$(date +%s)
	local temp_file=$(mktemp)

	awk -F: -v time="$current_time" '(time - $3) < 86400 {print $0}' "$SESSION_FILE" > "$temp_file"
	mv "$temp_file" "$SESSION_FILE"
	chmod 600 "$SESSION_FILE"
}

case "$ACTION" in
	"login")
		USERNAME=$(echo "$POST_DATA" | grep -oP 'username=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
		PASSWORD=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

		if ! [[ "$USERNAME" =~ ^[A-Za-z0-9]+$ ]]; then
			echo "Content-type: application/json"
			echo "Status: 400"
			echo
			echo '{"status":"error","code":"invalid_username","message":"Invalid username format"}'
			exit 0
		fi

		if ! [[ "$PASSWORD" =~ ^[A-Za-z0-9!@$]+$ ]]; then
			echo "Content-type: application/json"
			echo "Status: 400"
			echo
			echo '{"status":"error","code":"invalid_password","message":"Invalid password format"}'
			exit 0
		fi

		STORED_HASH=$(grep "^$USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
		INPUT_HASH=$(echo -n "$PASSWORD" | md5sum | cut -d' ' -f1)

		if [ "$STORED_HASH" = "$INPUT_HASH" ]; then
			cleanup_sessions
			create_session "$USERNAME"

			echo "Content-type: application/json"
			echo "Status: 200"
			echo
			echo '{"status":"success","message":"Login successful"}'
		else
			sleep 1
			echo "Content-type: application/json"
			echo "Status: 401"
			echo
			echo '{"status":"error","code":"invalid_credentials","message":"Invalid username or password"}'
		fi
		;;

	"logout")
		if [ -n "$AUTH_TOKEN" ]; then
			sed -i "/^$AUTH_TOKEN:/d" "$SESSION_FILE"
			echo "Content-type: application/json"
			echo "Set-Cookie: auth_token=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT"
			echo "Status: 200"
			echo
			echo '{"status":"success","message":"Logout successful"}'
		fi
		;;

	"get_username")
		if [ -n "$AUTH_TOKEN" ]; then
			USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
			if [ -n "$USERNAME" ]; then
				echo "Content-type: application/json"
				echo "Status: 200"
				echo
				echo "{\"status\":\"success\",\"username\":\"$USERNAME\"}"
			else
				echo "Content-type: application/json"
				echo "Status: 401"
				echo
				echo '{"status":"error","code":"invalid_session","message":"Invalid session"}'
			fi
		else
			echo "Content-type: application/json"
			echo "Status: 401"
			echo
			echo '{"status":"error","code":"no_session","message":"No session found"}'
		fi
		;;

	"change_username")
		if [ -n "$AUTH_TOKEN" ]; then
			CURRENT_USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
			NEW_USERNAME=$(echo "$POST_DATA" | grep -oP 'new_username=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
			PASSWORD=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

			if ! [[ $NEW_USERNAME =~ ^[A-Za-z0-9]+$ ]]; then
				echo "Content-type: application/json"
				echo "Status: 400"
				echo
				echo '{"status":"error","code":"invalid_username","message":"Invalid username format"}'
				exit 0
			fi

			if ! [[ $PASSWORD =~ ^[A-Za-z0-9!@$]+$ ]]; then
				echo "Content-type: application/json"
				echo "Status: 400"
				echo
				echo '{"status":"error","code":"invalid_password","message":"Invalid password format"}'
				exit 0
			fi

			STORED_HASH=$(grep "^$CURRENT_USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
			INPUT_HASH=$(echo -n "$PASSWORD" | md5sum | cut -d' ' -f1)

			if [ "$STORED_HASH" = "$INPUT_HASH" ]; then
				if grep -q "^$NEW_USERNAME:" "$CONFIG_FILE"; then
					echo "Content-type: application/json"
					echo "Status: 409"
					echo
					echo '{"status":"error","code":"username_exists","message":"Username already exists"}'
				else
					sed -i "s/^$CURRENT_USERNAME:/$NEW_USERNAME:/" "$CONFIG_FILE"
					sed -i "s/:$CURRENT_USERNAME:/:$NEW_USERNAME:/" "$SESSION_FILE"
					sed -i "s/^$CURRENT_USERNAME:/$NEW_USERNAME:/" "$THEME_FILE"

					echo "Content-type: application/json"
					echo "Status: 200"
					echo
					echo '{"status":"success","message":"Username changed successfully"}'
				fi
			else
				sleep 1
				echo "Content-type: application/json"
				echo "Status: 401"
				echo
				echo '{"status":"error","code":"invalid_password","message":"Invalid password"}'
			fi
		else
			echo "Content-type: application/json"
			echo "Status: 401"
			echo
			echo '{"status":"error","code":"no_session","message":"No session found"}'
		fi
		;;

	*)
		echo "Content-type: application/json"
		echo "Status: 400"
		echo
		echo '{"status":"error","code":"invalid_action","message":"Invalid action"}'
		;;
esac
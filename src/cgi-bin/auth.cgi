#!/bin/bash

if [ "$REQUEST_METHOD" = "POST" ]; then
	read -n $CONTENT_LENGTH POST_DATA
fi

CONFIG_FILE="/opt/panelbase/config/user.conf"
SESSION_FILE="/opt/panelbase/config/sessions.conf"

for FILE in "$CONFIG_FILE" "$SESSION_FILE"; do
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
	echo "$token:$username:$current_time" >>"$SESSION_FILE"

	local expiry=$((current_time + 86400))
	echo "Set-Cookie: auth_token=$token; Path=/; HttpOnly; SameSite=Strict; Max-Age=86400; Expires=$(date -u -d "@$expiry" "+%a, %d %b %Y %H:%M:%S GMT")"
}

cleanup_sessions() {
	local current_time=$(date +%s)
	local temp_file=$(mktemp)

	awk -F: -v time="$current_time" '(time - $3) < 86400 {print $0}' "$SESSION_FILE" >"$temp_file"
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
		echo '{"status":"error","code":"400","message":"Invalid username format"}'
		exit 0
	fi

	if ! [[ "$PASSWORD" =~ ^[A-Za-z0-9!@$]+$ ]]; then
		echo "Content-type: application/json"
		echo "Status: 400"
		echo
		echo '{"status":"error","code":"400","message":"Invalid password format"}'
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
		echo '{"status":"success","code":"200","message":"Login successful"}'
	else
		sleep 1
		echo "Content-type: application/json"
		echo "Status: 401"
		echo
		echo '{"status":"error","code":"401","message":"Invalid username or password"}'
	fi
	;;

"logout")
	if [ -n "$AUTH_TOKEN" ]; then
		sed -i "/^$AUTH_TOKEN:/d" "$SESSION_FILE"
		echo "Content-type: application/json"
		echo "Set-Cookie: auth_token=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT"
		echo "Status: 200"
		echo
		echo '{"status":"success","code":"200","message":"Logout successful"}'
	fi
	;;

"username")
	if [ -n "$AUTH_TOKEN" ]; then
		USERNAME=$(grep "^$AUTH_TOKEN:" "$SESSION_FILE" | cut -d: -f2)
		if [ -n "$USERNAME" ]; then
			echo "Content-type: application/json"
			echo "Status: 200"
			echo
			echo "{\"status\":\"success\",\"code\":\"200\",\"message\":\"$USERNAME\"}"
		else
			echo "Content-type: application/json"
			echo "Status: 401"
			echo
			echo '{"status":"error","code":"401","message":"Invalid session"}'
		fi
	else
		echo "Content-type: application/json"
		echo "Status: 401"
		echo
		echo '{"status":"error","code":"401","message":"No session found"}'
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
			echo '{"status":"error","code":"400","message":"Invalid username format"}'
			exit 0
		fi

		if ! [[ $PASSWORD =~ ^[A-Za-z0-9!@$]+$ ]]; then
			echo "Content-type: application/json"
			echo "Status: 400"
			echo
			echo '{"status":"error","code":"400","message":"Invalid password format"}'
			exit 0
		fi

		STORED_HASH=$(grep "^$CURRENT_USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
		INPUT_HASH=$(echo -n "$PASSWORD" | md5sum | cut -d' ' -f1)

		if [ "$STORED_HASH" = "$INPUT_HASH" ]; then
			if grep -q "^$NEW_USERNAME:" "$CONFIG_FILE"; then
				echo "Content-type: application/json"
				echo "Status: 409"
				echo
				echo '{"status":"error","code":"409","message":"Username already exists"}'
			else
				sed -i "s/^$CURRENT_USERNAME:/$NEW_USERNAME:/" "$CONFIG_FILE"
				sed -i "s/:$CURRENT_USERNAME:/:$NEW_USERNAME:/" "$SESSION_FILE"

				echo "Content-type: application/json"
				echo "Status: 200"
				echo
				echo '{"status":"success","code":"200","message":"Username changed successfully"}'
			fi
		else
			sleep 1
			echo "Content-type: application/json"
			echo "Status: 401"
			echo
			echo '{"status":"error","code":"401","message":"Invalid password"}'
		fi
	else
		echo "Content-type: application/json"
		echo "Status: 401"
		echo
		echo '{"status":"error","code":"401","message":"No session found"}'
	fi
	;;

*)
	echo "Content-type: application/json"
	echo "Status: 400"
	echo
	echo '{"status":"error","code":"400","message":"Invalid action"}'
	;;
esac

#!/bin/bash

if [ "$REQUEST_METHOD" = "POST" ]; then
	read -n $CONTENT_LENGTH POST_DATA
fi

source "${INSTALL_DIR:-/opt/panelbase}/config/security.conf"
CONFIG_FILE="${INSTALL_DIR:-/opt/panelbase}/config/users.conf"
SESSION_FILE="${INSTALL_DIR:-/opt/panelbase}/config/sessions.conf"
THEME_FILE="${INSTALL_DIR:-/opt/panelbase}/config/themes.conf"

for FILE in "$CONFIG_FILE" "$SESSION_FILE" "$THEME_FILE"; do
	if [ ! -f "$FILE" ]; then
		touch "$FILE"
		chmod "$SECURE_FILE_PERMISSION" "$FILE"
		[ -n "$SUDO_USER" ] && chown "$SUDO_USER" "$FILE"
	fi
done

AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
ACTION=$(echo "$QUERY_STRING" | grep -oP 'action=\K[^&]+')

create_session() {
	local username="$1"
	local current_time=$(date +%s)

	local session_count=$(grep ":$username:" "$SESSION_FILE" | wc -l)
	if [ "$session_count" -ge "$MAX_SESSIONS_PER_USER" ]; then
		local oldest_session=$(grep ":$username:" "$SESSION_FILE" | sort -t: -k3 | head -n1)
		local oldest_token=$(echo "$oldest_session" | cut -d: -f1)
		sed -i "/^$oldest_token:/d" "$SESSION_FILE"
	fi
	sed -i "/:[^:]*$username:/d" "$SESSION_FILE"

	local token=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c "$SESSION_TOKEN_LENGTH")

	echo "$token:$username:$current_time" >> "$SESSION_FILE"

	local expiry=$((current_time + SESSION_LIFETIME))
	echo "Set-Cookie: auth_token=$token; Path=/; HttpOnly; SameSite=Strict; Max-Age=$SESSION_LIFETIME; Expires=$(date -u -d "@$expiry" "+%a, %d %b %Y %H:%M:%S GMT")"
}

cleanup_sessions() {
	local current_time=$(date +%s)
	local temp_file=$(mktemp)

	awk -F: -v time="$current_time" -v max_age="$SESSION_LIFETIME" '
		(time - $3) < max_age {print $0}
	' "$SESSION_FILE" > "$temp_file"

	mv "$temp_file" "$SESSION_FILE"
	chmod "$SECURE_FILE_PERMISSION" "$SESSION_FILE"
}

case "$ACTION" in
	"login")
		if ! check_rate_limit "$REMOTE_ADDR"; then
			log_security_event "WARN" "Rate limit exceeded for IP: $REMOTE_ADDR"
			echo "Content-type: application/json"
			echo "Status: 429"
			echo
			echo '{"error": "rate_limit_exceeded"}'
			exit 0
		fi

		USERNAME=$(echo "$POST_DATA" | grep -oP 'username=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
		PASSWORD=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

		if ! [[ "$USERNAME" =~ ^[A-Za-z0-9]+$ ]]; then
			log_security_event "WARN" "Invalid username format attempt: $USERNAME"
			echo "Content-type: application/json"
			echo "Status: 400"
			echo
			echo '{"error": "invalid_username_format"}'
			exit 0
		fi

		if ! [[ "$PASSWORD" =~ ^[A-Za-z0-9!@$]+$ ]]; then
			log_security_event "WARN" "Invalid password format attempt for user: $USERNAME"
			echo "Content-type: application/json"
			echo "Status: 400"
			echo
			echo '{"error": "invalid_password_format"}'
			exit 0
		fi

		STORED_HASH=$(grep "^$USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
		INPUT_HASH=$(echo -n "$PASSWORD" | md5sum | cut -d' ' -f1)

		if [ "$STORED_HASH" = "$INPUT_HASH" ]; then
			cleanup_sessions
			create_session "$USERNAME"
			log_security_event "INFO" "Successful login for user: $USERNAME"

			echo "Content-type: application/json"
			echo "Status: 200"
			echo
			echo '0'
		else
			log_security_event "WARN" "Failed login attempt for user: $USERNAME"
			sleep 1
			echo "Content-type: application/json"
			echo "Status: 401"
			echo
			echo '1'
		fi
		;;

	"logout")
		if [ -n "$AUTH_TOKEN" ]; then
			USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
			sed -i "/^$AUTH_TOKEN:/d" "$SESSION_FILE"
			log_security_event "INFO" "User logged out: $USERNAME"
		fi

		echo "Content-type: text/html"
		echo "Set-Cookie: auth_token=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT"
		echo "Status: 302"
		echo "Location: /"
		echo
		;;

	"get_username")
		if [ -n "$AUTH_TOKEN" ]; then
			USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
			if [ -n "$USERNAME" ]; then
				echo "Content-type: application/json"
				echo "Status: 200"
				echo
				echo "$USERNAME"
			else
				echo "Content-type: application/json"
				echo "Status: 401"
				echo
				echo '1'
			fi
		else
			echo "Content-type: application/json"
			echo "Status: 401"
			echo
			echo '1'
		fi
		;;

	"set_theme")
		if [ -n "$AUTH_TOKEN" ]; then
			USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
			THEME=$(echo "$POST_DATA" | grep -oP 'theme=\K[^&]+')

			if [ "$THEME" = "dark" ] || [ "$THEME" = "light" ]; then
				sed -i "/^$USERNAME:/d" "$THEME_FILE"
				echo "$USERNAME:$THEME" >> "$THEME_FILE"

				echo "Content-type: application/json"
				echo "Status: 200"
				echo
				echo '0'
			else
				echo "Content-type: application/json"
				echo "Status: 400"
				echo
				echo '1'
			fi
		else
			echo "Content-type: application/json"
			echo "Status: 401"
			echo
			echo '2'
		fi
		;;

	"get_theme")
		if [ -n "$AUTH_TOKEN" ]; then
			USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
			THEME=$(grep "^$USERNAME:" "$THEME_FILE" | cut -d: -f2)

			echo "Content-type: application/json"
			echo "Status: 200"
			echo
			if [ -n "$THEME" ]; then
				echo "\"$THEME\""
			else
				echo "null"
			fi
		else
			echo "Content-type: application/json"
			echo "Status: 401"
			echo
			echo '1'
		fi
		;;

	"change_password")
		if [ -n "$AUTH_TOKEN" ]; then
			USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
			OLD_PASSWORD=$(echo "$POST_DATA" | grep -oP 'old_password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
			NEW_PASSWORD=$(echo "$POST_DATA" | grep -oP 'new_password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

			if ! [[ $OLD_PASSWORD =~ ^[A-Za-z0-9!@$]+$ ]]; then
				echo "Content-type: application/json"
				echo "Status: 400"
				echo
				echo '{"error": "invalid_old_password_format"}'
				exit 0
			fi

			if ! [[ $NEW_PASSWORD =~ ^[A-Za-z0-9!@$]+$ ]]; then
				echo "Content-type: application/json"
				echo "Status: 400"
				echo
				echo '{"error": "invalid_new_password_format"}'
				exit 0
			fi

			if [ ${#NEW_PASSWORD} -lt 6 ]; then
				echo "Content-type: application/json"
				echo "Status: 400"
				echo
				echo '{"error": "password_too_short"}'
				exit 0
			fi

			STORED_HASH=$(grep "^$USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
			OLD_HASH=$(echo -n "$OLD_PASSWORD" | md5sum | cut -d' ' -f1)

			if [ "$STORED_HASH" = "$OLD_HASH" ]; then
				NEW_HASH=$(echo -n "$NEW_PASSWORD" | md5sum | cut -d' ' -f1)
				sed -i "s/^$USERNAME:.*/$USERNAME:$NEW_HASH/" "$CONFIG_FILE"

				echo "Content-type: application/json"
				echo "Status: 200"
				echo
				echo '0'
			else
				sleep 1
				echo "Content-type: application/json"
				echo "Status: 401"
				echo
				echo '1'
			fi
		else
			echo "Content-type: application/json"
			echo "Status: 401"
			echo
			echo '2'
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
				echo '{"error": "invalid_username_format"}'
				exit 0
			fi

			if ! [[ $PASSWORD =~ ^[A-Za-z0-9!@$]+$ ]]; then
				echo "Content-type: application/json"
				echo "Status: 400"
				echo
				echo '{"error": "invalid_password_format"}'
				exit 0
			fi

			STORED_HASH=$(grep "^$CURRENT_USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
			INPUT_HASH=$(echo -n "$PASSWORD" | md5sum | cut -d' ' -f1)

			if [ "$STORED_HASH" = "$INPUT_HASH" ]; then
				if grep -q "^$NEW_USERNAME:" "$CONFIG_FILE"; then
					echo "Content-type: application/json"
					echo "Status: 409"
					echo
					echo '1'
				else
					sed -i "s/^$CURRENT_USERNAME:/$NEW_USERNAME:/" "$CONFIG_FILE"
					sed -i "s/:$CURRENT_USERNAME:/:$NEW_USERNAME:/" "$SESSION_FILE"
					sed -i "s/^$CURRENT_USERNAME:/$NEW_USERNAME:/" "$THEME_FILE"

					echo "Content-type: application/json"
					echo "Status: 200"
					echo
					echo '0'
				fi
			else
				sleep 1
				echo "Content-type: application/json"
				echo "Status: 401"
				echo
				echo '2'
			fi
		else
			echo "Content-type: application/json"
			echo "Status: 401"
			echo
			echo '3'
		fi
		;;

	*)
		log_security_event "WARN" "Invalid action requested: $ACTION"
		echo "Content-type: application/json"
		echo "Status: 400"
		echo
		echo '{"error": "invalid_action"}'
		;;
esac
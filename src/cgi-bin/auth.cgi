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

case "$ACTION" in
	"login")
		USERNAME=$(echo "$POST_DATA" | grep -oP 'username=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
		PASSWORD=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

		if ! [[ "$USERNAME" =~ ^[A-Za-z0-9]+$ ]]; then
			echo "Content-type: application/json"
			echo "Status: 400"
			echo
			echo '{"error": "invalid_username_format"}'
			exit 0
		fi

		if ! [[ "$PASSWORD" =~ ^[A-Za-z0-9!@$]+$ ]]; then
			echo "Content-type: application/json"
			echo "Status: 400"
			echo
			echo '{"error": "invalid_password_format"}'
			exit 0
		fi

		STORED_HASH=$(grep "^$USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
		INPUT_HASH=$(echo -n "$PASSWORD" | md5sum | cut -d' ' -f1)

		if [ "$STORED_HASH" = "$INPUT_HASH" ]; then
			sed -i "/^.*:$USERNAME:/d" "$SESSION_FILE"

			TOKEN=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)

			echo "$TOKEN:$USERNAME:$(date +%s)" >> "$SESSION_FILE"

			EXPIRY=$(($(date +%s) + 86400))

			echo "Content-type: application/json"
			echo "Set-Cookie: auth_token=$TOKEN; Path=/; HttpOnly; SameSite=Strict; Max-Age=86400; Expires=$(date -u -d "@$EXPIRY" "+%a, %d %b %Y %H:%M:%S GMT")"
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
		;;

	"logout")
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
		if [ -n "$AUTH_TOKEN" ]; then
			USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
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
		echo "Content-type: application/json"
		echo "Status: 400"
		echo
		echo '1'
		;;
esac
#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

readonly CONFIG_DIR="/opt/panelbase/src/config"
readonly CONFIG_FILE="${CONFIG_DIR}/users.conf"
readonly SESSION_FILE="${CONFIG_DIR}/sessions.conf"
readonly SESSION_TIMEOUT=86400

init_files() {
	local files=("$CONFIG_FILE" "$SESSION_FILE")
	for file in "${files[@]}"; do
		if [ ! -f "$file" ]; then
			touch "$file"
			chmod 600 "$file"
			chown www-data:www-data "$file"
		fi
	done
}

validate_username() {
	local username=$1
	if ! [[ "$username" =~ ^[A-Za-z0-9]+$ ]]; then
		send_json_response 400 '{"error": "invalid_username_format"}'
		exit 0
	fi
}

validate_password() {
	local password=$1
	if ! [[ "$password" =~ ^[A-Za-z0-9!@$]+$ ]]; then
		send_json_response 400 '{"error": "invalid_password_format"}'
		exit 0
	fi
}

send_json_response() {
	local status=$1
	local content=$2
	echo "Content-type: application/json"
	echo "Status: $status"
	echo
	echo "$content"
}

send_redirect_response() {
	local location=$1
	echo "Content-type: text/html"
	echo "Status: 302"
	echo "Location: $location"
	echo
}

set_auth_cookie() {
	local token=$1
	local expiry=$(($(date +%s) + SESSION_TIMEOUT))
	echo "Set-Cookie: auth_token=$token; Path=/; HttpOnly; SameSite=Strict; Max-Age=$SESSION_TIMEOUT; Expires=$(date -u -d "@$expiry" "+%a, %d %b %Y %H:%M:%S GMT")"
}

clear_auth_cookie() {
	echo "Set-Cookie: auth_token=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT"
}

handle_login() {
	local username password stored_hash input_hash token

	username=$(echo "$POST_DATA" | grep -oP 'username=\K[^&]+' | sed 's/%40/@/g;s/%2B/+/g;s/%20/ /g')
	password=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g;s/%2B/+/g;s/%20/ /g')

	validate_username "$username"
	validate_password "$password"

	stored_hash=$(grep "^$username:" "$CONFIG_FILE" | cut -d: -f2)
	input_hash=$(echo -n "$password" | md5sum | cut -d' ' -f1)

	if [ "$stored_hash" = "$input_hash" ]; then
		sed -i "/^.*:$username:/d" "$SESSION_FILE"

		token=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)
		echo "$token:$username:$(date +%s)" >> "$SESSION_FILE"

		echo "Content-type: application/json"
		set_auth_cookie "$token"
		echo "Status: 200"
		echo
		echo '0'
	else
		sleep 1
		send_json_response 401 '1'
	fi
}

handle_logout() {
	if [ -n "$AUTH_TOKEN" ]; then
		sed -i "/^$AUTH_TOKEN:/d" "$SESSION_FILE"
	fi

	echo "Content-type: text/html"
	clear_auth_cookie
	send_redirect_response "/"
}

handle_get_username() {
	if [ -n "$AUTH_TOKEN" ]; then
		local username
		username=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
		send_json_response 200 "\"$username\""
	else
		send_json_response 401 '1'
	fi
}

handle_change_password() {
	if [ -n "$AUTH_TOKEN" ]; then
		local username old_password new_password stored_hash old_hash new_hash
		
		username=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
		old_password=$(echo "$POST_DATA" | grep -oP 'old_password=\K[^&]+' | sed 's/%40/@/g;s/%2B/+/g;s/%20/ /g')
		new_password=$(echo "$POST_DATA" | grep -oP 'new_password=\K[^&]+' | sed 's/%40/@/g;s/%2B/+/g;s/%20/ /g')

		validate_password "$old_password"
		validate_password "$new_password"

		if [ ${#new_password} -lt 6 ]; then
			send_json_response 400 '{"error": "password_too_short"}'
			exit 0
		fi

		stored_hash=$(grep "^$username:" "$CONFIG_FILE" | cut -d: -f2)
		old_hash=$(echo -n "$old_password" | md5sum | cut -d' ' -f1)

		if [ "$stored_hash" = "$old_hash" ]; then
			new_hash=$(echo -n "$new_password" | md5sum | cut -d' ' -f1)
			sed -i "s/^$username:.*/$username:$new_hash/" "$CONFIG_FILE"
			send_json_response 200 '0'
		else
			sleep 1
			send_json_response 401 '1'
		fi
	else
		send_json_response 401 '2'
	fi
}

handle_change_username() {
	if [ -n "$AUTH_TOKEN" ]; then
		local current_username new_username password stored_hash input_hash
		current_username=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "$SESSION_FILE")
		new_username=$(echo "$POST_DATA" | grep -oP 'new_username=\K[^&]+' | sed 's/%40/@/g;s/%2B/+/g;s/%20/ /g')
		password=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g;s/%2B/+/g;s/%20/ /g')

		validate_username "$new_username"
		validate_password "$password"

		stored_hash=$(grep "^$current_username:" "$CONFIG_FILE" | cut -d: -f2)
		input_hash=$(echo -n "$password" | md5sum | cut -d' ' -f1)

		if [ "$stored_hash" = "$input_hash" ]; then
			if grep -q "^$new_username:" "$CONFIG_FILE"; then
				send_json_response 409 '1'
			else
				sed -i "s/^$current_username:/$new_username:/" "$CONFIG_FILE"
				sed -i "s/:$current_username:/:$new_username:/" "$SESSION_FILE"
				send_json_response 200 '0'
			fi
		else
			sleep 1
			send_json_response 401 '2'
		fi
	else
		send_json_response 401 '3'
	fi
}

main() {
	init_files

	if [ "$REQUEST_METHOD" = "POST" ]; then
		read -n "$CONTENT_LENGTH" POST_DATA
	fi

	AUTH_TOKEN=$(echo "${HTTP_COOKIE:-}" | grep -oP 'auth_token=\K[^;]+' || echo "")

	ACTION=$(echo "$QUERY_STRING" | grep -oP 'action=\K[^&]+')

	case "$ACTION" in
		"login") handle_login ;;
		"logout") handle_logout ;;
		"get_username") handle_get_username ;;
		"change_password") handle_change_password ;;
		"change_username") handle_change_username ;;
		*) send_json_response 400 '1' ;;
	esac
}

main
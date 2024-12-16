#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

readonly CONFIG_DIR="/opt/panelbase/config"
readonly SESSION_FILE="${CONFIG_DIR}/sessions.conf"
readonly THEME_FILE="${CONFIG_DIR}/themes.conf"
readonly SESSION_TIMEOUT=86400

check_required_files() {
	local files=("$SESSION_FILE" "$THEME_FILE")
	for file in "${files[@]}"; do
		if [ ! -f "$file" ]; then
			send_json_response 500 '{"error": "required_file_not_found"}'
			exit 1
		fi
	done
}

send_json_response() {
	local status=$1
	local content=$2
	echo "Content-type: application/json"
	echo "Status: $status"
	echo
	echo "$content"
}

validate_session() {
	local auth_token=$1
	local current_time
	current_time=$(date +%s)

	awk -F: -v token="$auth_token" -v time="$current_time" -v timeout="$SESSION_TIMEOUT" \
		'$1 == token && (time - $3) < timeout {print $2}' "$SESSION_FILE"
}

handle_set_theme() {
	local username=$1
	local theme
	theme=$(echo "$POST_DATA" | grep -oP 'theme=\K[^&]+')

	if [ "$theme" = "dark" ] || [ "$theme" = "light" ]; then
		sed -i "/^$username:/d" "$THEME_FILE"
		echo "$username:$theme" >> "$THEME_FILE"
		send_json_response 200 '0'
	else
		send_json_response 400 '{"error": "invalid_theme"}'
	fi
}

handle_get_theme() {
	local username=$1
	local theme
	theme=$(grep "^$username:" "$THEME_FILE" | cut -d: -f2)

	if [ -n "$theme" ]; then
		send_json_response 200 "\"$theme\""
	else
		send_json_response 200 "null"
	fi
}

main() {
	check_required_files

	if [ "$REQUEST_METHOD" = "POST" ]; then
		read -n "$CONTENT_LENGTH" POST_DATA
	fi

	local auth_token
	auth_token=$(echo "${HTTP_COOKIE:-}" | grep -oP 'auth_token=\K[^;]+' || echo "")

	if [ -z "$auth_token" ]; then
		send_json_response 401 '{"error": "unauthorized"}'
		exit 0
	fi

	local username
	username=$(validate_session "$auth_token")
	if [ -z "$username" ]; then
		send_json_response 401 '{"error": "session_expired"}'
		exit 0
	fi

	local action
	action=$(echo "$QUERY_STRING" | grep -oP 'action=\K[^&]+')

	case "$action" in
		"set_theme") handle_set_theme "$username" ;;
		"get_theme") handle_get_theme "$username" ;;
		*) send_json_response 400 '{"error": "invalid_action"}' ;;
	esac
}

main
#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

readonly CONFIG_DIR="/opt/panelbase/config"
readonly ROUTES_CONF="${CONFIG_DIR}/routes.conf"
readonly SESSION_FILE="${CONFIG_DIR}/sessions.conf"
readonly SESSION_TIMEOUT=86400
readonly DOCUMENT_ROOT="/opt/panelbase/www"

check_required_files() {
	local files=("$ROUTES_CONF" "$SESSION_FILE")
	for file in "${files[@]}"; do
		if [ ! -f "$file" ]; then
			echo "Status: 500"
			echo "Content-type: text/plain"
			echo
			echo "Error: Required file not found: $file"
			exit 1
		fi
	done
}

validate_session() {
	local auth_token=$1
	local current_time
	current_time=$(date +%s)

	awk -F: -v token="$auth_token" -v time="$current_time" -v timeout="$SESSION_TIMEOUT" \
		'$1 == token && (time - $3) < timeout {print $2}' "$SESSION_FILE"
}

handle_api_request() {
	local api_path=$1
	local route
	local method
	local command

	api_path=$(echo "$api_path" | cut -d'?' -f1)

	route=$(grep "^$api_path " "$ROUTES_CONF" || grep "^${api_path%/*}/[^[:space:]]* " "$ROUTES_CONF")

	if [ -n "$route" ]; then
		method=$(echo "$route" | awk '{print $2}')
		command=$(echo "$route" | cut -d' ' -f3-)

		if [ "$REQUEST_METHOD" = "$method" ]; then
			if echo "$api_path" | grep -q "/[^/]*$"; then
				local param
				param=$(echo "$api_path" | grep -o "/[^/]*$" | cut -c2-)
				command=$(echo "$command" | sed "s/\$1/$param/g")
			fi

			if [ "$REQUEST_METHOD" = "POST" ] && [ -n "${CONTENT_LENGTH:-}" ]; then
				local post_data
				read -n "$CONTENT_LENGTH" post_data

				IFS='&' read -ra params <<< "$post_data"
				for i in "${!params[@]}"; do
					local param_value
					param_value=$(echo "${params[$i]}" | cut -d'=' -f2)
					command=$(echo "$command" | sed "s/\$%$((i+1))/$param_value/g")
				done
			fi

			eval "$command"
			return 0
		fi
	fi

	return 1
}

show_404_page() {
	if echo "${HTTP_USER_AGENT:-}" | grep -qi "curl\|wget\|postman\|insomnia"; then
		echo "Content-type: text/plain"
		echo "Status: 404"
		echo
		echo "404 Not Found"
	else
		echo "Content-type: text/html"
		echo "Status: 404"
		echo
		cat "$DOCUMENT_ROOT/404.html"
	fi
}

handle_static_file() {
	local request_path=$1
	local file_path

	request_path=$(echo "$request_path" | cut -d'?' -f1)
	request_path=$(echo -e "${request_path//%/\\x}")
	file_path="${DOCUMENT_ROOT}${request_path}"

	if [ -f "$file_path" ]; then
		exec /opt/panelbase/cgi-bin/static.cgi
		return 0
	fi

	show_404_page
	return 1
}

main() {
	check_required_files

	if echo "$REQUEST_URI" | grep -q "^/api/"; then
		local auth_token
		auth_token=$(echo "${HTTP_COOKIE:-}" | grep -oP 'auth_token=\K[^;]+' || echo "")

		if [ -z "$auth_token" ]; then
			echo "Status: 401"
			echo "Content-type: application/json"
			echo
			echo '{"error": "unauthorized"}'
			exit 0
		fi

		local username
		username=$(validate_session "$auth_token")
		if [ -z "$username" ]; then
			echo "Status: 401"
			echo "Content-type: application/json"
			echo
			echo '{"error": "session_expired"}'
			exit 0
		fi

		handle_api_request "$REQUEST_URI"
	elif echo "$REQUEST_URI" | grep -q "^/cgi-bin/panel\.cgi"; then
		local auth_token
		auth_token=$(echo "${HTTP_COOKIE:-}" | grep -oP 'auth_token=\K[^;]+' || echo "")

		if [ -z "$auth_token" ]; then
			echo "Status: 302"
			echo "Location: /"
			echo
			exit 0
		fi

		local username
		username=$(validate_session "$auth_token")
		if [ -z "$username" ]; then
			echo "Status: 302"
			echo "Location: /"
			echo
			exit 0
		fi

		exec /opt/panelbase/cgi-bin/panel.cgi
	else
		handle_static_file "$REQUEST_URI"
	fi
}

main
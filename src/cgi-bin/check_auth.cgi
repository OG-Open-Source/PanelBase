#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

readonly CONFIG_DIR="/opt/panelbase/config"
readonly ROUTES_CONF="${CONFIG_DIR}/routes.conf"
readonly SESSION_FILE="${CONFIG_DIR}/sessions.conf"
readonly SESSION_TIMEOUT=86400
readonly DOCUMENT_ROOT="/opt/panelbase/www"

send_json_error() {
	local status=$1
	local code=$2
	local message=$3
	echo "Content-type: application/json"
	echo "Status: $status"
	echo "Cache-Control: no-store, no-cache, must-revalidate"
	echo "Pragma: no-cache"
	echo
	echo "{\"status\": \"error\", \"code\": \"$code\", \"message\": \"$message\"}"
}

send_redirect() {
	local location=$1
	echo "Status: 302"
	echo "Location: $location"
	echo "Cache-Control: no-store, no-cache, must-revalidate"
	echo "Pragma: no-cache"
	echo
}

check_required_files() {
	local files=("$ROUTES_CONF" "$SESSION_FILE")
	for file in "${files[@]}"; do
		if [ ! -f "$file" ]; then
			send_json_error 500 "file_not_found" "Required configuration file not found: $file"
			exit 1
		fi
		if [ ! -r "$file" ]; then
			send_json_error 500 "file_not_readable" "Configuration file not readable: $file"
			exit 1
		fi
	done
}

validate_session() {
	local auth_token=$1
	local current_time
	current_time=$(date +%s)

	if ! [ -w "$SESSION_FILE" ]; then
		send_json_error 500 "session_file_not_writable" "Session file is not writable"
		exit 1
	fi

	awk -F: -v token="$auth_token" -v time="$current_time" -v timeout="$SESSION_TIMEOUT" \
		'$1 == token && (time - $3) < timeout {print $2}' "$SESSION_FILE"
}

handle_api_request() {
	local api_path=$1
	local route
	local method
	local command

	api_path=$(echo "$api_path" | cut -d'?' -f1)

	if ! [ -r "$ROUTES_CONF" ]; then
		send_json_error 500 "routes_file_not_readable" "Routes configuration file not readable"
		exit 1
	fi

	route=$(grep "^$api_path " "$ROUTES_CONF" || grep "^${api_path%/*}/[^[:space:]]* " "$ROUTES_CONF")

	if [ -z "$route" ]; then
		send_json_error 404 "api_not_found" "API endpoint not found: $api_path"
		return 1
	fi

	method=$(echo "$route" | awk '{print $2}')
	if [ "$REQUEST_METHOD" != "$method" ]; then
		send_json_error 405 "method_not_allowed" "Method not allowed: $REQUEST_METHOD"
		return 1
	fi

	command=$(echo "$route" | cut -d' ' -f3-)

	if echo "$api_path" | grep -q "/[^/]*$"; then
		local param
		param=$(echo "$api_path" | grep -o "/[^/]*$" | cut -c2-)
		command=$(echo "$command" | sed "s/\$1/$param/g")
	fi

	if [ "$REQUEST_METHOD" = "POST" ] && [ -n "${CONTENT_LENGTH:-}" ]; then
		if [ "$CONTENT_LENGTH" -gt 1048576 ]; then
			send_json_error 413 "request_too_large" "Request body too large"
			return 1
		fi

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
}

show_404_page() {
	if echo "${HTTP_USER_AGENT:-}" | grep -qi "curl\|wget\|postman\|insomnia"; then
		echo "Content-type: text/plain"
		echo "Status: 404"
		echo "Cache-Control: no-store, no-cache, must-revalidate"
		echo "Pragma: no-cache"
		echo
		echo "404 Not Found"
	else
		if [ ! -f "$DOCUMENT_ROOT/404.html" ]; then
			send_json_error 500 "404_page_not_found" "404 page template not found"
			exit 1
		fi
		if [ ! -r "$DOCUMENT_ROOT/404.html" ]; then
			send_json_error 500 "404_page_not_readable" "404 page template not readable"
			exit 1
		fi
		echo "Content-type: text/html"
		echo "Status: 404"
		echo "Cache-Control: no-store, no-cache, must-revalidate"
		echo "Pragma: no-cache"
		echo
		cat "$DOCUMENT_ROOT/404.html"
	fi
}

check_auth() {
	local auth_token
	auth_token=$(echo "${HTTP_COOKIE:-}" | grep -oP 'auth_token=\K[^;]+' || echo "")

	if [ -z "$auth_token" ]; then
		return 1
	fi

	if ! [[ "$auth_token" =~ ^[A-Za-z0-9]+$ ]]; then
		send_json_error 400 "invalid_token_format" "Invalid authentication token format"
		exit 1
	fi

	local username
	username=$(validate_session "$auth_token")
	if [ -z "$username" ]; then
		return 1
	fi

	return 0
}

handle_static_file() {
	local request_path=$1
	local file_path

	request_path=$(echo "$request_path" | cut -d'?' -f1)
	request_path=$(echo -e "${request_path//%/\\x}")

	if echo "$request_path" | grep -q "\.\."; then
		send_json_error 400 "invalid_path" "Path traversal not allowed"
		exit 1
	fi

	if echo "$request_path" | grep -q "[<>\"'&\$]"; then
		send_json_error 400 "invalid_characters" "Invalid characters in path"
		exit 1
	fi

	if ! check_auth; then
		send_redirect "/"
		exit 0
	fi

	file_path="${DOCUMENT_ROOT}${request_path}"

	if [ ! -e "$file_path" ]; then
		show_404_page
		return 1
	fi

	if [ ! -f "$file_path" ]; then
		send_json_error 400 "not_a_file" "Requested path is not a file"
		return 1
	fi

	if [ ! -r "$file_path" ]; then
		send_json_error 403 "file_not_readable" "File not readable"
		return 1
	fi

	exec /opt/panelbase/cgi-bin/static.cgi
	return 0
}

main() {
	check_required_files

	if echo "$REQUEST_URI" | grep -q "^/api/"; then
		if ! check_auth; then
			send_json_error 401 "unauthorized" "Not logged in"
			exit 0
		fi

		handle_api_request "$REQUEST_URI"
	elif echo "$REQUEST_URI" | grep -q "^/cgi-bin/panel\.cgi"; then
		if ! check_auth; then
			send_redirect "/"
			exit 0
		fi

		if [ ! -x "/opt/panelbase/cgi-bin/panel.cgi" ]; then
			send_json_error 500 "panel_cgi_not_executable" "Panel CGI script not executable"
			exit 1
		fi

		exec /opt/panelbase/cgi-bin/panel.cgi
	else
		handle_static_file "$REQUEST_URI"
	fi
}

main
#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

readonly CONFIG_DIR="/opt/panelbase/config"
readonly ROUTES_CONF="${CONFIG_DIR}/routes.conf"
readonly SESSION_FILE="${CONFIG_DIR}/sessions.conf"
readonly SESSION_TIMEOUT=86400

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

	echo "Content-type: text/plain"
	echo "Status: 404"
	echo
	echo "404 Not Found"
	return 1
}

main() {
	check_required_files

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

	if echo "$REQUEST_URI" | grep -q "^/api/"; then
		handle_api_request "$REQUEST_URI"
	elif echo "$REQUEST_URI" | grep -q "^/cgi-bin/panel\.cgi"; then
		exec /opt/panelbase/cgi-bin/panel.cgi
	else
		exec /opt/panelbase/cgi-bin/static.cgi
	fi
}

main
#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

readonly DOCUMENT_ROOT="/opt/panelbase/www"
readonly DEFAULT_404_PAGE="${DOCUMENT_ROOT}/404.html"
readonly MIME_TYPES=(
	["html"]="text/html"
	["css"]="text/css"
	["js"]="application/javascript"
	["json"]="application/json"
	["png"]="image/png"
	["jpg"]="image/jpeg"
	["jpeg"]="image/jpeg"
	["gif"]="image/gif"
	["svg"]="image/svg+xml"
	["ico"]="image/x-icon"
	["woff"]="font/woff"
	["woff2"]="font/woff2"
	["ttf"]="font/ttf"
	["eot"]="application/vnd.ms-fontobject"
	["pdf"]="application/pdf"
	["zip"]="application/zip"
	["txt"]="text/plain"
)

send_404_response() {
	if echo "${HTTP_USER_AGENT:-}" | grep -qi "curl\|wget\|postman\|insomnia"; then
		echo "Content-type: text/plain"
		echo "Status: 404"
		echo
		echo "404 Not Found"
	else
		echo "Content-type: text/html"
		echo "Status: 404"
		echo
		cat "$DEFAULT_404_PAGE"
	fi
}

get_mime_type() {
	local file=$1
	local extension=${file##*.}
	local mime_type=${MIME_TYPES[${extension,,}]:-application/octet-stream}
	echo "$mime_type"
}

is_path_safe() {
	local path=$1
	if echo "$path" | grep -q "\.\."; then
		return 1
	fi
	if [[ "$path" != /* ]]; then
		return 1
	fi
	if echo "$path" | grep -q "[<>\"'&\$]"; then
		return 1
	fi
	return 0
}

handle_static_file() {
	local request_path=$1
	local file_path

	request_path=$(echo "$request_path" | cut -d'?' -f1)

	request_path=$(echo -e "${request_path//%/\\x}")

	if ! is_path_safe "$request_path"; then
		send_404_response
		return 1
	fi

	file_path="${DOCUMENT_ROOT}${request_path}"

	if [ -d "$file_path" ]; then
		if [ -f "${file_path}/index.html" ]; then
			file_path="${file_path}/index.html"
		else
			send_404_response
			return 1
		fi
	fi

	if [ ! -f "$file_path" ]; then
		send_404_response
		return 1
	fi

	if [ ! -r "$file_path" ]; then
		echo "Status: 403"
		echo "Content-type: text/plain"
		echo
		echo "403 Forbidden"
		return 1
	fi

	local mime_type
	mime_type=$(get_mime_type "$file_path")
	echo "Content-type: $mime_type"

	if [[ "$mime_type" =~ ^image/ || "$mime_type" =~ ^font/ ]]; then
		echo "Cache-Control: public, max-age=31536000"
	else
		echo "Cache-Control: no-cache"
	fi

	echo "X-Content-Type-Options: nosniff"
	echo "X-Frame-Options: SAMEORIGIN"
	echo "X-XSS-Protection: 1; mode=block"

	echo
	cat "$file_path"
	return 0
}

main() {
	if [ -z "${REQUEST_URI:-}" ]; then
		echo "Status: 500"
		echo "Content-type: text/plain"
		echo
		echo "500 Internal Server Error: Missing REQUEST_URI"
		exit 1
	fi

	handle_static_file "$REQUEST_URI"
}

main
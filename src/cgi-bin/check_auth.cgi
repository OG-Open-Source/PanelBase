#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

AUTH_TOKEN=$(echo "${HTTP_COOKIE:-}" | grep -oP 'auth_token=\K[^;]+' || echo "")
ORIGINAL_URL="$REQUEST_URI"
DOCUMENT_ROOT="/opt/panelbase/www"
SESSION_FILE="/opt/panelbase/config/sessions.conf"

if [ "$ORIGINAL_URL" = "/" ] || [ "$ORIGINAL_URL" = "/index.html" ]; then
	if [ -n "$AUTH_TOKEN" ] && [ -f "$SESSION_FILE" ]; then
		CURRENT_TIME=$(date +%s)
		VALID_SESSION=$(awk -F: -v token="$AUTH_TOKEN" -v time="$CURRENT_TIME" \
			'$1 == token && (time - $3) < 86400 {print $2}' "$SESSION_FILE")

		if [ -n "$VALID_SESSION" ]; then
			echo "Status: 302"
			echo "Location: /panel.html"
			echo
			exit 0
		fi
	fi
	echo "Status: 302"
	echo "Location: /index.html"
	echo
	exit 0
fi

if [ -z "$AUTH_TOKEN" ]; then
	echo "Status: 302"
	echo "Location: /"
	echo
	exit 0
fi

SESSION_FILE="/opt/panelbase/config/sessions.conf"
if [ ! -f "$SESSION_FILE" ]; then
	echo "Status: 302"
	echo "Location: /"
	echo
	exit 0
fi

CURRENT_TIME=$(date +%s)
VALID_SESSION=$(awk -F: -v token="$AUTH_TOKEN" -v time="$CURRENT_TIME" \
	'$1 == token && (time - $3) < 86400 {print $2}' "$SESSION_FILE")

if [ -z "$VALID_SESSION" ]; then
	echo "Status: 302"
	echo "Location: /"
	echo
	exit 0
fi

if echo "$ORIGINAL_URL" | grep -q "^/cgi-bin/panel\.cgi"; then
	exec /opt/panelbase/cgi-bin/panel.cgi
	exit 0
fi

REQUESTED_FILE="${DOCUMENT_ROOT}${ORIGINAL_URL}"

if [ ! -f "$REQUESTED_FILE" ]; then
	if [ -f "$DOCUMENT_ROOT/404.html" ]; then
		echo "Content-type: text/html"
		echo "Status: 404"
		echo "Cache-Control: no-store, no-cache, must-revalidate"
		echo "Pragma: no-cache"
		echo
		cat "$DOCUMENT_ROOT/404.html"
	else
		echo "Content-type: text/plain"
		echo "Status: 404"
		echo "Cache-Control: no-store, no-cache, must-revalidate"
		echo "Pragma: no-cache"
		echo
		echo "404 Not Found"
	fi
	exit 0
fi

EXTENSION="${REQUESTED_FILE##*.}"
case "$EXTENSION" in
	"html") CONTENT_TYPE="text/html" ;;
	"css") CONTENT_TYPE="text/css" ;;
	"js") CONTENT_TYPE="application/javascript" ;;
	"png") CONTENT_TYPE="image/png" ;;
	"jpg"|"jpeg") CONTENT_TYPE="image/jpeg" ;;
	"gif") CONTENT_TYPE="image/gif" ;;
	"svg") CONTENT_TYPE="image/svg+xml" ;;
	*) CONTENT_TYPE="application/octet-stream" ;;
esac

echo "Content-type: $CONTENT_TYPE"
echo "Cache-Control: no-store, no-cache, must-revalidate"
echo "Pragma: no-cache"
echo

cat "$REQUESTED_FILE"
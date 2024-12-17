#!/bin/bash

AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
ORIGINAL_URL="$REQUEST_URI"
DOCUMENT_ROOT="/opt/panelbase/www"

SESSION_ROTATION() {
	local token="$1"
	local session_file="$2"
	local current_time="$3"
	local rotation_interval=3600

	local session_time=$(awk -F: -v token="$token" '$1 == token {print $3}' "$session_file")
	if [ -n "$session_time" ] && [ $((current_time - session_time)) -gt $rotation_interval ]; then
		local new_token=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)
		local username=$(awk -F: -v token="$token" '$1 == token {print $2}' "$session_file")
		
		sed -i "/^$token:/d" "$session_file"
		echo "$new_token:$username:$current_time" >> "$session_file"
		
		echo "Set-Cookie: auth_token=$new_token; Path=/; HttpOnly; SameSite=Strict; Max-Age=86400"
	fi
}

SECURITY_HEADERS() {
	echo "Content-type: text/html"
	echo "X-Content-Type-Options: nosniff"
	echo "X-Frame-Options: SAMEORIGIN"
	echo "X-XSS-Protection: 1; mode=block"
	echo "Referrer-Policy: strict-origin-when-cross-origin"
	echo "Permissions-Policy: geolocation=(), microphone=(), camera=()"
	echo "Content-Security-Policy: default-src 'self' https://cdnjs.cloudflare.com; \
script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdnjs.cloudflare.com; \
style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; \
img-src 'self' data: https:; \
font-src 'self' https://cdnjs.cloudflare.com; \
frame-ancestors 'none'; \
form-action 'self'; \
base-uri 'self';"
	[ -n "$1" ] && echo "Status: $1"
	echo
}

if [ -z "$AUTH_TOKEN" ]; then
	SECURITY_HEADERS
	cat "$DOCUMENT_ROOT/index.html"
	exit 0
fi

SESSION_FILE="/opt/panelbase/config/sessions.conf"
if [ ! -f "$SESSION_FILE" ]; then
	SECURITY_HEADERS
	cat "$DOCUMENT_ROOT/index.html"
	exit 0
fi

CURRENT_TIME=$(date +%s)
VALID_SESSION=$(awk -F: -v token="$AUTH_TOKEN" -v time="$CURRENT_TIME" \
	'$1 == token && (time - $3) < 86400 {print $2}' "$SESSION_FILE")

if [ -z "$VALID_SESSION" ]; then
	SECURITY_HEADERS
	cat "$DOCUMENT_ROOT/index.html"
	exit 0
fi

SESSION_ROTATION "$AUTH_TOKEN" "$SESSION_FILE" "$CURRENT_TIME"

if echo "$ORIGINAL_URL" | grep -q "^/cgi-bin/panel\.cgi"; then
	exec /opt/panelbase/cgi-bin/panel.cgi
	exit 0
fi

REQUESTED_FILE="${DOCUMENT_ROOT}${ORIGINAL_URL}"

if [ ! -f "$REQUESTED_FILE" ]; then
	if [ -f "$DOCUMENT_ROOT/404.html" ]; then
		SECURITY_HEADERS "404"
		cat "$DOCUMENT_ROOT/404.html"
	else
		SECURITY_HEADERS "404"
		echo "<!DOCTYPE html>"
		echo "<html><head><title>404 Not Found</title></head>"
		echo "<body><h1>404 Not Found</h1>"
		echo "<p>The requested URL $ORIGINAL_URL was not found on this server.</p>"
		echo "</body></html>"
	fi
	exit 0
fi

EXTENSION="${REQUESTED_FILE##*.}"
case "$EXTENSION" in
	"html") 
		SECURITY_HEADERS
		;;
	"css") 
		echo "Content-type: text/css"
		echo "Cache-Control: public, max-age=31536000"
		echo
		;;
	"js") 
		echo "Content-type: application/javascript"
		echo "Cache-Control: public, max-age=31536000"
		echo
		;;
	"png"|"jpg"|"jpeg"|"gif") 
		echo "Content-type: image/${EXTENSION}"
		echo "Cache-Control: public, max-age=31536000"
		echo
		;;
	"svg") 
		echo "Content-type: image/svg+xml"
		echo "Cache-Control: public, max-age=31536000"
		echo
		;;
	"woff"|"woff2"|"ttf"|"eot")
		echo "Content-type: font/${EXTENSION}"
		echo "Cache-Control: public, max-age=31536000"
		echo
		;;
	*) 
		echo "Content-type: application/octet-stream"
		echo
		;;
esac

cat "$REQUESTED_FILE"
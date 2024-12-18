#!/bin/bash

if [ -f "/opt/panelbase/config/security.conf" ]; then
	source "/opt/panelbase/config/security.conf"
else
	echo "Content-type: application/json"
	echo "Status: 500"
	echo
	echo '{"status":"error","code":"config_not_found","message":"Security configuration file not found"}'
	exit 1
fi

log_auth_event() {
	local level="$1"
	local message="$2"
	echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $message" >> "$LOG_FILE"
}

WHITELIST_REGEX=$(echo "$WHITELIST_FILES" | sed 's/\./\\./g' | sed 's/\*/.*/g' | tr ' ' '|')
BLACKLIST_REGEX=$(echo "$BLACKLIST_FILES" | sed 's/\./\\./g' | sed 's/\*/.*/g' | tr ' ' '|')

check_file_access() {
	local file="$1"
	local referer="$2"
	local filename=$(basename "$file")
	local is_allowed=false

	case "$ACCESS_CONTROL_MODE" in
		"whitelist")
			if echo "$filename" | grep -qE "^($WHITELIST_REGEX)$"; then
				is_allowed=true
			elif [ "$ALLOW_HTML_REFERENCE" = "true" ] && [ -n "$referer" ] && echo "$referer" | grep -q "^/.*\.html"; then
				is_allowed=true
			fi
			;;
		"blacklist")
			if ! echo "$filename" | grep -qE "^($BLACKLIST_REGEX)$"; then
				is_allowed=true
			elif [ "$ALLOW_HTML_REFERENCE" = "true" ] && [ -n "$referer" ] && echo "$referer" | grep -q "^/.*\.html"; then
				is_allowed=true
			fi
			;;
		*)
			log_auth_event "ERROR" "Invalid ACCESS_CONTROL_MODE: $ACCESS_CONTROL_MODE"
			is_allowed=false
			;;
	esac

	[ "$is_allowed" = "true" ]
}

AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')
ORIGINAL_URL="$REQUEST_URI"
REFERER=$(echo "$HTTP_REFERER" | grep -oP 'http://[^/]+\K.*' || echo "")

SECURITY_HEADERS() {
	local content_type="${1:-text/html}"
	local status="$2"

	echo "Content-type: $content_type"
	echo "X-Content-Type-Options: nosniff"
	echo "X-Frame-Options: SAMEORIGIN"
	echo "X-XSS-Protection: 1; mode=block"
	echo "Referrer-Policy: strict-origin-when-cross-origin"
	echo "Permissions-Policy: geolocation=(), microphone=(), camera=()"
	echo "Content-Security-Policy: $SECURITY_HEADERS_CSP"
	[ -n "$status" ] && echo "Status: $status"
	echo
}

SHOW_ERROR() {
	local status="$1"
	local code="$2"
	local message="$3"

	log_auth_event "WARN" "$message"
	SECURITY_HEADERS "application/json" "$status"
	echo "{\"status\":\"error\",\"code\":\"$code\",\"message\":\"$message\"}"
	exit 0
}

SHOW_FORBIDDEN() {
	local message="$1"
	SHOW_ERROR "403" "forbidden" "$message"
}

SHOW_NOT_FOUND() {
	local message="$1"
	SHOW_ERROR "404" "not_found" "$message"
}

is_public_resource() {
	local url="$1"
	case "$url" in
		"/"|"/index.html"|"/auth.cgi"|"/cgi-bin/auth.cgi")
			return 0
			;;
		*)
			return 1
			;;
	esac
}

if is_public_resource "$ORIGINAL_URL"; then
	if [ "$ORIGINAL_URL" = "/" ] || [ "$ORIGINAL_URL" = "/index.html" ]; then
		SECURITY_HEADERS
		cat "$DOCUMENT_ROOT/index.html"
	else
		exec "$INSTALL_DIR$ORIGINAL_URL"
	fi
	exit 0
fi

SESSION_FILE="$INSTALL_DIR/config/sessions.conf"
if [ ! -f "$SESSION_FILE" ]; then
	log_auth_event "ERROR" "Session file not found"
	SHOW_ERROR "500" "session_file_missing" "Session file not found"
	exit 1
fi

if [ -z "$AUTH_TOKEN" ]; then
	log_auth_event "WARN" "No auth token provided"
	SHOW_ERROR "401" "authentication_required" "Authentication required"
	exit 0
fi

CURRENT_TIME=$(date +%s)
VALID_SESSION=$(awk -F: -v token="$AUTH_TOKEN" -v time="$CURRENT_TIME" -v max_age="$SESSION_LIFETIME" \
	'$1 == token && (time - $3) < max_age {print $2}' "$SESSION_FILE")

if [ -z "$VALID_SESSION" ]; then
	log_auth_event "WARN" "Invalid or expired session token: $AUTH_TOKEN"
	echo "Set-Cookie: auth_token=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT"
	SHOW_ERROR "401" "invalid_session" "Invalid or expired session"
	exit 0
fi

if [ $((CURRENT_TIME - $(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $3}' "$SESSION_FILE"))) -gt "$SESSION_ROTATION_INTERVAL" ]; then
	NEW_TOKEN=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)
	sed -i "/^$AUTH_TOKEN:/d" "$SESSION_FILE"
	echo "$NEW_TOKEN:$VALID_SESSION:$CURRENT_TIME" >> "$SESSION_FILE"
	chmod "$CONFIG_FILE_MODE" "$SESSION_FILE"

	log_auth_event "INFO" "Session rotated for user: $VALID_SESSION"

	SECURITY_HEADERS "application/json" "200"
	echo "Set-Cookie: auth_token=$NEW_TOKEN; Path=/; HttpOnly; SameSite=Strict; Max-Age=$SESSION_LIFETIME"
	echo "{\"status\":\"success\",\"code\":\"session_rotated\",\"message\":\"Session rotated successfully\"}"
	exit 0
fi

if [ "$ORIGINAL_URL" = "/panel.html" ]; then
	log_auth_event "INFO" "Access to panel.html: $VALID_SESSION"
	SECURITY_HEADERS
	cat "$DOCUMENT_ROOT/panel.html"
	exit 0
fi

if echo "$ORIGINAL_URL" | grep -q "^/cgi-bin/panel\.cgi"; then
	log_auth_event "INFO" "Access to panel.cgi: $VALID_SESSION"
	exec "$INSTALL_DIR/cgi-bin/panel.cgi"
	exit 0
fi

REQUESTED_FILE="${DOCUMENT_ROOT}${ORIGINAL_URL}"

if echo "$REQUESTED_FILE" | grep -q "\.\."; then
	SHOW_FORBIDDEN "Path traversal attempt detected"
fi

if ! check_file_access "$REQUESTED_FILE" "$REFERER"; then
	log_auth_event "WARN" "Access denied to file: $ORIGINAL_URL (Mode: $ACCESS_CONTROL_MODE, Referer: $REFERER)"
	SHOW_FORBIDDEN "Access to this resource is not allowed"
fi

if [ ! -f "$REQUESTED_FILE" ]; then
	log_auth_event "INFO" "404 Not Found: $REQUESTED_FILE"
	SHOW_NOT_FOUND "The requested URL $ORIGINAL_URL was not found on this server"
fi

EXTENSION="${REQUESTED_FILE##*.}"
case "$EXTENSION" in
	"html")
		SECURITY_HEADERS
		;;
	"css")
		SECURITY_HEADERS "text/css"
		echo "Cache-Control: public, max-age=$CACHE_MAX_AGE"
		;;
	"js")
		SECURITY_HEADERS "application/javascript"
		echo "Cache-Control: public, max-age=$CACHE_MAX_AGE"
		;;
	"png"|"jpg"|"jpeg"|"gif")
		SECURITY_HEADERS "image/${EXTENSION}"
		echo "Cache-Control: public, max-age=$CACHE_MAX_AGE"
		;;
	"svg")
		SECURITY_HEADERS "image/svg+xml"
		echo "Cache-Control: public, max-age=$CACHE_MAX_AGE"
		;;
	"woff"|"woff2"|"ttf"|"eot")
		SECURITY_HEADERS "font/${EXTENSION}"
		echo "Cache-Control: public, max-age=$CACHE_MAX_AGE"
		;;
	*)
		SECURITY_HEADERS "application/octet-stream"
		;;
esac

cat "$REQUESTED_FILE"
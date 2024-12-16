#!/bin/bash

AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')

ORIGINAL_URL="$REQUEST_URI"
DOCUMENT_ROOT="/opt/panelbase/www"

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
	echo "Content-type: text/html"
	echo "Status: 404"
	echo
	cat << EOF
<!DOCTYPE html>
<html lang="zh-TW">
<head>
	<meta charset="UTF-8">
	<title>404 - 頁面未找到</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			text-align: center;
			padding: 50px;
			background: #f5f5f5;
		}
		.error-container {
			background: white;
			padding: 30px;
			border-radius: 8px;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
			display: inline-block;
		}
		h1 { color: #e74c3c; }
		.back-link {
			margin-top: 20px;
			color: #3498db;
			text-decoration: none;
		}
		.back-link:hover {
			text-decoration: underline;
		}
	</style>
</head>
<body>
	<div class="error-container">
		<h1>404 - 頁面未找到</h1>
		<p>抱歉，您請求的頁面不存在。</p>
		<a href="/panel.html" class="back-link">返回主頁</a>
	</div>
</body>
</html>
EOF
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
echo

cat "$REQUESTED_FILE"
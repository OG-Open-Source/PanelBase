#!/bin/bash

read -n $CONTENT_LENGTH POST_DATA

if echo "$QUERY_STRING" | grep -q "action=logout"; then
	echo "Content-type: text/plain"
	echo "Set-Cookie: auth_token=; Path=/; HttpOnly; SameSite=Strict; Max-Age=0"
	echo "Status: 200"
	echo
	echo "OK"
	exit 0
fi

USERNAME=$(echo "$POST_DATA" | grep -oP 'username=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')
PASSWORD=$(echo "$POST_DATA" | grep -oP 'password=\K[^&]+' | sed 's/%40/@/g' | sed 's/%2B/+/g' | sed 's/%20/ /g')

CONFIG_FILE="/opt/panelbase/config/users.conf"
if [ ! -f "$CONFIG_FILE" ]; then
	echo "admin:$(echo -n "admin" | md5sum | cut -d' ' -f1)" > "$CONFIG_FILE"
	chmod 600 "$CONFIG_FILE"
fi

STORED_HASH=$(grep "^$USERNAME:" "$CONFIG_FILE" | cut -d: -f2)
INPUT_HASH=$(echo -n "$PASSWORD" | md5sum | cut -d' ' -f1)

if [ "$STORED_HASH" = "$INPUT_HASH" ]; then
	TOKEN=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)

	echo "$TOKEN:$USERNAME:$(date +%s)" >> "/opt/panelbase/config/sessions.conf"

	EXPIRY=$(($(date +%s) + 86400))
	echo "Content-type: text/plain"
	echo "Set-Cookie: auth_token=$TOKEN; Path=/; HttpOnly; SameSite=Strict; Max-Age=86400; Expires=$(date -u -d "@$EXPIRY" "+%a, %d %b %Y %H:%M:%S GMT")"
	echo "Status: 200"
	echo
	echo "OK"
else
	echo "Content-type: text/plain"
	echo "Status: 401"
	echo
	echo "Unauthorized"
fi
#!/bin/bash

source "/opt/panelbase/config/security.conf"

ROUTES_FILE="/opt/panelbase/config/routes.conf"
if [ ! -f "$ROUTES_FILE" ]; then
	echo "Content-type: application/json"
	echo "Status: 500"
	echo
	echo '{"status":"error","code":"500","message":"Routes configuration not found"}'
	exit 1
fi

REQUEST_PATH=$(echo "$REQUEST_URI" | cut -d'?' -f1 | sed 's/\/cgi-bin\/panel\.cgi//')
QUERY_STRING="${QUERY_STRING:-}"

while IFS=: read -r route command || [[ -n "$route" ]]; do
	[[ "$route" =~ ^[[:space:]]*# ]] && continue
	[ -z "$route" ] && continue

	route=$(echo "$route" | xargs)
	command=$(echo "$command" | xargs)

	if [ "$REQUEST_PATH" = "$route" ]; then
		output=$(eval "$command" 2>&1)
		exit_code=$?

		echo "Content-type: application/json"
		echo "Cache-Control: no-cache"
		[ $exit_code -ne 0 ] && echo "Status: 500"
		echo
		echo "{\"status\":\"$([ $exit_code -eq 0 ] && echo 'success' || echo 'error')\",\"code\":\"$exit_code\",\"message\":\"$output\"}"
		exit $exit_code
	fi
done < "$ROUTES_FILE"

echo "Content-type: application/json"
echo "Status: 404"
echo
echo '{"status":"error","code":"404","message":"API endpoint not found"}'
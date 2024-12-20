#!/bin/bash

source "/opt/panelbase/config/security.conf"

ROUTES_FILE="/opt/panelbase/config/routes.conf"
if [ ! -f "$ROUTES_FILE" ]; then
	echo "Content-type: application/json"
	echo "Status: 500"
	echo
	echo '{"status":"error","code":"routes_not_found","message":"Routes configuration not found"}'
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
		echo "Content-type: text/plain"
		echo "Cache-Control: no-cache"
		echo "X-Accel-Buffering: no"
		echo

		IFS=';' read -ra COMMANDS <<< "$command"
		for cmd in "${COMMANDS[@]}"; do
			cmd=$(echo "$cmd" | xargs)
			eval "$cmd" | while IFS= read -r line; do
				echo "$line"
			done
		done
		exit ${PIPESTATUS[0]}
	fi
done < "$ROUTES_FILE"

echo "Content-type: application/json"
echo "Status: 404"
echo
echo '{"status":"error","code":"route_not_found","message":"API endpoint not found"}'
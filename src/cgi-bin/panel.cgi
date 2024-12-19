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
REQUEST_METHOD="$REQUEST_METHOD"
QUERY_STRING="${QUERY_STRING:-}"
CONTENT_LENGTH="${CONTENT_LENGTH:-0}"

if [ "$REQUEST_METHOD" = "POST" ] && [ "$CONTENT_LENGTH" -gt 0 ]; then
	read -n "$CONTENT_LENGTH" POST_DATA
fi

parse_route_params() {
	local route="$1"
	local path="$2"
	local params=()

	IFS='/' read -ra ROUTE_PARTS <<< "$route"
	IFS='/' read -ra PATH_PARTS <<< "$path"

	for i in "${!ROUTE_PARTS[@]}"; do
		if [[ "${ROUTE_PARTS[$i]}" =~ ^\$[0-9]+$ ]]; then
			params+=("${PATH_PARTS[$i]}")
		fi
	done

	echo "${params[@]}"
}

parse_post_params() {
	local post_data="$1"
	local params=()

	IFS='&' read -ra PAIRS <<< "$post_data"
	for pair in "${PAIRS[@]}"; do
		IFS='=' read -r key value <<< "$pair"
		params+=("$value")
	done

	echo "${params[@]}"
}

match_route() {
	local route="$1"
	local path="$2"

	local route_regex=$(echo "$route" | sed 's/\$[0-9]\+/[^\/]\+/g')
	[[ "$path" =~ ^$route_regex$ ]]
}

while IFS= read -r line || [[ -n "$line" ]]; do
	[[ "$line" =~ ^[[:space:]]*# ]] && continue
	[ -z "$line" ] && continue

	read -r ROUTE_PATH ROUTE_METHOD ROUTE_COMMAND <<< "$line"

	if match_route "$ROUTE_PATH" "$REQUEST_PATH" && [ "$ROUTE_METHOD" = "$REQUEST_METHOD" ]; then
		ROUTE_PARAMS=($(parse_route_params "$ROUTE_PATH" "$REQUEST_PATH"))

		if [ "$REQUEST_METHOD" = "POST" ]; then
			POST_PARAMS=($(parse_post_params "$POST_DATA"))
		fi

		COMMAND="$ROUTE_COMMAND"
		for i in "${!ROUTE_PARAMS[@]}"; do
			COMMAND=${COMMAND//\$$((i+1))/${ROUTE_PARAMS[$i]}}
		done
		for i in "${!POST_PARAMS[@]}"; do
			COMMAND=${COMMAND//\%$((i+1))/${POST_PARAMS[$i]}}
		done

		eval "$COMMAND"
		exit 0
	fi
done < "$ROUTES_FILE"

echo "Content-type: application/json"
echo "Status: 404"
echo
echo '{"status":"error","code":"route_not_found","message":"API endpoint not found"}'
#!/bin/bash

source "/opt/panelbase/config/security.conf"

ROUTES_FILE="/opt/panelbase/config/routes.conf"
TEMP_DIR="/opt/panelbase/temp"
ACCESS_LOG="/opt/panelbase/logs/access.log"

mkdir -p "$TEMP_DIR"
mkdir -p "$(dirname "$ACCESS_LOG")"
touch "$ACCESS_LOG"

if [ ! -f "$ROUTES_FILE" ]; then
	echo "Content-type: application/json"
	echo "Status: 500"
	echo
	echo '{"status":"error","code":"500","message":"Routes configuration not found"}'
	exit 1
fi

REQUEST_PATH=$(echo "$REQUEST_URI" | cut -d'?' -f1 | sed 's/\/cgi-bin\/panel\.cgi//')
QUERY_STRING="${QUERY_STRING:-}"

log_command() {
	echo "[$(date '+%Y-%m-%d %H:%M:%S')] $2" >> "$ACCESS_LOG"
}

execute_command() {
	local commands="$1"
	local timestamp=$(date '+%Y%m%d%H%M%S')
	local temp_file="$TEMP_DIR/${timestamp}.log"
	local total_commands=0
	local current_command=0

	IFS=';' read -ra CMD_ARRAY <<< "$commands"
	total_commands=${#CMD_ARRAY[@]}

	local existing_log=$(find "$TEMP_DIR" -name "*.log" -type f -exec grep -l "^1|$commands" {} \; | sort | head -n1)
	
	if [ -n "$existing_log" ]; then
		temp_file="$existing_log"
	else
		for ((i=0; i<total_commands; i++)); do
			echo "$((i+1))|${CMD_ARRAY[i]}" >> "$temp_file"
		done
	fi

	local next_cmd=$(grep -v "\[Done\]\|\[Failed\]" "$temp_file" | head -n1)
	if [ -z "$next_cmd" ]; then
		echo "Content-type: application/json"
		echo "Cache-Control: no-cache"
		echo
		echo '{"status":"success","code":"0","message":"All commands completed"}'
		return 0
	fi

	current_command=$(echo "$next_cmd" | cut -d'|' -f1)
	local cmd=$(echo "$next_cmd" | cut -d'|' -f2)

	log_command "$timestamp" "(${current_command}/${total_commands}) Executing: $cmd"
	output=$(bash -c "$cmd" 2>&1)
	exit_code=$?

	if [ $exit_code -eq 0 ]; then
		sed -i "${current_command}s/^${current_command}|/&[Done] /" "$temp_file"
		log_command "$timestamp" "(${current_command}/${total_commands}) Completed successfully"
		echo "Content-type: application/json"
		echo "Cache-Control: no-cache"
		echo
		if [ $current_command -eq $total_commands ]; then
			echo '{"status":"success","code":"0","message":"All commands completed"}'
		else
			echo "{\"status\":\"success\",\"code\":\"0\",\"message\":\"Command executed to ${current_command}/${total_commands}\"}"
		fi
	else
		sed -i "${current_command}s/^${current_command}|/&[Failed] /" "$temp_file"
		log_command "$timestamp" "(${current_command}/${total_commands}) Failed with code $exit_code"
		echo "Content-type: application/json"
		echo "Status: 500"
		echo
		echo "{\"status\":\"error\",\"code\":\"$exit_code\",\"message\":\"Command failed at ${current_command}/${total_commands}: $output\"}"
	fi

	return $exit_code
}

while IFS=: read -r route command || [[ -n "$route" ]]; do
	[[ "$route" =~ ^[[:space:]]*# ]] && continue
	[ -z "$route" ] && continue

	route=$(echo "$route" | xargs)
	command=$(echo "$command" | xargs)

	if [ "$REQUEST_PATH" = "$route" ]; then
		execute_command "$command"
		exit $?
	fi
done < "$ROUTES_FILE"

echo "Content-type: application/json"
echo "Status: 404"
echo
echo '{"status":"error","code":"404","message":"API endpoint not found"}'
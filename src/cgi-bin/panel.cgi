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

execute_commands() {
	local commands="$1"
	local timestamp=$(date '+%Y%m%d%H%M%S')
	local temp_file="$TEMP_DIR/${timestamp}.log"
	local total_commands=0
	local current_command=0
	local all_output=""
	local final_exit_code=0

	IFS=';' read -ra CMD_ARRAY <<< "$commands"
	total_commands=${#CMD_ARRAY[@]}

	for ((i=0; i<total_commands; i++)); do
		echo "$((i+1))|${CMD_ARRAY[i]}" >> "$temp_file"
	done

	for ((i=0; i<total_commands; i++)); do
		current_command=$((i+1))
		cmd="${CMD_ARRAY[i]}"
		
		log_command "$timestamp" "(${current_command}/${total_commands}) Executing: $cmd"
		
		output=$(eval "$cmd" 2>&1)
		exit_code=$?
		
		all_output="${all_output}${output}\n"
		
		if [ $exit_code -eq 0 ]; then
			sed -i "${current_command}s/^${current_command}|/&[Done] /" "$temp_file"
			log_command "$timestamp" "(${current_command}/${total_commands}) Completed successfully"
		else
			sed -i "${current_command}s/^${current_command}|/&[Failed] /" "$temp_file"
			log_command "$timestamp" "(${current_command}/${total_commands}) Failed with code $exit_code"
			final_exit_code=$exit_code
			break
		fi
	done

	if [ $final_exit_code -eq 0 ]; then
		if [ $current_command -eq $total_commands ]; then
			echo "{\"status\":\"success\",\"code\":\"0\",\"message\":\"Command executed successfully\"}"
		else
			echo "{\"status\":\"success\",\"code\":\"0\",\"message\":\"Command executed to ${current_command}/${total_commands}\"}"
		fi
	else
		echo "{\"status\":\"error\",\"code\":\"$final_exit_code\",\"message\":\"Command failed at ${current_command}/${total_commands}: ${output}\"}"
	fi

	return $final_exit_code
}

while IFS=: read -r route command || [[ -n "$route" ]]; do
	[[ "$route" =~ ^[[:space:]]*# ]] && continue
	[ -z "$route" ] && continue

	route=$(echo "$route" | xargs)
	command=$(echo "$command" | xargs)

	if [ "$REQUEST_PATH" = "$route" ]; then
		echo "Content-type: application/json"
		echo "Cache-Control: no-cache"
		echo

		result=$(execute_commands "$command")
		exit_code=$?
		
		[ $exit_code -ne 0 ] && echo "Status: 500"
		echo "$result"
		exit $exit_code
	fi
done < "$ROUTES_FILE"

echo "Content-type: application/json"
echo "Status: 404"
echo
echo '{"status":"error","code":"404","message":"API endpoint not found"}'
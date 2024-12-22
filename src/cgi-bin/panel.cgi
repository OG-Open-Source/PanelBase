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
ACCEPT_HEADER="${HTTP_ACCEPT:-application/json}"

log_command() {
	echo "[$(date '+%Y-%m-%d %H:%M:%S')] $2" >> "$ACCESS_LOG"
}

output_result() {
	local status="$1"
	local code="$2"
	local message="$3"
	local output="$4"
	local current="$5"
	local total="$6"
	local cmd="$7"

	if [[ "$ACCEPT_HEADER" == *"text/plain"* ]]; then
		echo "Content-type: text/plain"
		echo "Cache-Control: no-cache"
		[ "$status" = "error" ] && echo "Status: $code"
		echo
		if [ "$status" = "error" ]; then
			echo "Execution failed (${current}/${total}): $cmd"
			echo "Error code: $code"
			echo "Error message:"
			echo "$output"
		else
			[ -n "$cmd" ] && echo "Executing command (${current}/${total}): $cmd"
			[ -n "$output" ] && echo "$output"
			[ -n "$cmd" ] && echo "----------------------------------------"
			fi
	else
		echo "Content-type: application/json"
		echo "Cache-Control: no-cache"
		[ "$status" = "error" ] && echo "Status: $code"
		echo
		if [ -n "$output" ]; then
			echo "{\"status\":\"$status\",\"code\":\"$code\",\"message\":\"$message\",\"output\":\"$(echo "$output" | sed 's/"/\\"/g' | tr '\n' ' ')\"}"
		else
			echo "{\"status\":\"$status\",\"code\":\"$code\",\"message\":\"$message\"}"
		fi
	fi
}

execute_command() {
	local commands="$1"
	local date_prefix=$(date '+%Y-%m-%d')
	local api_path=$(echo "$REQUEST_PATH" | sed 's/\//_/g')
	local temp_file="$TEMP_DIR/${date_prefix}${api_path}.log"
	local total_commands=0
	local current_command=0

	IFS=';' read -ra CMD_ARRAY <<< "$commands"
	total_commands=${#CMD_ARRAY[@]}

	if [ ! -f "$temp_file" ]; then
		echo "[$(date '+%Y-%m-%d %H:%M:%S')] Command execution started" > "$temp_file"
		printf "2%.0s" $(seq 1 $total_commands) >> "$temp_file"
		echo "" >> "$temp_file"
		for ((i=0; i<total_commands; i++)); do
			echo "$((i+1))|${CMD_ARRAY[i]}" >> "$temp_file"
		done
	fi

	local status_line=$(sed -n '2p' "$temp_file")
	
	current_command=0
	for ((i=0; i<${#status_line}; i++)); do
		if [ "${status_line:$i:1}" = "2" ]; then
			current_command=$((i + 1))
			break
		fi
	done

	if [ $current_command -eq 0 ]; then
		if ! echo "$status_line" | grep -q "[^0]"; then
			echo "[$(date '+%Y-%m-%d %H:%M:%S')] All commands completed successfully" >> "$temp_file"
			cat "$temp_file"
			rm -f "$temp_file"
			for ((i=0; i<total_commands; i++)); do
				echo "$((i+1))|${CMD_ARRAY[i]}" >> "$temp_file"
			done
			return 0
		fi
		if echo "$status_line" | grep -q "1"; then
			rm -f "$temp_file"
			execute_command "$commands"
			return $?
		fi
		output_result "success" "0" "All commands completed" "" "" "" ""
		return 0
	fi

	local cmd=$(sed -n "$((current_command + 2))p" "$temp_file" | cut -d'|' -f2)

	log_command "$date_prefix" "(${current_command}/${total_commands}) Executing: $cmd"
	output=$(bash -c "$cmd" 2>&1)
	exit_code=$?

	if [ $exit_code -eq 0 ]; then
		sed -i "2s/./0/$current_command" "$temp_file"
		log_command "$date_prefix" "(${current_command}/${total_commands}) Completed successfully"
		output_result "success" "0" "Command executed to ${current_command}/${total_commands}" "$output" "$current_command" "$total_commands" "$cmd"
	else
		sed -i "2s/./1/$current_command" "$temp_file"
		log_command "$date_prefix" "(${current_command}/${total_commands}) Failed with code $exit_code"
		output_result "error" "$exit_code" "Command failed at ${current_command}/${total_commands}" "$output" "$current_command" "$total_commands" "$cmd"
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

output_result "error" "404" "API endpoint not found" "" "" "" ""
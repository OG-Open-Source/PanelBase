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
		for ((i=0; i<total_commands; i++)); do
			echo "$((i+1))|${CMD_ARRAY[i]}" >> "$temp_file"
		done
	fi

	local next_cmd=$(grep -v "\[Done\]\|\[Failed\]" "$temp_file" | head -n1)
	if [ -z "$next_cmd" ]; then
		if [ $(grep -c "\[Done\]" "$temp_file") -eq $total_commands ]; then
			rm -f "$temp_file"
			for ((i=0; i<total_commands; i++)); do
				echo "$((i+1))|${CMD_ARRAY[i]}" >> "$temp_file"
			done
			execute_command "$commands"
			return $?
		fi
		output_result "success" "0" "All commands completed" "" "" "" ""
		return 0
	fi

	current_command=$(echo "$next_cmd" | cut -d'|' -f1)
	local cmd=$(echo "$next_cmd" | cut -d'|' -f2)

	log_command "$date_prefix" "(${current_command}/${total_commands}) Executing: $cmd"
	output=$(bash -c "$cmd" 2>&1)
	exit_code=$?

	if [ $exit_code -eq 0 ]; then
		sed -i "${current_command}s/^${current_command}|/&[Done] /" "$temp_file"
		log_command "$date_prefix" "(${current_command}/${total_commands}) Completed successfully"
		
		if [ $current_command -eq $total_commands ]; then
			if [ $(grep -c "\[Done\]" "$temp_file") -eq $total_commands ]; then
				output_result "success" "0" "All commands completed" "$output" "$current_command" "$total_commands" "$cmd"
				sed -i "${current_command}s/^${current_command}|/&[Done] /" "$temp_file"
				rm -f "$temp_file"
				for ((i=0; i<total_commands; i++)); do
					echo "$((i+1))|${CMD_ARRAY[i]}" >> "$temp_file"
				done
				execute_command "$commands"
				return $?
			fi
		else
			output_result "success" "0" "Command executed to ${current_command}/${total_commands}" "$output" "$current_command" "$total_commands" "$cmd"
		fi
	else
		sed -i "${current_command}s/^${current_command}|/&[Failed] /" "$temp_file"
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
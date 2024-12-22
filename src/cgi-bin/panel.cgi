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
	local duration="$8"
	local start_time="$9"
	local end_time=$(date +%s)
	local elapsed=$((end_time - start_time))
	local hours=$((elapsed / 3600))
	local minutes=$(( (elapsed % 3600) / 60 ))
	local seconds=$((elapsed % 60))
	local elapsed_time=$(printf "%02d:%02d:%02d" $hours $minutes $seconds)

	if [[ "$ACCEPT_HEADER" == *"text/plain"* ]]; then
		if [ -n "$cmd" ]; then
			echo "Content-type: text/plain"
			echo "Cache-Control: no-cache"
			echo
			echo "[${elapsed_time}] Executing command (${current}/${total}): $cmd"
			[ -n "$output" ] && echo "$output"
		elif [ "$status" = "error" ]; then
			echo "Content-type: text/plain"
			echo "Cache-Control: no-cache"
			echo "Status: $code"
			echo
			echo "[${elapsed_time}] Execution failed (${current}/${total}): $cmd"
			echo "Error code: $code"
			echo "Error message:"
			echo "$output"
		else
			echo "Content-type: text/plain"
			echo "Cache-Control: no-cache"
			echo
			echo "[${elapsed_time}] ${message}"
		fi
	else
		echo "Content-type: application/json"
		echo "Cache-Control: no-cache"
		[ "$status" = "error" ] && echo "Status: $code"
		echo
		if [ -n "$output" ]; then
			message="[${elapsed_time}] ${message}\n${output}"
		else
			message="[${elapsed_time}] ${message}"
		fi
		echo "{\"status\":\"$status\",\"code\":\"$code\",\"message\":\"$(echo "$message" | sed 's/"/\\"/g')\"}"
	fi
}

check_and_reset() {
	local temp_file="$1"
	local start_time="$2"
	local status_line=$(head -n1 "$temp_file")

	local end_time=$(date +%s)
	local duration=$((end_time - start_time))

	if ! echo "$status_line" | grep -q "[^0]"; then
		output_result "success" "0" "All commands completed" "" "" "" "" "$duration" "$start_time"
		rm -f "$temp_file"
		return 2
	fi

	if echo "$status_line" | grep -q "1"; then
		rm -f "$temp_file"
		return 1
	fi

	return 0
}

execute_command() {
	local commands="$1"
	local date_prefix=$(date '+%Y-%m-%d')
	local api_path=$(echo "$REQUEST_PATH" | sed 's/\//_/g')
	local temp_file="$TEMP_DIR/${date_prefix}${api_path}.log"
	local total_commands=0
	local current_command=0
	local start_time=$(date +%s)

	IFS=';' read -ra CMD_ARRAY <<< "$commands"
	total_commands=${#CMD_ARRAY[@]}

	if [ ! -f "$temp_file" ]; then
		printf "2%.0s" $(seq 1 $total_commands) > "$temp_file"
		echo "" >> "$temp_file"
		for ((i=0; i<total_commands; i++)); do
			printf "%d|%s\n" "$((i+1))" "${CMD_ARRAY[i]}" >> "$temp_file"
		done
	fi

	local status_line=$(head -n1 "$temp_file")
	
	current_command=0
	for ((i=0; i<${#status_line}; i++)); do
		if [ "${status_line:$i:1}" = "2" ]; then
			current_command=$((i + 1))
			break
		fi
	done

	if [ $current_command -eq 0 ]; then
		check_and_reset "$temp_file" "$start_time"
		return $?
	fi

	local cmd
	cmd=$(sed -n "$((current_command + 1))p" "$temp_file" | cut -d'|' -f2-)

	log_command "$date_prefix" "(${current_command}/${total_commands}) Executing: $cmd"
	output=$(eval "$cmd" 2>&1)
	exit_code=$?
	local end_time=$(date +%s)
	local duration=$((end_time - start_time))

	if [ $exit_code -eq 0 ]; then
		sed -i "1s/./0/$current_command" "$temp_file"
		log_command "$date_prefix" "(${current_command}/${total_commands}) Completed successfully"
		output_result "success" "0" "Command executed to ${current_command}/${total_commands}" "$output" "$current_command" "$total_commands" "$cmd" "$duration" "$start_time"
		return 0
	else
		sed -i "1s/./1/$current_command" "$temp_file"
		log_command "$date_prefix" "(${current_command}/${total_commands}) Failed with code $exit_code"
		output_result "error" "$exit_code" "Command failed at ${current_command}/${total_commands}" "$output" "$current_command" "$total_commands" "$cmd" "$duration" "$start_time"
		
		if [ $current_command -eq $total_commands ]; then
			return 1
		fi
		return 0
	fi
}

while IFS=: read -r route command || [[ -n "$route" ]]; do
	[[ "$route" =~ ^[[:space:]]*# ]] && continue
	[ -z "$route" ] && continue

	route=$(echo "$route" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
	command=$(echo "$command" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')

	if [ "$REQUEST_PATH" = "$route" ]; then
		execute_command "$command"
		result=$?
		case $result in
			0) ;;
			1|2) execute_command "$command" ;;
		esac
		exit 0
	fi
done < "$ROUTES_FILE"

output_result "error" "404" "API endpoint not found" "" "" "" ""
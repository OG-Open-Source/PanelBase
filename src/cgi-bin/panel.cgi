#!/bin/bash

source "/opt/panelbase/config/security.conf"

ROUTES_FILE="/opt/panelbase/config/routes.conf"
TEMP_DIR="/opt/panelbase/temp"
ACCESS_LOG="/opt/panelbase/logs/access.log"
REQUEST_LOG="/opt/panelbase/logs/request.log"

mkdir -p "$TEMP_DIR" "$(dirname "$ACCESS_LOG")"
touch "$ACCESS_LOG" "$REQUEST_LOG"

[ ! -f "$ROUTES_FILE" ] && {
	echo -e "Content-type: application/json\nStatus: 500\n\n{\"status\":\"error\",\"code\":\"500\",\"message\":\"Routes configuration not found\"}"
	exit 1
}

REQUEST_PATH=$(echo "$REQUEST_URI" | cut -d'?' -f1 | sed 's/\/cgi-bin\/panel\.cgi//')
QUERY_STRING="${QUERY_STRING:-}"
ACCEPT_HEADER="${HTTP_ACCEPT:-application/json}"
SHOULD_LOG=1
MAX_LOG_SIZE=10485760
MAX_REQUESTS=1000
REQUEST_WINDOW=60

check_request_limit() {
	local now=$(date +%s)
	local window_start=$((now - REQUEST_WINDOW))
	
	sed -i "/^[0-9]\{10\} /!d" "$REQUEST_LOG"
	sed -i "/^[$window_start-$now]/!d" "$REQUEST_LOG"
	
	echo "$now $REQUEST_PATH" >> "$REQUEST_LOG"
	
	local count=$(wc -l < "$REQUEST_LOG")
	[ "$count" -gt "$MAX_REQUESTS" ] && {
		echo -e "Content-type: application/json\nStatus: 429\n\n{\"status\":\"error\",\"code\":\"429\",\"message\":\"Too many requests\"}"
		exit 1
	}
}

check_log_size() {
	local size=$(stat -f%z "$ACCESS_LOG" 2>/dev/null || stat -c%s "$ACCESS_LOG")
	[ "$size" -gt "$MAX_LOG_SIZE" ] && {
		local timestamp=$(date '+%Y%m%d_%H%M%S')
		mv "$ACCESS_LOG" "${ACCESS_LOG}.${timestamp}"
		touch "$ACCESS_LOG"
	}
}

log_command() {
	[ "$SHOULD_LOG" = "1" ] && {
		check_log_size
		echo "[${1}] ${2}" >> "$ACCESS_LOG"
	}
}

output_result() {
	local status="$1" code="$2" message="$3" output="$4" current="$5" total="$6" cmd="$7" duration="$8" start_time="$9"
	local end_time=$(date +%s) elapsed=$((end_time - start_time))
	local hours=$((elapsed / 3600)) minutes=$(( (elapsed % 3600) / 60 )) seconds=$((elapsed % 60))
	local elapsed_time=$(printf "%02d:%02d:%02d" $hours $minutes $seconds)

	if [[ "$ACCEPT_HEADER" == *"text/plain"* ]]; then
		echo -e "Content-type: text/plain\nCache-Control: no-cache\n"
		if [ -n "$cmd" ]; then
			echo "[${elapsed_time}] Executing command (${current}/${total}): $cmd"
			[ -n "$output" ] && echo "$output"
		elif [ "$status" = "error" ]; then
			echo -e "Status: $code\n\n[${elapsed_time}] Execution failed (${current}/${total}): $cmd\nError code: $code\nError message:\n$output"
		else
			echo "[${elapsed_time}] ${message}"
		fi
	else
		echo -e "Content-type: application/json\nCache-Control: no-cache"
		[ "$status" = "error" ] && echo "Status: $code"
		echo
		if [ -n "$output" ]; then
			message="[${elapsed_time}] ${message}"
			echo "{\"status\":\"$status\",\"code\":\"$code\",\"message\":\"$(echo -e "$message" | sed 's/"/\\"/g')\",\"output\":\"$(echo -e "$output" | sed 's/"/\\"/g')\"}"
		else
			message="[${elapsed_time}] ${message}"
			echo "{\"status\":\"$status\",\"code\":\"$code\",\"message\":\"$(echo -e "$message" | sed 's/"/\\"/g')\"}"
		fi
	fi
}

check_and_reset() {
	local temp_file="$1" start_time="$2" status_line=$(head -n1 "$temp_file")
	local end_time=$(date +%s) duration=$((end_time - start_time))

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
	local commands="$1" date_prefix=$(date '+%Y-%m-%d')
	commands=$(echo "$commands" | awk '
		BEGIN { RS=""; FS="" }
		{
			in_quote=0
			in_bracket=0
			result=""
			for(i=1;i<=length($0);i++) {
				c=substr($0,i,1)
				if(c=="\"" && substr($0,i-1,1)!="\\") {
					in_quote=!in_quote
				}
				if(c=="[" && !in_quote) in_bracket++
				if(c=="]" && !in_quote) in_bracket--
				if(c==";" && !in_quote && !in_bracket) {
					result=result ";" substr($0,i+1)
					i++
					while(substr($0,i,1) ~ /[[:space:]]/) i++
					i--
				} else {
					result=result c
				}
			}
			print result
		}
	')
	local api_path=$(echo "$REQUEST_PATH" | sed 's/\//_/g')
	local temp_file="$TEMP_DIR/${date_prefix}${api_path}.log"
	local total_commands=0 current_command=0 start_time=$(date +%s)

	readarray -t CMD_ARRAY < <(echo "$commands" | awk '
		BEGIN { RS=";" }
		NF { gsub(/^[[:space:]]+|[[:space:]]+$/, ""); print }
	')
	total_commands=${#CMD_ARRAY[@]}

	if [ ! -f "$temp_file" ] || grep -q "1" "$temp_file"; then
		printf "2%.0s" $(seq 1 $total_commands) > "$temp_file"
		echo "" >> "$temp_file"
		for ((i=0; i<total_commands; i++)); do
			local cmd="${CMD_ARRAY[i]}"
			printf "%d|%s\n" "$((i+1))" "$cmd" >> "$temp_file"
		done
	fi

	local status_line=$(head -n1 "$temp_file")
	for ((i=0; i<${#status_line}; i++)); do
		[ "${status_line:$i:1}" = "2" ] && { current_command=$((i + 1)); break; }
	done

	[ $current_command -eq 0 ] && { check_and_reset "$temp_file" "$start_time"; return $?; }

	local cmd=$(sed -n "$((current_command + 1))p" "$temp_file" | cut -d'|' -f2-)
	log_command "$date_prefix" "(${current_command}/${total_commands}) Executing: $cmd"

	output=$(eval "$cmd" 2>&1)
	exit_code=$?
	local end_time=$(date +%s) duration=$((end_time - start_time))

	if [ $exit_code -eq 0 ]; then
		sed -i "1s/./0/$current_command" "$temp_file"
		log_command "$date_prefix" "(${current_command}/${total_commands}) Completed successfully"
		output_result "success" "0" "Command executed to ${current_command}/${total_commands}" "$output" "$current_command" "$total_commands" "$cmd" "$duration" "$start_time"

		if ! grep -q "2" "$temp_file"; then
			output_result "success" "0" "All commands completed" "" "" "" "" "$duration" "$start_time"
			rm -f "$temp_file"
			return 2
		fi
		return 0
	else
		sed -i "1s/./1/$current_command" "$temp_file"
		log_command "$date_prefix" "(${current_command}/${total_commands}) Failed with code $exit_code"
		output_result "error" "$exit_code" "Command failed at ${current_command}/${total_commands}" "$output" "$current_command" "$total_commands" "$cmd" "$duration" "$start_time"
		return 1
	fi
}

check_request_limit

while IFS=: read -r route command || [[ -n "$route" ]]; do
	[[ "$route" =~ ^[[:space:]]*# ]] && continue
	[ -z "$route" ] && continue

	route=$(echo "$route" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
	command=$(echo "$command" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')

	if [[ "$route" =~ ^([^@]+)@([^:]+)$ ]]; then
		route="${BASH_REMATCH[1]}"
		local flags=(${BASH_REMATCH[2]/,/ })
		for flag in "${flags[@]}"; do
			case "$flag" in
				nolog) SHOULD_LOG=0 ;;
				maxlog=*) MAX_LOG_SIZE=$((${flag#maxlog=} * 1024 * 1024)) ;;
				maxreq=*) MAX_REQUESTS=${flag#maxreq=} ;;
				window=*) REQUEST_WINDOW=${flag#window=} ;;
			esac
		done
	fi

	[ "$REQUEST_PATH" = "$route" ] && {
		execute_command "$command"
		result=$?
		case $result in
			0) ;;
			1|2) execute_command "$command" ;;
		esac
		exit 0
	}
done < "$ROUTES_FILE"

output_result "error" "404" "API endpoint not found" "" "" "" ""
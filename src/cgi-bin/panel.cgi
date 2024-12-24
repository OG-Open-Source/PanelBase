#!/bin/bash

INSTALL_DIR="/opt/panelbase"
ROUTES_CONF="$INSTALL_DIR/config/routes.conf"

format_time() { date -u "+%Y-%m-%dT%H:%M:%SZ"; }

calculate_elapsed() {
	local start=$1
	local end=$2
	echo "$((end - start))s"
}

send_response() {
	local status="$1"
	local data="$2"
	echo "Content-type: application/json"
	echo
	echo "{\"status\":\"$status\",\"data\":$data}"
}

execute_command() {
	local command="$1"
	local start_time=$(date +%s)
	local start_time_iso=$(format_time)
	local steps=()
	local errors=()
	local current=0
	local total=0
	IFS=';' read -ra COMMANDS <<< "$command"
	total=${#COMMANDS[@]}
	for cmd in "${COMMANDS[@]}"; do
		cmd=$(echo "$cmd" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
		((current++))
		local step_start=$(date +%s)
		local output
		local exit_code
		output=$(eval "$cmd" 2>&1)
		exit_code=$?
		local step_end=$(date +%s)
		local step_elapsed=$(calculate_elapsed $step_start $step_end)
		if [ $exit_code -eq 0 ]; then
			steps+=("{\"command\":\"$cmd\",\"output\":\"$output\",\"status\":\"success\",\"elapsed_time\":\"$step_elapsed\",\"step\":\"$current\",\"total\":\"$total\"}")
			errors+=("\"\"")
		else
			steps+=("{\"command\":\"$cmd\",\"output\":\"$output\",\"status\":\"error\",\"elapsed_time\":\"$step_elapsed\",\"step\":\"$current\",\"total\":\"$total\"}")
			errors+=("\"$output\"")
			break
		fi
	done
	local end_time=$(date +%s)
	local end_time_iso=$(format_time)
	local elapsed_time=$(calculate_elapsed $start_time $end_time)
	local percentage=$((current * 100 / total))
	local steps_json=$(IFS=,; echo "${steps[*]}")
	local errors_json=$(IFS=,; echo "${errors[*]}")
	local data="{\"command\":\"$command\",\"start_time\":\"$start_time_iso\",\"end_time\":\"$end_time_iso\",\"elapsed_time\":\"$elapsed_time\",\"progress\":{\"current\":$current,\"total\":$total,\"percentage\":$percentage},\"steps\":[$steps_json],\"errors\":[$errors_json]}"
	if [ $current -eq $total ]; then
		send_response "success" "$data"
	else
		send_response "error" "$data"
	fi
}

main() {
	[ -z "$REQUEST_PATH" ] && { send_response "error" "{\"error\":\"No request path provided\"}"; exit 1; }
	[ ! -f "$ROUTES_CONF" ] && { send_response "error" "{\"error\":\"Routes configuration not found\"}"; exit 1; }
	command=$(grep "^$REQUEST_PATH:" "$ROUTES_CONF" | cut -d':' -f2-)
	[ -z "$command" ] && { send_response "error" "{\"error\":\"Route not found\"}"; exit 1; }
	execute_command "$command"
}

main
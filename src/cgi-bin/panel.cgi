#!/bin/bash

INSTALL_DIR="/opt/panelbase"
ROUTES_CONF="$INSTALL_DIR/config/routes.conf"
REQUEST_PATH="${PATH_INFO}"

decode_url() {
	local encoded="$1"
	echo "$encoded" | sed -e 's/+/ /g' \
		-e 's/%20/ /g' \
		-e 's/%21/!/g' \
		-e 's/%22/"/g' \
		-e 's/%23/#/g' \
		-e 's/%24/$/g' \
		-e 's/%25/%/g' \
		-e 's/%26/\&/g' \
		-e 's/%27/'\''/g' \
		-e 's/%28/(/g' \
		-e 's/%29/)/g' \
		-e 's/%2A/*/g' \
		-e 's/%2B/+/g' \
		-e 's/%2C/,/g' \
		-e 's/%2D/-/g' \
		-e 's/%2E/./g' \
		-e 's/%2F/\//g' \
		-e 's/%3A/:/g' \
		-e 's/%3B/;/g' \
		-e 's/%3C/</g' \
		-e 's/%3D/=/g' \
		-e 's/%3E/>/g' \
		-e 's/%3F/?/g' \
		-e 's/%40/@/g' \
		-e 's/%5B/[/g' \
		-e 's/%5C/\\/g' \
		-e 's/%5D/]/g' \
		-e 's/%5E/^/g' \
		-e 's/%5F/_/g' \
		-e 's/%60/`/g' \
		-e 's/%7B/{/g' \
		-e 's/%7C/|/g' \
		-e 's/%7D/}/g' \
		-e 's/%7E/~/g'
}

get_query_param() {
	local param_name="$1"
	local is_base64="$2"
	local query_string="$QUERY_STRING"
	local param_value

	while IFS='=' read -d '&' key value || [ -n "$key" ]; do
		if [ "$key" = "$param_name" ]; then
			param_value="$value"
			break
		fi
	done < <(echo -n "$query_string")

	if [ -n "$param_value" ]; then
		param_value=$(decode_url "$param_value")
		[ "$is_base64" = "true" ] && param_value=$(echo "$param_value" | base64 -d 2>/dev/null || echo "$param_value")
		echo "$param_value"
	else
		echo ""
	fi
}

format_time() { date -u "+%Y-%m-%dT%H:%M:%SZ"; }

calculate_elapsed() {
	local start="$1"
	local end="$2"
	echo "$((end - start))s"
}

escape_json() {
	local text="$1"
	text=$(echo "$text" | sed 's/\x1b\[[0-9;]*[mGKHF]//g')
	text="${text//\\/\\\\}"
	text="${text//\"/\\\"}"
	text="${text//$'\b'/\\b}"
	text="${text//$'\f'/\\f}"
	text="${text//$'\n'/\\n}"
	text="${text//$'\r'/\\r}"
	text="${text//$'\t'/\\t}"
	echo "$text"
}

send_error_response() {
	local error_msg="$1"
	local current_time=$(format_time)
	local error_json="{\"status\":\"error\",\"data\":{\
\"command\":\"\",\
\"start_time\":\"$current_time\",\
\"end_time\":\"$current_time\",\
\"elapsed_time\":\"0s\",\
\"progress\":{\"current\":0,\"total\":0,\"percentage\":0},\
\"steps\":[],\
\"errors\":[\"$(escape_json "$error_msg")\"]\
}}"

	echo "Content-type: application/json"
	echo
	echo "$error_json"
}

send_response() {
	local status="$1"
	local data="$2"

	echo "Content-type: application/json"
	echo
	echo "{\"status\":\"$status\",\"data\":$data}"
}

split_commands() {
	local input="$1"
	input=$(echo "$input" | sed 's/;\([[:space:]]*\)\\/; \\/g')
	echo "$input" | sed -e 's/; \\/\n/g'
}

execute_command() {
	local command="$1"
	local start_time current_time end_time
	local start_time_iso end_time_iso elapsed_time
	local steps=() errors=() output
	local current=0 total=0 percentage=0
	local exit_code step_start step_end step_elapsed
	local steps_json errors_json data

	command=$(echo "$command" | sed 's/;\([[:space:]]*\)\\/; \\/g')

	start_time=$(date +%s)
	start_time_iso=$(format_time)

	if [[ "$command" =~ \$\{[^}]+\} ]]; then
		while [[ "$command" =~ \$\{([^}]+)\} ]]; do
			param_name="${BASH_REMATCH[1]}"
			param_value=$(get_query_param "$param_name" "true")
			command=${command//${BASH_REMATCH[0]}/$param_value}
		done
	fi

	readarray -t COMMANDS < <(split_commands "$command")
	total=${#COMMANDS[@]}
	[ "$total" -eq 0 ] && total=1 && COMMANDS=("$command")

	for cmd in "${COMMANDS[@]}"; do
		cmd=$(echo "$cmd" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
		((current++))

		step_start=$(date +%s)
		output=$(eval "$cmd" 2>&1)
		exit_code=$?
		step_end=$(date +%s)
		step_elapsed=$(calculate_elapsed $step_start $step_end)

		if [ $exit_code -eq 0 ]; then
			steps+=("{\"command\":\"$(escape_json "$cmd")\",\
\"output\":\"$(escape_json "$output")\",\
\"status\":\"success\",\
\"elapsed_time\":\"$step_elapsed\",\
\"step\":\"$current\",\
\"total\":\"$total\"}")
			errors+=("\"\"")
		else
			steps+=("{\"command\":\"$(escape_json "$cmd")\",\
\"output\":\"$(escape_json "$output")\",\
\"status\":\"error\",\
\"elapsed_time\":\"$step_elapsed\",\
\"step\":\"$current\",\
\"total\":\"$total\"}")
			errors+=("\"$(escape_json "$output")\"")
			break
		fi
	done

	end_time=$(date +%s)
	end_time_iso=$(format_time)
	elapsed_time=$(calculate_elapsed $start_time $end_time)
	percentage=$((current * 100 / total))

	steps_json=$(IFS=,; echo "${steps[*]}")
	errors_json=$(IFS=,; echo "${errors[*]}")
	data="{\"command\":\"$(escape_json "$command")\",\
\"start_time\":\"$start_time_iso\",\
\"end_time\":\"$end_time_iso\",\
\"elapsed_time\":\"$elapsed_time\",\
\"progress\":{\"current\":$current,\"total\":$total,\"percentage\":$percentage},\
\"steps\":[$steps_json],\
\"errors\":[$errors_json]}"

	[ $current -eq $total ] && send_response "success" "$data" || send_response "error" "$data"
}

main() {
	[ -z "$REQUEST_PATH" ] && { send_error_response "No request path provided"; exit 1; }
	[ ! -f "$ROUTES_CONF" ] && { send_error_response "Routes configuration not found"; exit 1; }
	command=$(grep "^$REQUEST_PATH:" "$ROUTES_CONF" | cut -d':' -f2-)
	[ -z "$command" ] && { send_error_response "Route not found"; exit 1; }
	if [ -n "$QUERY_STRING" ]; then
		while IFS='=' read -r name value; do
			[ -n "$name" ] && [ -n "$value" ] && \
			command=${command//\$\{$name\}/$(decode_url "$value")}
		done < <(echo "$QUERY_STRING" | tr '&' '\n')
	fi
	execute_command "$command"
}
main
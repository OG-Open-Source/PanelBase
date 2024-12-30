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
	local query_string="$QUERY_STRING"
	local param_value

	while IFS='=' read -d '&' key value; do
		[ "$key" = "$param_name" ] && param_value="$value" && break
	done < <(echo -n "$query_string")

	[ -n "$param_value" ] && decode_url "$param_value" || echo ""
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

normalize_command() {
	local input="$1"
	input=$(echo "$input" | sed 's/;\s*\\/; \\/g')
	input=$(echo "$input" | sed 's/\s\+/ /g')
	input=$(echo "$input" | sed 's/^ *//;s/ *$//')
	echo "$input"
}

split_commands() {
	local input="$1"
	local tmp_file="$INSTALL_DIR/cmd_$$.tmp"
	local count=0

	input=$(normalize_command "$input")

	count=$(echo "$input" | grep -o '; \\' | wc -l)
	count=$((count + 1))

	printf '2%.0s' $(seq 1 $count) >"$tmp_file"
	echo >>"$tmp_file"

	echo "$input" | sed 's/; \\/\n/g' | while read -r cmd; do
		[ -n "$cmd" ] && echo "$cmd" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' >>"$tmp_file"
	done

	echo "$tmp_file"
}

execute_command() {
	local command="$1"
	local original_command="$command"
	local start_time current_time end_time
	local start_time_iso end_time_iso elapsed_time
	local steps=() errors=() output
	local current=0 total=0 percentage=0
	local exit_code step_start step_end step_elapsed
	local steps_json errors_json data
	local has_error=false
	local cmd_file status_line cmd_index

	start_time=$(date +%s)
	start_time_iso=$(format_time)

	if [[ "$command" =~ \$\{[^}]+\} ]]; then
		while [[ "$command" =~ \$\{([^}]+)\} ]]; do
			param_name="${BASH_REMATCH[1]}"
			param_value=$(get_query_param "$param_name" "true")
			command=${command//${BASH_REMATCH[0]}/"$param_value"}
		done
	fi

	command=$(echo "$command" | sed 's/\\\\*/\\/g' | sed 's/;\\/; \\/g')
	original_command="$command"

	cmd_file="$INSTALL_DIR/cmd_$$.tmp"

	if [ -f "$cmd_file" ]; then
		read -r status_line <"$cmd_file"
		if [[ "$status_line" =~ "1" ]]; then
			if [[ "$original_command" =~ "; \\" ]]; then
				count=$(echo "$command" | sed 's/; \\/\n/g' | grep -v '^[[:space:]]*$' | wc -l)
				printf '2%.0s' $(seq 1 $count) >"$cmd_file"
				echo >>"$cmd_file"
				echo "$command" | sed 's/; \\/\n/g' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | grep -v '^[[:space:]]*$' >>"$cmd_file"
			else
				echo "2" >"$cmd_file"
				echo "$command" >>"$cmd_file"
			fi
		fi
	else
		if [[ "$original_command" =~ "; \\" ]]; then
			count=$(echo "$command" | sed 's/; \\/\n/g' | grep -v '^[[:space:]]*$' | wc -l)
			printf '2%.0s' $(seq 1 $count) >"$cmd_file"
			echo >>"$cmd_file"
			echo "$command" | sed 's/; \\/\n/g' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | grep -v '^[[:space:]]*$' >>"$cmd_file"
		else
			echo "2" >"$cmd_file"
			echo "$command" >>"$cmd_file"
		fi
	fi

	read -r status_line <"$cmd_file"
	total=${#status_line}

	cmd_index=$(echo "$status_line" | grep -b "2" | head -1 | cut -d: -f1)
	if [ -n "$cmd_index" ]; then
		current=$((cmd_index + 1))
		cmd=$(sed -n "$((current + 1))p" "$cmd_file")

		step_start=$(date +%s)
		output=$(eval "$cmd" 2>&1)
		exit_code=$?
		step_end=$(date +%s)
		step_elapsed=$(calculate_elapsed $step_start $step_end)

		if [ $exit_code -eq 0 ]; then
			status_line="${status_line:0:$cmd_index}0${status_line:$((cmd_index + 1))}"
			steps+=("{\"command\":\"$(escape_json "$cmd")\",\
\"output\":\"$(escape_json "$output")\",\
\"status\":\"success\",\
\"elapsed_time\":\"$step_elapsed\",\
\"step\":\"$current\",\
\"total\":\"$total\"}")
			errors+=("\"\"")
		else
			status_line="${status_line:0:$cmd_index}1${status_line:$((cmd_index + 1))}"
			has_error=true
			steps+=("{\"command\":\"$(escape_json "$cmd")\",\
\"output\":\"$(escape_json "$output")\",\
\"status\":\"error\",\
\"elapsed_time\":\"$step_elapsed\",\
\"step\":\"$current\",\
\"total\":\"$total\"}")
			errors+=("\"$(escape_json "$output")\"")
		fi

		sed -i "1c\\$status_line" "$cmd_file"
	fi

	end_time=$(date +%s)
	end_time_iso=$(format_time)
	elapsed_time=$(calculate_elapsed $start_time $end_time)
	percentage=$((current * 100 / total))

	steps_json=$(
		IFS=,
		echo "${steps[*]}"
	)
	errors_json=$(
		IFS=,
		echo "${errors[*]}"
	)
	data="{\"command\":\"$(escape_json "$command")\",\
\"start_time\":\"$start_time_iso\",\
\"end_time\":\"$end_time_iso\",\
\"elapsed_time\":\"$elapsed_time\",\
\"progress\":{\"current\":$current,\"total\":$total,\"percentage\":$percentage},\
\"steps\":[$steps_json],\
\"errors\":[$errors_json],\
\"status_line\":\"$status_line\"}"

	[ "$has_error" = true ] && send_response "error" "$data" || send_response "success" "$data"
}

main() {
	[ -z "$REQUEST_PATH" ] && {
		send_error_response "No request path provided"
		exit 1
	}
	[ ! -f "$ROUTES_CONF" ] && {
		send_error_response "Routes configuration not found"
		exit 1
	}
	command=$(grep "^$REQUEST_PATH:" "$ROUTES_CONF" | cut -d':' -f2-)
	[ -z "$command" ] && {
		send_error_response "Route not found"
		exit 1
	}
	if [ -n "$QUERY_STRING" ]; then
		while IFS='=' read -r name value; do
			[ -n "$name" ] && [ -n "$value" ] &&
				command=${command//\$\{$name\}/$(decode_url "$value")}
		done < <(echo "$QUERY_STRING" | tr '&' '\n')
	fi
	execute_command "$command"
}
main

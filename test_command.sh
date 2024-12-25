#!/bin/bash

# 格式化時間
format_time() { date -u "+%Y-%m-%dT%H:%M:%SZ"; }

# 計算經過時間
calculate_elapsed() {
	local start=$1
	local end=$2
	echo "$((end - start))s"
}

# JSON 轉義
escape_json() {
	local text="$1"
	text="${text//\\/\\\\}"
	text="${text//\"/\\\"}"
	text="${text//$'\b'/\\b}"
	text="${text//$'\f'/\\f}"
	text="${text//$'\n'/\\n}"
	text="${text//$'\r'/\\r}"
	text="${text//$'\t'/\\t}"
	echo "$text"
}

# 執行命令並返回 JSON 格式結果
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
			steps+=("{\"command\":\"$(escape_json "$cmd")\",\"output\":\"$(escape_json "$output")\",\"status\":\"success\",\"elapsed_time\":\"$step_elapsed\",\"step\":\"$current\",\"total\":\"$total\"}")
			errors+=("\"\"")
		else
			steps+=("{\"command\":\"$(escape_json "$cmd")\",\"output\":\"$(escape_json "$output")\",\"status\":\"error\",\"elapsed_time\":\"$step_elapsed\",\"step\":\"$current\",\"total\":\"$total\"}")
			errors+=("\"$(escape_json "$output")\"")
			break
		fi
	done

	local end_time=$(date +%s)
	local end_time_iso=$(format_time)
	local elapsed_time=$(calculate_elapsed $start_time $end_time)
	local percentage=$((current * 100 / total))

	local steps_json=$(IFS=,; echo "${steps[*]}")
	local errors_json=$(IFS=,; echo "${errors[*]}")

	local data="{\"command\":\"$(escape_json "$command")\",\"start_time\":\"$start_time_iso\",\"end_time\":\"$end_time_iso\",\"elapsed_time\":\"$elapsed_time\",\"progress\":{\"current\":$current,\"total\":$total,\"percentage\":$percentage},\"steps\":[$steps_json],\"errors\":[$errors_json]}"

	if [ $current -eq $total ]; then
		echo "{\"status\":\"success\",\"data\":$data}"
	else
		echo "{\"status\":\"error\",\"data\":$data}"
	fi
}

# 如果沒有提供命令，顯示使用方法
if [ $# -eq 0 ]; then
	echo "使用方法: $0 \"command\""
	echo "例如: $0 \"ls -la; pwd\""
	exit 1
fi

# 執行命令並格式化輸出
execute_command "$1" | jq .
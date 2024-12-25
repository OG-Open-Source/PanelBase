#!/bin/bash

ROUTES_CONF="/opt/panelbase/config/routes.conf"
FORMAT_JSON=true  # 是否使用 jq 格式化輸出

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

# 清理命令字符串
clean_command() {
	local cmd="$1"
	# 清理分號後的空格
	echo "$cmd" | sed 's/;\s\+/;/g'
}

# 執行命令並返回 JSON 格式結果
execute_command() {
	local command="$1"
	# 預處理命令
	command=$(clean_command "$command")
	
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

# 顯示使用方法
show_usage() {
	echo "使用方法:"
	echo "  $0 [選項] <路由路徑>"
	echo
	echo "選項:"
	echo "  -l, --list     列出所有可用的路由"
	echo "  -r, --raw      輸出原始 JSON（不格式化）"
	echo "  -h, --help     顯示此幫助信息"
	echo
	echo "例如:"
	echo "  $0 /api/system/cpu     # 測試 CPU 信息路由"
	echo "  $0 -l                  # 列出所有路由"
	echo "  $0 -r /api/system/cpu  # 輸出原始 JSON"
}

# 列出所有路由
list_routes() {
	echo "可用的路由："
	echo "----------------------------------------"
	while IFS=: read -r route cmd; do
		[ -z "$route" ] && continue
		[[ "$route" =~ ^#.*$ ]] && continue
		# 清理命令中的多餘空格
		cmd=$(clean_command "$cmd")
		printf "%-30s %s\n" "$route" "$cmd"
	done < "$ROUTES_CONF"
}

# 解析命令行參數
while [[ $# -gt 0 ]]; do
	case $1 in
		-l|--list)
			list_routes
			exit 0
			;;
		-r|--raw)
			FORMAT_JSON=false
			shift
			;;
		-h|--help)
			show_usage
			exit 0
			;;
		*)
			REQUEST_PATH="$1"
			shift
			;;
	esac
done

# 檢查必要條件
[ ! -f "$ROUTES_CONF" ] && { echo "錯誤：找不到 $ROUTES_CONF 文件"; exit 1; }
[ -z "$REQUEST_PATH" ] && { show_usage; exit 1; }

# 獲取並執行命令
command=$(grep "^$REQUEST_PATH:" "$ROUTES_CONF" | cut -d':' -f2-)
[ -z "$command" ] && { echo "錯誤：找不到路由 $REQUEST_PATH"; exit 1; }

# 執行命令並輸出結果
if [ "$FORMAT_JSON" = true ]; then
	execute_command "$command" | jq .
else
	execute_command "$command"
fi
#!/bin/bash

# 載入共用函數
source /opt/panelbase/cgi-bin/common.cgi

# 檢查是否為 API 請求
if echo "$QUERY_STRING" | grep -q "action="; then
	# 獲取請求參數
	PATH_INFO=$(echo "$QUERY_STRING" | grep -oP 'path=\K[^&]+')
	METHOD="$REQUEST_METHOD"
	ACTION=$(echo "$QUERY_STRING" | grep -oP 'action=\K[^&]+')

	# 路由請求
	route_request "$PATH_INFO" "$METHOD" "$ACTION"
	exit 0
fi

# 如果不是 API 請求，返回 HTML 頁面
echo "Content-type: text/html"
echo

# 檢查緩存
cache_key="panel_html"
if ! html=$(get_cache "$cache_key"); then
	# 生成 HTML 內容
	html=$(cat << EOF
<!DOCTYPE html>
<html lang="zh-TW">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>PanelBase 管理面板</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			margin: 0;
			padding: 20px;
			background-color: #f5f5f5;
		}
		.container {
			max-width: 1200px;
			margin: 0 auto;
			background-color: white;
			padding: 20px;
			border-radius: 5px;
			box-shadow: 0 2px 5px rgba(0,0,0,0.1);
		}
		.header {
			background-color: #2c3e50;
			color: white;
			padding: 20px;
			margin: -20px -20px 20px -20px;
			border-radius: 5px 5px 0 0;
		}
		.section {
			margin-bottom: 20px;
			padding: 15px;
			border: 1px solid #ddd;
			border-radius: 4px;
		}
		.info-item {
			margin: 10px 0;
		}
		.status-ok {
			color: green;
		}
		.status-error {
			color: red;
		}
		.random-number {
			font-size: 1.2em;
			font-weight: bold;
			color: #2c3e50;
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>PanelBase 管理面板</h1>
		</div>
EOF
)
	# 設置緩存
	set_cache "$cache_key" "$html"
fi

# 輸出 HTML
echo "$html"
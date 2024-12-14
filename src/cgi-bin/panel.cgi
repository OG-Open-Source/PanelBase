#!/bin/bash

#使用安裝腳本
[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source ~/utilkit.sh

# 檢查是否為 API 請求
if echo "$QUERY_STRING" | grep -q "action="; then
	# 獲取 action 參數
	ACTION=$(echo "$QUERY_STRING" | grep -oP 'action=\K[^&]+')

	case "$ACTION" in
		"get_username")
			AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')

			USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "/opt/panelbase/config/sessions.conf")

			echo "Content-type: text/plain"
			echo
			echo "$USERNAME"
			;;

		"get_system_info")
			# 檢查 lighttpd 狀態
			if systemctl is-active lighttpd >/dev/null 2>&1; then
				LIGHTTPD_STATUS="運行中"
			else
				LIGHTTPD_STATUS="已停止"
			fi

			# 返回 JSON 格式的數據
			echo "Content-type: application/json"
			echo "Cache-Control: no-cache"
			echo
			cat << EOF
{
	"cpu": "$(CPU_USAGE)",
	"memory": "$(MEM_USAGE)",
	"disk": "$(DISK_USAGE)",
	"lighttpd_status": "${LIGHTTPD_STATUS}"
}
EOF
			;;
		*)
			# 未知的 action
			echo "Content-type: application/json"
			echo "Status: 400"
			echo
			echo '{"error": "Unknown action"}'
			;;
	esac
	exit 0
fi

# 如果不是 API 請求，返回 HTML 頁面
echo "Content-type: text/html"
echo

cat << EOF
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
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>PanelBase 管理面板</h1>
		</div>
EOF

# 系統資訊
echo '<div class="section">'
echo '<h2>系統資訊</h2>'

# CPU 使用率
echo '<div class="info-item">'
echo "<strong>CPU 使用率：</strong>"
CPU_USAGE
echo '</div>'

# 記憶體使用情況
echo '<div class="info-item">'
echo "<strong>記憶體使用情況：</strong>"
MEM_USAGE
echo '</div>'

# 磁碟使用情況
echo '<div class="info-item">'
echo "<strong>磁碟使用情況：</strong>"
DISK_USAGE
echo '</div>'

echo '</div>'

# 服務狀態
echo '<div class="section">'
echo '<h2>服務狀態</h2>'

echo '<div class="info-item">'
echo "<strong>Lighttpd 狀態：</strong>"
if systemctl is-active lighttpd >/dev/null 2>&1; then
	echo '<span class="status-ok">運行中</span>'
else
	echo '<span class="status-error">已停止</span>'
fi
echo '</div>'

echo '</div>'

cat << EOF
	</div>
	<script>
		// 自動重新整理頁面
		setTimeout(function() {
			location.reload();
		}, 1000);
	</script>
</body>
</html>
EOF
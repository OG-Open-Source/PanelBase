#!/bin/bash

# 檢查是否為 API 請求
if echo "$QUERY_STRING" | grep -q "action="; then
	# 獲取 action 參數
	ACTION=$(echo "$QUERY_STRING" | grep -oP 'action=\K[^&]+')

	case "$ACTION" in
		"get_username")
			# 從 cookie 中獲取 token
			AUTH_TOKEN=$(echo "$HTTP_COOKIE" | grep -oP 'auth_token=\K[^;]+')

			# 從 sessions.conf 中獲取用戶名
			USERNAME=$(awk -F: -v token="$AUTH_TOKEN" '$1 == token {print $2}' "/opt/panelbase/config/sessions.conf")

			echo "Content-type: text/plain"
			echo
			echo "$USERNAME"
			;;

		"get_system_info")
			# 獲取 CPU 使用率
			CPU_USAGE=$(top -bn1 | grep "Cpu(s)" | sed 's/.*, *\([0-9.]*\)%* id.*/\1/' | awk '{print 100 - $1}')

			# 獲取記憶體使用情況
			MEM_TOTAL=$(free -m | awk '/Mem:/ {print $2}')
			MEM_USED=$(free -m | awk '/Mem:/ {print $3}')
			MEM_FREE=$(free -m | awk '/Mem:/ {print $4}')
			MEM_INFO="總計: ${MEM_TOTAL}MB 已使用: ${MEM_USED}MB 可用: ${MEM_FREE}MB"

			# 獲取磁碟使用情況
			DISK_TOTAL=$(df -h / | tail -n1 | awk '{print $2}')
			DISK_USED=$(df -h / | tail -n1 | awk '{print $3}')
			DISK_FREE=$(df -h / | tail -n1 | awk '{print $4}')
			DISK_USAGE=$(df -h / | tail -n1 | awk '{print $5}')
			DISK_INFO="總計: ${DISK_TOTAL} 已使用: ${DISK_USED} 可用: ${DISK_FREE} 使用率: ${DISK_USAGE}"

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
	"cpu": "${CPU_USAGE}",
	"memory": "${MEM_INFO}",
	"disk": "${DISK_INFO}",
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
CPU_USAGE=$(top -bn1 | grep "Cpu(s)" | sed 's/.*, *\([0-9.]*\)%* id.*/\1/' | awk '{print 100 - $1}')
echo "${CPU_USAGE}%"
echo '</div>'

# 記憶體使用情況
echo '<div class="info-item">'
echo "<strong>記憶體使用情況：</strong>"
free -h | grep "Mem:" | awk '{print "總計: " $2 "  已使用: " $3 "  可用: " $4}'
echo '</div>'

# 磁碟使用情況
echo '<div class="info-item">'
echo "<strong>磁碟使用情況：</strong>"
df -h / | tail -n 1 | awk '{print "總計: " $2 "  已使用: " $3 "  可用: " $4 "  使用率: " $5}'
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
		}, 10000);
	</script>
</body>
</html>
EOF
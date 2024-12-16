#!/bin/bash

# 只返回 HTML 頁面
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
			<div id="username"></div>
		</div>

		<div class="section">
			<h2>系統資訊</h2>
			<div id="systemInfo">載入中...</div>
		</div>

		<div class="section">
			<h2>服務狀態</h2>
			<div id="serviceStatus">載入中...</div>
		</div>
	</div>

	<script>
		// 獲取用戶名
		async function getUsername() {
			try {
				const response = await fetch('/api/panel/username');
				const username = await response.json();
				document.getElementById('username').textContent = '歡迎，' + username;
			} catch (error) {
				console.error('Error fetching username:', error);
			}
		}

		// 獲取系統資訊
		async function getSystemInfo() {
			try {
				const response = await fetch('/api/panel/system_info');
				const data = await response.json();
				
				document.getElementById('systemInfo').innerHTML = \`
					<div class="info-item">
						<strong>CPU 使用率：</strong>\${data.cpu}%
					</div>
					<div class="info-item">
						<strong>記憶體使用情況：</strong>\${data.memory}
					</div>
					<div class="info-item">
						<strong>磁碟使用情況：</strong>\${data.disk}
					</div>
					<div class="info-item">
						<strong>隨機數：</strong>
						<span class="random-number">\${data.random}</span>
					</div>
				\`;

				document.getElementById('serviceStatus').innerHTML = \`
					<div class="info-item">
						<strong>Lighttpd 狀態：</strong>
						<span class="\${data.lighttpd_status === '運行中' ? 'status-ok' : 'status-error'}">
							\${data.lighttpd_status}
						</span>
					</div>
				\`;
			} catch (error) {
				console.error('Error fetching system info:', error);
				document.getElementById('systemInfo').innerHTML = '<div class="error">無法載入系統資訊</div>';
				document.getElementById('serviceStatus').innerHTML = '<div class="error">無法載入服務狀態</div>';
			}
		}

		// 初始化
		getUsername();
		getSystemInfo();

		// 定期更新
		setInterval(getSystemInfo, 1000);
	</script>
</body>
</html>
EOF
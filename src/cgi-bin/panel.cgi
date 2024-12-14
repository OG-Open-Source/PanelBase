#!/bin/bash

# 設置 HTTP 頭
echo "Content-type: text/html"
echo ""

# HTML 頭部
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

# 系統資訊區段
echo '<div class="section">'
echo '<h2>系統資訊</h2>'

# CPU 資訊
echo '<div class="info-item">'
echo "<strong>CPU 使用率：</strong>"
CPU_USAGE=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}')
echo "$CPU_USAGE%"
echo '</div>'

# 記憶體資訊
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

# 服務狀態區段
echo '<div class="section">'
echo '<h2>服務狀態</h2>'

# lighttpd 狀態
echo '<div class="info-item">'
echo "<strong>Lighttpd 狀態：</strong>"
if systemctl is-active lighttpd >/dev/null 2>&1; then
    echo '<span class="status-ok">運行中</span>'
else
    echo '<span class="status-error">已停止</span>'
fi
echo '</div>'

echo '</div>'

# HTML 尾部
cat << EOF
    </div>
    <script>
        // 自動重新整理頁面
        setTimeout(function() {
            location.reload();
        }, 30000);
    </script>
</body>
</html>
EOF 
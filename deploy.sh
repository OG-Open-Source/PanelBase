#!/bin/bash

get_port() {
	while true; do
		port=$(((RANDOM % (49151 - 1024 + 1)) + 1024))
		if ! nc -z localhost $port &>/dev/null; then
			echo $port
			return
		fi
		sleep 1
	done
}

get_entry() {
	head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9!@#$%^&*()_+-=' | head -c 16
}

get_ip() {
	curl -s -4 ifconfig.me
}

# 檢查是否已經運行
check_running() {
	if [ -f ".pid" ]; then
		pid=$(cat .pid)
		if ps -p $pid >/dev/null 2>&1; then
			return 0
		fi
	fi
	return 1
}

# 停止服務
stop_service() {
	if check_running; then
		pid=$(cat .pid)
		kill $pid
		rm .pid
		echo "PanelBase service stopped"
	else
		echo "PanelBase service is not running"
	fi
}

# 啟動服務
start_service() {
	if check_running; then
		echo "PanelBase service is already running"
		return
	fi

	# 檢查 Go 版本
	GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
	MAJOR_VERSION=$(echo $GO_VERSION | cut -d. -f1)
	MINOR_VERSION=$(echo $GO_VERSION | cut -d. -f2)

	if [[ "$MAJOR_VERSION" != "1" || "$MINOR_VERSION" -gt "19" ]]; then
		echo "Warning: PanelBase requires Go version 1.19 or lower"
		echo "Current Go version: $GO_VERSION"
		read -p "Do you want to continue? (y/n) " -n 1 -r
		echo
		if [[ ! $REPLY =~ ^[Yy]$ ]]; then
			exit 1
		fi
	fi

	# 下載最新版本
	echo "Downloading PanelBase..."
	if [ -d "PanelBase" ]; then
		rm -rf PanelBase
	fi
	git clone https://github.com/OG-Open-Source/PanelBase.git
	cd PanelBase

	# 初始化 Go 模組
	echo "Initializing Go module..."
	go mod init github.com/OG-Open-Source/PanelBase
	go mod tidy

	# 設置環境變數
	PORT=$(get_port)
	ENTRY=$(get_entry)
	IP=$(get_ip)

	echo "PORT=$PORT" >.env
	echo "ENTRY=$ENTRY" >>.env

	# 安裝依賴
	echo "Installing dependencies..."
	go mod download

	# 建置程式碼
	echo "Building PanelBase..."
	go build -o panelbase ./cmd/panelbase/main.go

	# 啟動服務
	./panelbase &
	echo $! >.pid
	echo "PanelBase service started"

	# 顯示連線資訊
	echo "============================================"
	echo "PanelBase Agent Connection Details:"
	echo "--------------------------------------------"
	echo "IP Address: $IP"
	echo "Port: $PORT"
	echo "Entry Token: $ENTRY"
	echo "============================================"
	echo "Please use these details to connect through the web interface."
}

# 重啟服務
restart_service() {
	stop_service
	start_service
}

# 根據參數執行對應操作
case "$1" in
start)
	start_service
	;;
stop)
	stop_service
	;;
restart)
	restart_service
	;;
*)
	echo "Usage: $0 {start|stop|restart}"
	exit 1
	;;
esac

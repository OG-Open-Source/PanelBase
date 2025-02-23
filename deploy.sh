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

PORT=$(get_port)
ENTRY=$(get_entry)

echo "PORT: $PORT"
echo "ENTRY: $ENTRY"

# 建置程式碼
go build -o panelbase ./cmd/panelbase/main.go

# 設定環境變數
echo "PORT=$PORT" > .env
echo "ENTRY=$ENTRY" >> .env 
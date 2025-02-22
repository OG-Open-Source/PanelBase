#!/bin/bash
set -e

if [ ! -f go.mod ]; then
    go mod init github.com/OG-Open-Source/PanelBase
fi

go mod download

mkdir -p internal/{config,handlers} web/{static/js,templates}
mv config/config.go internal/config/ 2>/dev/null || true
mv utils/external.go internal/handlers/ 2>/dev/null || true
mv js/panel.js web/static/js/ 2>/dev/null || true
mv index.html panel.html web/templates/ 2>/dev/null || true
mv routes.json web/ 2>/dev/null || true

go build -o panelbase ./cmd/panelbase
./panelbase

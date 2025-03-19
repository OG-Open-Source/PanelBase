#!/bin/bash

# Exit on any error
set -e

echo "Building PanelBase..."
go build -o panelbase ./cmd/panelbase

echo "Creating required directories..."
mkdir -p data logs web/static/css web/static/js web/static/img

echo "Starting PanelBase..."
./panelbase
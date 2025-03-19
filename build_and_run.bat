@echo off
echo Building PanelBase...
go build -o panelbase.exe ./cmd/panelbase

echo Creating required directories...
mkdir data 2>nul
mkdir logs 2>nul
mkdir web\static\css 2>nul
mkdir web\static\js 2>nul
mkdir web\static\img 2>nul

echo Starting PanelBase...
panelbase.exe 
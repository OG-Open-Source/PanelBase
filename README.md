# PanelBase

A lightweight Go-based management panel for Linux servers.

## Features

_(Add key features here as they are developed)_

- JWT-based Authentication (Web session & API tokens)
- API Token Management (Create, List, Update, Delete)
- Basic Permission System
- Persistent Token Storage (BoltDB)
- Customizable Logging
- Configuration Management (JSON-based)

## Getting Started

_(Add instructions on how to build and run the project)_

```bash
# Example
go run cmd/panelbase/main.go
```

## Configuration

Configuration is managed via JSON files in the `config` directory:

- `config/config.json`: Main application configuration
  - Server settings (host, port, mode)
  - Authentication settings (JWT secret, token expiry, cookie name)

Example configuration:
```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 35960,
    "mode": "release"
  },
  "auth": {
    "jwt_secret": "your-secret-key",
    "token_expiry": "24h",
    "cookie_name": "panelbase_jwt"
  }
}
```

If the configuration file doesn't exist, a default configuration will be created automatically.

## Changelog

### [Unreleased]
- Added JSON-based configuration management
- Added automatic default configuration creation
- Added configuration validation and loading
- Added configuration saving functionality
- 优化日志格式：服务器启动前的 bootstrap 相关日志使用 Go 标准日志格式，服务器启动后的日志使用 RFC3339 时间格式

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

### Changed

- **Refactor**: Reorganized project structure for API versioning and separation of concerns.

  - Moved API v1 routes and handlers to `internal/api/v1/`.
  - Created placeholder for API v2 (`internal/api/v2/`).
  - Moved core services (Auth Utils, API Token Service, UI Settings Service, User Service) to `pkg/`.
  - Updated all imports and references to reflect the new structure.
  - Standardized API token routes: `/account/token` for self-service and `/users/:user_id/token` for admin actions, both using `:id` path parameter for specific token operations (GET, PATCH, DELETE).

- **Refactor**: Reorganized API routes and handlers for versioning (`v1`, `v2` placeholder).

  - Moved v1 routes to `internal/api/v1/routes.go`.
  - Moved v1 handlers (auth, token, settings) to `internal/api/v1/handlers/` subdirectories.
  - Kept service logic in original packages (`internal/auth`, `internal/api_token`, `internal/ui_settings`). // NOTE: This line seems outdated now.
  - Updated imports accordingly.

- Added JSON-based configuration management
- Added automatic default configuration creation
- Added configuration validation and loading
- Added configuration saving functionality

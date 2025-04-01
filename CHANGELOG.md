# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- (Future changes will go here)

## [0.1.0] - 2025-04-01

### Added

- Initialized Go project (`go mod init github.com/OG-Open-Source/PanelBase`).
- Added Gin web framework (`go mod tidy`).
- Created basic Gin server structure in `cmd/panelbase/main.go`.
- Implemented configuration loading from `configs/config.toml` and environment variables (`PANELBASE_JWT_SECRET`) in `internal/config/config.go`.
- Updated `cmd/panelbase/main.go` to load configuration on startup and use it to set Gin mode and server address.
- Implemented standardized API response format (`internal/server/response.go`) with `Response` struct and `SuccessResponse`/`ErrorResponse` helpers.
- Implemented JWT authentication middleware (`internal/middleware/auth.go`) using `github.com/golang-jwt/jwt/v5`.
- Applied JWT middleware to protected API routes in `internal/routes/routes.go`.
- Created data models for User, APIToken, and UsersConfig in `internal/models` reflecting scope-based permissions.
- Added bootstrap functionality in `internal/bootstrap/bootstrap.go` to automatically create and initialize configuration files:
  - `themes.json` with empty themes list
  - `commands.json` with empty commands list
  - `plugins.json` with empty plugins list
  - `users.json` with default admin user (username: admin, password: admin) and random user ID
  - `config.toml` with server settings and feature flags

### Changed

- **Project Structure:** Established new directory structure (`themes/`, categorized `commands/`). (Reflected in planning, code changes pending).
- **API:** Adopted standardized API response format (Implementation started in `internal/server/response.go`).
- **Security:**
  - Changed JWT secret source from environment variable to `configs/users.json` (`jwt_secret` field).
  - Implemented JWT validation middleware.
  - Redesigned permission system to use Scopes (e.g., `resource:action:target`).
  - Removed `role` field from user data structure in favor of explicit Scopes.
- **Configuration:**
  - Added automatic configuration file creation on startup
  - Implemented random port generation (1024-49151) with availability check
  - Added feature flags for commands and plugins in `config.toml`
  - Changed configuration format from YAML to TOML
  - Updated JSON configuration files to use consistent key ordering
  - Simplified configuration structure to only include essential fields
  - Added cookie name configuration for JWT authentication
  - Added server mode configuration (debug/release)
  - Changed user ID generation to use random string
  - Removed default themes, commands, and plugins from configuration files
- **Routing:**
  - Fixed static file serving path to avoid route conflicts by using a custom `NoRoute` handler instead of `router.Static` for the root path.
  - Restructured API routes to match specified endpoints
  - Configured root path `/` to serve the entire `./web` directory via the custom `NoRoute` handler.
  - Added fallback to `index.html` for non-API 404 errors (SPA support) in the custom `NoRoute` handler.

### Fixed

- Fixed configuration loading error by updating `config.go` to use TOML format instead of YAML
- Fixed route conflicts by moving static file serving to `/static` path
- Fixed web interface not displaying by adding proper static file serving
- Fixed API route structure to only include specified endpoints
- Fixed JWT cookie name configuration error in auth middleware
- Fixed server mode configuration error in main.go
- Fixed linter errors in `routes.go` related to incorrect function arguments
- Fixed Gin routing panic caused by conflict between `router.Static("/", ...)` and API routes by using a custom `NoRoute` handler.

[Unreleased]: https://github.com/OG-Open-Source/PanelBase/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/OG-Open-Source/PanelBase/releases/tag/v0.1.0

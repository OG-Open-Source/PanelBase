# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- (Future changes will go here)

## [0.4.0] - 2025-04-04

### Changed

- **API Token Implementation**: 
    - Changed API token generation from opaque random strings to JWTs signed with a user-specific secret (`api.jwt_secret` in `users.json`).
    - Implemented a limit of 10 active API tokens per user.
    - `POST /api/v1/users/token` endpoint now requires `name`, `scopes`, and `duration` in the request body.
    - The response for token creation now includes the token `id`, metadata (`name`, `description`, `scopes`, `created_at`, `expires_at`), and the signed `token` (JWT string).
- **User Data Management**:
    - Refactored `users.json` file reading and writing logic into a new dedicated service: `internal/user/userservice.go`, including mutex locking for safe concurrent access.
    - Updated `bootstrap` process to use `userservice` for initializing the `users.json` file.
    - Updated `users.json` structure: added `api.jwt_secret` field per user and changed `api.tokens` to store token metadata as a map keyed by token ID.
- **API Structure**:
    - Consolidated API token management routes under `/api/v1/users/token` (POST, PUT, DELETE). Removed the standalone `/api/v1/token` group.
    - Removed the `GET /api/v1/users/token` route.

### Fixed

- Fixed linter error caused by redeclaration of `UsersConfig` struct in the `models` package by removing `internal/models/users_config.go`.
- Corrected various issues during refactoring, including function signatures, data structure mismatches, and import errors.

## [0.3.0] - 2025-04-03

### Added

- Basic `/api/v1/auth/login` handler (`internal/auth/auth_service.go`): validates credentials, generates JWT, sets cookie, returns token.
- Basic `/api/v1/auth/token` handler (`internal/auth/auth_service.go`) for refreshing `web_session` tokens (requires valid session token).
- `/logs` directory creation during bootstrap (`internal/bootstrap/bootstrap.go`).
- Logging configuration (`cmd/panelbase/main.go`) to output logs to console and `/logs/panelbase.log`.
- Graceful server shutdown implementation (`cmd/panelbase/main.go`).
- `/api/v1/users/token` routes (GET, POST, PUT, DELETE) placeholders for managing user's own API tokens.
- Basic permission checking framework (`internal/middleware/permissions.go`) with `CheckPermission` and `CheckReadPermission` helpers.

### Changed

- **Authentication:**
    - Enhanced `AuthMiddleware` (`internal/middleware/auth.go`) to check JWT from Cookie/Header and store actual audience.
    - Updated JWT Claims (`internal/middleware/auth.go`) to use `username` tag.
    - Moved `POST /api/v1/auth/token` route under authentication protection.
    - Changed user/token permission structure to `map[string][]string` (`models.UserPermissions`).
    - Updated auth handlers and middleware to use new permission structure.
- **API Structure:**
    - Consolidated API token management for the current user under `/api/v1/users/token` (replacing the previously planned `/api/v1/account/tokens`).
- **Models:**
    - Updated `User` model (`internal/models/user.go`) for correct JSON unmarshalling of `api: {tokens: [...]}`.
- **Server:**
    - Standardized API response format (`internal/server/response.go`).
- **Configuration:**
    - Set default `server.mode` to `"release"` in bootstrap.
    - Changed `main.go` to use `server.mode` directly for Gin mode setting.
    - Removed explicit `gin.TestMode` support in `main.go`.
- **Code:**
    - Translated comments in `bootstrap.go` and `main.go` to English.
    - Removed verbose logging from `main.go` startup.

### Fixed

- Fixed `auth_service.go` syntax errors.
- Resolved `undefined: auth.LoginHandler` error.
- Fixed JSON unmarshal error for User model.
- Corrected Gin mode setting logic.
- Fixed `Invalid audience format` error in `RefreshTokenHandler`.

## [0.2.0] - 2025-04-02

### Changed

- **API:**
  - Consolidated resource endpoints to use the base path (e.g., `/api/v1/commands`) for all methods (GET, POST, PUT, DELETE).
  - Defined specific behaviors based on HTTP method and request body content (details below require handler implementation):
    - `GET /resource`: List all items. Optionally provide `{"id": "..."}` in body (non-standard) or query param (preferred) to get a specific item.
    - `POST /resource`: Behavior depends on body. For commands, `{"url": "..."}` could mean download/install, `{"id": "..."}` could mean execute (non-standard), otherwise create.
    - `PUT /resource`: Update item(s). Requires `{"id": "..."}` in body for a specific item. Omitting `id` might imply updating the list/batch update (non-standard).
    - `DELETE /resource`: Delete item(s). Requires `{"id": "..."}` in body for a specific item. Omitting `id` might imply clearing the list (non-standard).
- **Routing:**
  - Updated API route definitions in `internal/routes/routes.go` to support GET, POST, PUT, DELETE on the base resource paths (`/api/v1/{resource}`), removing `/resource/:id` style routes.

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
  - `config.toml` with server settings (including random port with availability check), feature flags, auth settings (cookie name, expiration).

### Changed

- **Project Structure:** Established new directory structure (`themes/`, categorized `commands/`). (Reflected in planning, code changes pending).
- **API:** Adopted standardized API response format.
- **Security:**
  - Changed JWT secret source from environment variable to `configs/users.json` (`jwt_secret` field).
  - Implemented JWT validation middleware.
  - Redesigned permission system to use Scopes (e.g., `resource:action:target`).
  - Removed `role` field from user data structure in favor of explicit Scopes.
- **Configuration:**
  - Added automatic configuration file creation on startup.
  - Changed configuration format from YAML to TOML.
  - Updated JSON configuration files to use consistent key ordering.
  - Simplified configuration structure to only include essential fields.
  - Changed user ID generation to use random string.
  - Removed default themes, commands, and plugins from initial configuration files.
- **Routing:**
  - Restructured API routes initially to match specified limited endpoints (`/api/v1/{commands, plugins, themes, users}` GET only, plus `/api/v1/auth/{login, token}`).

### Fixed

- Fixed configuration loading error by updating `config.go` to use TOML format instead of YAML.
- Fixed route conflicts between static file serving and API routes by using a custom `NoRoute` handler for the root path (`/`) instead of `router.Static`.
- Fixed web interface not displaying by implementing proper static file serving within the `NoRoute` handler, including SPA fallback to `index.html`.
- Fixed `Invalid path` error in `NoRoute` handler by using absolute paths for security checks.
- Fixed API route structure initially to only include specified limited endpoints.
- Fixed JWT cookie name configuration error in `internal/middleware/auth.go`.
- Fixed server mode configuration error in `cmd/panelbase/main.go`.
- Fixed various linter errors in `internal/routes/routes.go` related to incorrect function arguments during development.
- Fixed Gin routing panic caused by conflict between `router.Static("/", ...)` and API routes.

[Unreleased]: https://github.com/OG-Open-Source/PanelBase/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/OG-Open-Source/PanelBase/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/OG-Open-Source/PanelBase/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/OG-Open-Source/PanelBase/releases/tag/v0.1.0

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- (Future changes will go here)

## [0.5.0] - 2025-04-05 logger 重要更新及轉變

### Changed

- **Logging**:
  - Replaced default Gin logger with a custom logger (`internal/middleware/logger.go`).
  - New format: `[WEB/API] RFC3339_UTC_Timestamp | BgColored Status Reset | Latency | ClientIP | TextColored Method Reset PathErrorMessage`.
  - Request body (if not empty) is displayed on a new indented line, prefixed with the User ID (`USER_ID: ...`). If User ID (sub) is unavailable, `-:` is shown.
  - Request body is formatted as compact single-line JSON (JSONL style) if valid JSON.
  - Log prefix (`[API]` or `[WEB]`) determined by request path (`/api/` prefix).
  - Status code uses background colors with space padding.
  - HTTP method uses text colors.
  - Removed trailing `|` from the main log line.
  - Log timestamps use UTC RFC3339 format.
- **Middleware**: Added `CacheRequestBody` middleware to enable reading request body in logger.

## [0.4.0] - 2025-04-04

### Changed

- **Token Management**:
  - Integrated `tokenstore` (using BoltDB) for persistent storage of API and session token metadata (ID, UserID, Name, Scopes, Timestamps).
  - Removed `Tokens map[string]APIToken` from `UserAPISettings` in `users.json` model.
  - Updated `CreateAPIToken` service function to store token metadata in `tokenstore` instead of `users.json`.
  - Updated `LoginHandler` to store session token metadata in `tokenstore`.
  - Updated `DeleteTokenHandler` to revoke tokens in `tokenstore` instead of modifying `users.json`.
  - Refactored `CreateTokenHandler`, `GetTokensHandler`, and `UpdateTokenHandler` to use `tokenstore` for checking token existence, ownership, and revocation status, removing reliance on the old `Tokens` map in the user object.
  - Removed redundant `user.UpdateUser` calls in token handlers as user object is no longer modified directly for token operations.
  - Implemented listing of user's API tokens in `GET /api/v1/users/token` (when no ID is provided) by adding `GetUserTokens` to `tokenstore` and updating `GetTokensHandler`.
  - Implemented token rotation on refresh: `RefreshTokenHandler` now revokes the old session token.
- **Bootstrap**: Removed initialization of the `Tokens` map when creating the default user.
- **Authorization**:
  - Corrected permission check in `GET /api/v1/users/token` (when ID is provided) from `read:self` to `read:item` to match defined user permissions.
  - Fixed context key mismatch for permissions and user ID between `AuthMiddleware` and permission checking functions (`CheckPermission`, `CheckReadPermission`).
  - Fixed context key mismatch for permissions, user ID, and audience in `RefreshTokenHandler`.
- **Authentication**: `AuthMiddleware` now checks token revocation status using `tokenstore`.
- **Configuration**: Changed `tokenstore` database path to `configs/tokens.db`.
- **Time**:
  - Standardized internal timestamps (`CreatedAt`, `LastLogin`, `ExpiresAt` in models and tokenstore) to use UTC.
  - Standardized JSON serialization/deserialization of timestamps to use RFC3339 format (second precision, e.g., `2006-01-02T15:04:05Z`) via a custom `models.RFC3339Time` type. Ensures consistency between `users.json` and logs.

### Fixed

- Fixed various Linter and Build errors related to package renames (`apitoken` -> `api_token`), unused variables/constants (`usersFilePath`, `log`), function signatures (`api_token` handlers), undefined context keys (`ContextKeyUserID`), and incorrect type usage (`models.RFC3339Time`).

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
- Basic routing setup in `internal/routes/routes.go`.
- Added configuration loading using TOML (`internal/config/config.go`, `configs/config.toml`).
- Implemented basic bootstrap logic (`internal/bootstrap/bootstrap.go`) to create necessary directories (`configs`, `logs`) and initialize default `config.toml` and `users.json`.
- Defined basic User model (`internal/models/user.go`).
- Initial commit.

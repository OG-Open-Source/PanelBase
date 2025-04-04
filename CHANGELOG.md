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
    - Added `last_login` timestamp field to user data, updated upon successful login.
- **API Structure**:
    - Consolidated API token management routes under `/api/v1/users/token` (POST, PUT, DELETE). Removed the standalone `/api/v1/token` group.
    - Removed the `GET /api/v1/users/token` route.
- **Authentication & Authorization**:
    - Refactored `AuthMiddleware` to dynamically select JWT validation secret based on token audience (`web` vs `api`).
    - Implemented permission checking logic in `internal/middleware/permissions.go` (specifically `CheckPermission`) based on scopes defined in `users.json` (exact match, no wildcards initially).
    - Applied permission checking middleware (`RequirePermission`) to the `POST /api/v1/users/token` route, requiring `api:create` permission.
    - Unified JWT structure for both login tokens and API tokens (aud, sub, name, jti, scopes, iss, iat, exp).
    - Corrected JWT `sub` claim to consistently use User ID (`usr_...`).
    - Corrected audience check in `RefreshTokenHandler` to use `web`.

### Fixed

- Fixed linter error caused by redeclaration of `UsersConfig` struct in the `models` package by removing `internal/models/users_config.go`.
- Fixed various issues during refactoring, including function signatures, data structure mismatches, import errors, unused variables, and incorrect field access (e.g., `user.IsActive` vs `user.Active`).
- Resolved `User information missing from context` error in `RefreshTokenHandler` by correcting context keys set by `AuthMiddleware`.
- Resolved `Authenticated user not found` error in `CreateTokenHandler` by using `GetUserByID` instead of `GetUserByUsername`.

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
- Created basic Gin server structure in `
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- (Future changes will go here)

## [0.6.0] - 2025-04-06 UI Settings and Template Rendering

### Added

- **UI Settings**: Implemented backend service (`internal/uisettings`) to manage global UI settings (Title, LogoURL, FaviconURL, CustomCSS, CustomJS) stored in `configs/ui_settings.json`.
- **API**: Added new API endpoints for UI settings under `/api/v1/settings`:
  - `GET /ui`: Retrieves current UI settings (requires `settings:read` permission).
  - `PUT /ui`: Updates UI settings, accepting partial updates (requires `settings:update` permission).
- **Permissions**: Added `settings:read` and `settings:update` permissions to the default `admin` user created during bootstrap (`internal/bootstrap/bootstrap.go`).
- **Documentation**: Created `DEVELOP_UI.md` detailing the available UI settings and their usage in templates.
- **Documentation**: Updated `COMMANDS.md` to include documentation for the new `/api/v1/settings/ui` endpoints.

### Changed

- **Templating**: Modified the server routing (`internal/routes/routes.go`) to render all `.html` and `.htm` files within the `web/` directory using Go's `html/template` engine. UI settings are automatically injected into these templates.
- **Static Files**: Refactored static file serving. Files under `web/assets/` are served efficiently via `router.Static`. Other non-HTML files under `web/` are served directly by the `NoRoute` handler.
- **Templating**: Ensured CustomCSS and CustomJS from UI settings are treated as safe content (`template.CSS`, `template.JS`) during HTML rendering to prevent incorrect escaping.

### Fixed

- **HTML**: Corrected a syntax error (missing `>`) in `web/index.html` that caused template parsing errors.
- **Logging**: Fixed an issue where requests served by the HTML template renderer (`serveHTMLTemplate`) might incorrectly log a 404 status code. Status is now explicitly set to 200 OK.
- Removed reference to non-existent `main.js` from `web/index.html`.

## [0.5.0] - 2025-04-05 Logger Major Updates and Changes

### Changed

- **API**: The `GET /api/v1/users/token` list endpoint now correctly returns only API tokens associated with the requesting user, filtering out web session tokens based on the `Audience` field in the token store (`api:read:list` permission still required).
- **Logging**: Updates to the custom logger format (`internal/middleware/logger.go`):
  - Status code color formatting based on range (2xx green, 3xx cyan, 4xx yellow, 5xx red).
  - Request body (if present and readable) is logged on a second line, prefixed with `[REQ BODY]`, for relevant methods (POST, PUT, PATCH, DELETE).
  - Log prefix determination (`[GIN]`, `[PANELBASE]`) based on route path (`/api/`).
  - Standardized timestamps to UTC RFC3339 format.
  - Slightly adjusted spacing and formatting for better readability.
- **Logging**: Removed redundant logging of detailed token parsing errors (`Token parsing error: ...`) from the authentication middleware (`internal/middleware/auth.go`). The main log line already indicates an invalid/expired token.

## [0.4.0] - 2025-04-04 Token Storage and Time Standardization

### Changed

- **Token Management**:
  - Integrated `tokenstore` (using BoltDB at `configs/tokens.db`) for persistent storage of API and session token metadata.
  - Removed `Tokens map[string]APIToken` from `UserAPISettings` in `users.json`.
  - Updated token service functions (`CreateAPIToken`, `DeleteTokenHandler`, `GetTokensHandler`, `UpdateTokenHandler`) and `LoginHandler` to use `tokenstore` instead of modifying `users.json` directly for token operations.
  - Implemented listing of user's API tokens in `GET /api/v1/users/token` using `tokenstore`.
  - Implemented session token rotation on refresh in `RefreshTokenHandler`.
- **Bootstrap**: Removed initialization of the `Tokens` map.
- **Authorization**: Corrected permission check in `GET /api/v1/users/token` to `read:item`. Fixed context key mismatches for permissions, user ID, and audience.
- **Authentication**: `AuthMiddleware` now checks token revocation status via `tokenstore`.
- **Time**: Standardized internal timestamps and JSON serialization to UTC RFC3339 via `models.RFC3339Time`.

### Fixed

- Fixed various Linter/Build errors related to package renames, unused variables, function signatures, undefined context keys, and incorrect type usage.

## [0.3.0] - 2025-04-03 Basic Authentication and Routing

### Added

- Basic `/api/v1/auth/login`, `/api/v1/auth/token` (refresh) handlers.
- `/logs` directory creation and basic file logging.
- Graceful server shutdown.
- `/api/v1/users/token` placeholder routes.
- Basic permission checking framework (`internal/middleware/permissions.go`).

### Changed

- **Authentication**: Enhanced `AuthMiddleware` (JWT from Cookie/Header, audience check). Updated JWT claims. Moved refresh token route under auth. Changed permission structure to `map[string][]string`.
- **API Structure**: Consolidated user API token management under `/api/v1/users/token`.
- **Models**: Updated `User` model for `api: {tokens: [...]}` JSON.
- **Server**: Standardized API response format.
- **Configuration**: Set default server mode to "release".
- **Code**: Translated comments, removed verbose logging.

### Fixed

- Fixed various syntax errors, undefined handler errors, JSON unmarshal errors, Gin mode logic, and audience format errors.

## [0.2.0] - 2025-04-02 API Route Structure Adjustment

### Changed

- **API**: Consolidated resource endpoints to use base paths (e.g., `/api/v1/commands`) for all methods (GET, POST, PUT, DELETE).
- **Routing**: Updated route definitions to reflect the consolidated API structure.

## [0.1.0] - 2025-04-01 Project Initialization

### Added

- Initialized Go project.
- Added Gin framework.
- Basic server and routing structure.
- Configuration loading (TOML).
- Bootstrap logic (directories, default config/users.json).
- Basic User model.
- Initial commit.

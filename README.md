# PanelBase

A Go-based web panel.

## Setup

Project initialization (directory/file creation) is now handled automatically when starting the server.

## Configuration Notes

- **Trusted Proxies:** If PanelBase runs behind a reverse proxy (like Nginx, Caddy, Cloudflare), it's **crucial** to configure `server.trusted_proxy` in `configs/config.toml` with the proxy's IP address or CIDR block. This allows logging to see the _real_ client IP address instead of the proxy's IP. Example: `trusted_proxy = "192.168.1.1"` or `trusted_proxy = "10.0.0.0/8"`. If not running behind a proxy, leave this empty.

## API Response Format

All API v1 endpoints (except for successful DELETE requests which return 204 No Content) adhere to a standard JSON response format:

```json
{
  "status": "success | failure",
  "message": "Descriptive message about the outcome.",
  "data": { ... } // Optional: Contains the actual response data (object or array)
}
```

- **status**: Indicates the overall outcome.
  - `"success"`: The request was successful (typically HTTP 2xx).
  - `"failure"`: The request failed due to client-side issues (e.g., validation, permissions, not found - typically HTTP 4xx) or server-side issues (typically HTTP 5xx).
- **message**: A user-friendly string describing the result.
- **data**: An optional field containing the response payload. For list endpoints, this will be an array. For single resource endpoints, it will be an object. It's omitted or null for errors or operations that don't return data.

## Run

Run `go run cmd/server/main.go` to start the server.

## Changelog

- Initial project setup: Initialized Go module, created directory structure, and added PROJECT_STATUS.md.
- Added basic Gin server setup in `cmd/server/main.go`.
- Created initial v1 API route structure in `internal/api/v1/routes.go`.
- Registered v1 API routes in `main.go` under the `/api/v1` group.
- Implemented custom Gin logger format in `main.go`.
- Refactored and corrected custom logger format into `pkg/logger/formatter.go`.
- Implemented automatic creation of timestamped log files in `/logs` directory, outputting logs to both console and file.
- Added initialization script `cmd/init/main.go` to create project structure and default files.
- Attempted to fix console color loss when using multi-writer logger.
- Refactored log file creation logic into `pkg/logger/logger.go`.
- Refactored `cmd/init/main.go` to create minimal mandatory directories (`configs`, `themes`) and conditionally create others (`plugins`, `commands`, `users.json`) based on `configs/config.toml`.
- Integrated project initialization logic into server startup (`cmd/server/main.go` now calls `pkg/bootstrap.InitializeProject`).
- Deprecated `cmd/init/main.go` as a standalone command.
- Ensured `/web` directory is also created during initialization.
- Modified initialization to dynamically generate an available port and random entry string in `configs/config.toml` on first creation.
- Server now reads IP, port, and entry from `configs/config.toml` at startup.
- Removed unused `ensureFile` function from `pkg/bootstrap/init.go`.
- Initialization now creates an entry-specific directory under `/web` (i.e., `/web/<entry>`).
- Server now serves static files from `/web/<entry>` at the `/entry` URL path.
- Implemented custom static file handling for the entry path: Direct `.html`/`.htm` access is denied; files are accessible via clean URLs (filename without extension).
- Refactored static file handling to use a shared handler function (`createStaticHandler`).
- If `server.entry` is empty in `config.toml`, the server now serves the `/web` directory at the root URL (`/`) using the same custom rules.
- Refactored static file serving to use `NoRoute` handler for root path when `server.entry` is empty, resolving route conflicts.
- Initialization process now ensures `/configs/ui_settings.json` exists, creating it with default values if necessary.
- Server now loads UI settings from `/configs/ui_settings.json` at startup.
- HTML/HTM files served via the custom handler are now rendered using Go's template engine, injecting UI settings data.
- Changed template loading from `LoadHTMLGlob` to manual file walking (`LoadHTMLFiles`) to support both `.html` and `.htm` without panic.
- Initialization process now automatically creates a default `index.html` in the target web directory (`web/` or `web/<entry>/`) if neither `index.html` nor `index.htm` exists.
- Added `server.mode` option to `configs/config.toml` (defaulting to "debug" on creation).
- Gin run mode is now set based on `server.mode` configuration before router initialization.
- Changed default Gin mode to `release`. Bootstrap now writes `mode = "release"` on first config creation, and server defaults to release mode if config is missing or invalid.
- Reduced verbosity of initialization and template loading logs.
- Combined server startup information (Mode, Address, Admin Entry) into a single log line.
- Added JWT authentication and permission-based authorization:
  - Added `/api/v1/auth/register` and `/api/v1/auth/login` endpoints.
  - Implemented user storage using `configs/users.json` (for development).
  - Passwords are hashed using bcrypt.
  - Login returns a JWT containing user ID, username, and permissions.
  - Added `/api/v1/protected` example routes secured by JWT middleware.
  - Middleware checks for required permissions (e.g., `/protected/admin/users` requires "admin" permission).
  - Configured JWT secret and duration via `configs/config.toml`.
- Moved `users.json` creation logic from storage layer to bootstrap initialization.
- Removed verbose "Creating..." log messages during bootstrap initialization.
- Updated user model (`models.User`) and storage (`users.json` format, `JSONUserStore`) to match example structure (including Name, Email, Active, Scopes).
- Updated JWT claims and authorization middleware to use Scopes map instead of simple permissions list.
- Removed example `/protected` API routes.
- Added initial API tests for auth endpoints (`/test/auth_api_test.go`) using testify.
- Added `functions.themes` boolean flag to `configs/config.toml`.
- Made creation of `/themes`, `/plugins`, `/commands` directories conditional based on `functions.*` flags in `config.toml`.
- Registration endpoint (`/api/v1/auth/register`) is now conditionally enabled based on `functions.users` flag in `config.toml`.
- Implemented custom error handling for HTTP status codes (4xx, 5xx):
  - Priority 1: Serves `/web/<entry>/templates/<status_code>.html` (or `.htm`) if it exists, rendered with `uiSettingsData`.
  - Priority 2: Serves `/web/<entry>/error.html` (or `.htm`) if it exists, rendered with `http_status_code`, `http_status_message`, and `uiSettingsData`.
  - Priority 3: Returns PanelBase's default plain text message (`<code>: <Reason-Phrase>`) as fallback.
  - Integrated into `NoRoute` (404) and `NoMethod` (405) handlers, and also called directly by the static file handler on failure. API errors still return JSON.
- Refactored web serving logic (static files, templates, error pages) into a dedicated `internal/webserver` package:
  - Moved `handleStaticFileRequest`, `loadTemplates`, `handleErrorResponse` functions.
  - Created `webserver.RegisterHandlers` to encapsulate setup.
  - Updated `cmd/server/main.go` to call `webserver.RegisterHandlers`.
- Simplified server startup logs by removing verbose messages from `internal/webserver` package (template scanning, route registration details).
- Fixed error page handling for entry-specific paths (`/<entry>/...`) by having `handleStaticFileRequest` directly call `handleErrorResponse` when a file is not found.
- Further simplified logs by removing messages about which specific error template is being served.
- Re-created API tests for auth endpoints (`/test/auth_api_test.go`) covering registration and login scenarios (success, conflict, missing fields, incorrect credentials, registration disabled).
- Added interactive PowerShell API test script (`/test/api_test.ps1`) with menu for server control, config initialization, and running auth tests.
- Added `/debug` API endpoint and port override (32768) when `server.mode` is "debug".
- Fixed route conflict between root static file handler and entry-specific handler when `server.entry` is set.
- Implemented dynamic HTML template rendering (on-demand parsing) instead of pre-loading, allowing runtime addition of HTML files.
- Prevented direct URL access to files within the `/web/<entry>/templates` directory.
- Removed INFO log message when a requested file/template is not found.
- Added ID generator utility (`pkg/utils/id_generator.go`) for creating prefixed random IDs.
- Updated user creation (`JSONUserStore`) to use the new ID generator utility.
- Ensured `created_at` timestamps in `configs/users.json` are saved in RFC3339 format (without nanoseconds).
- Implemented graceful shutdown handling for the server.
- Removed IP-based rate limiting middleware.
- Standardized API JSON response format (`status`, `message`, `data`).
- Implemented full User Management API (`/api/v1/users`) with CRUD operations and fine-grained scope checks (`users:read`, `users:create`, `users:delete`, `users:update:name`, `users:update:email`, `users:update:active`, `users:update:scopes`, `users:update:api_tokens`).
- Implemented Account Management API (`/api/v1/account`) for self-service: profile updates (`account:profile:read`, `account:update:name`, `account:update:email`), password change (`account:password:update`), self-deletion (`account:self_delete:execute`), and API Token management (`account:tokens:create`, `account:tokens:read`, `account:tokens:delete`).
- Made authentication rules configurable via `configs/config.toml`: requiring old password for update, allowing self-deletion, and protecting specific User IDs from deletion.
- Added default scope assignment for new users based on `configs/config.toml`.
- Bootstrap process now adds the initial admin User ID to the protected list in `configs/config.toml`.

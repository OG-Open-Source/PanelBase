# Project Status: PanelBase

## Objective

To build a web panel application using the Go programming language and the Gin web framework, hosted at github.com/OG-Open-Source/PanelBase.

## Architecture Overview

- **Language:** Go
- **Web Framework:** Gin
- **Directory Structure:**
  - `/cmd`: Main application entry points (`cmd/server/main.go`).
  - `/configs`: Configuration files (e.g., `config.yaml`).
  - `/internal`: Private application code, strictly internal to this project.
    - `/internal/api`: Houses versioned API logic.
      - `/internal/api/v1`: Implementation for API version 1 (handlers, routes, middleware). Future versions (v2, etc.) will reside in parallel directories. API versions are designed to be independent.
    - `/internal/auth`: Authentication and authorization logic.
    - `/internal/database`: Database interaction layer.
    - `/internal/models`: Data structure definitions.
    - `/internal/services`: Core business logic implementation.
  - `/pkg`: Shared libraries reusable within this project or potentially externally (logger, errors, utils).
  - `/web`: Frontend assets, including static files (`/web/static`) and HTML templates (`/web/templates`). HTML Templates are now rendered dynamically on request.
- **API Versioning:** Implemented via URL path (`/api/v1`, `/api/v2`, ...). Routes are defined within their respective version directories (`internal/api/vX/routes.go`).
- **Naming Convention:** Directory and file names use underscores (`_`) as separators (e.g., `api_token`).

## Core Components (Planned)

- **Gin Router:** Initialized in `cmd/server/main.go`. Registers global middleware and versioned API route groups.
- **Configuration Management:** Load application settings from files in `/configs`.
- **API Handlers:** Located in `internal/api/vX/handlers/`. Contain logic specific to handling HTTP requests and responses for a given API version.
- **API Routes:** Defined in `internal/api/vX/routes.go`. Map URL paths to specific handlers for each API version.
- **Middleware:** Can be global (in `pkg` or `cmd`) or version-specific (`internal/api/vX/middleware/`).
- **Services:** Encapsulate business logic, located in `internal/services/`. Called by API handlers.
- **Database Layer:** Abstract database operations, located in `internal/database/`.
- **Models:** Define Go structs representing data entities, located in `internal/models/`.
- **Logging:** Centralized logging mechanism (`pkg/logger`).
- **Error Handling:** Consistent error handling strategy (`pkg/errors`).
- **Utilities:** Common helper functions (`pkg/utils`).

## TODO List / Initial Setup Tasks

- [x] Initialize Go module: `go mod init github.com/OG-Open-Source/PanelBase`
- [x] Create initial directory structure: `cmd/server`, `configs`, `internal/api/v1/handlers`, `internal/api/v1/middleware`, `internal/auth`, `internal/database`, `internal/models`, `internal/services`, `pkg/logger`, `pkg/errors`, `pkg/utils`, `web/static`, `web/templates`
- [x] Create basic `cmd/server/main.go` with Gin setup.
- [x] Create basic `internal/api/v1/routes.go` to define the v1 API group.
- [x] Create `PROJECT_STATUS.md` (this file).
- [x] Create initial `README.md`.
- [x] Add Gin dependency: `go get github.com/gin-gonic/gin`
- [x] Implement custom Gin logger format in `pkg/logger`.
- [x] Implement automatic log file creation in `/logs` with timestamped filename (handled by `pkg/logger`).
- [x] Refactor initialization script `cmd/init/main.go` to handle mandatory dirs (`configs`, `themes`, `web`) and conditional creation based on `configs/config.toml` (`plugins`, `commands`, `users.json`, `themes`).
- [x] Integrate project initialization into server startup (`cmd/server/main.go` calls `pkg/bootstrap`).
- [x] Dynamically generate available port (1024-49151) and random entry string in `config.toml` on first creation.
- [x] Read server IP/port/entry from `config.toml` at server startup.
- [x] Create entry-specific web directory (`/web/<entry>`) during initialization.
- [x] Implement custom static file serving using explicit routes and `NoRoute`: Deny direct `.html`/`.htm` access, allow access via clean URLs. Serve `/web/<entry>` at `/<entry>/` if entry exists, otherwise serve `/web` at `/` via `NoRoute`.
- [x] Ensure `/configs/ui_settings.json` is created with default content during initialization.
- [x] Load UI settings from `/configs/ui_settings.json`.
- [x] Render `.html`/`.htm` files as Go templates, passing UI settings data (using manual template loading).
- [x] Ensure a default `index.html` is created in the target web directory if none exists.
- [x] Set Gin mode (defaulting to release) based on `server.mode` in `configs/config.toml`.
- [x] Add dependencies for auth (bcrypt, jwt, uuid).
- [x] Add JWT config (`jwt_secret`, `token_duration_minutes`) to `config.toml` (bootstrap & server).
- [x] Define User model (`internal/models/user.go`) (Updated with Name, Email, Active, Scopes).
- [x] Define UserStore interface (`internal/storage/user_store.go`).
- [x] Implement JSONUserStore (`internal/storage/json_user_store.go`) (Updated for new file format and model).
- [x] Implement JWT generation/validation (`internal/auth/jwt.go`) (Updated for Scopes).
- [x] Implement auth handlers (Register, Login) (`internal/api/v1/handlers/auth_handler.go`) (Updated for new model/scopes).
- [x] Implement auth middleware (JWT validation, scope check) (`internal/api/v1/middleware/auth_middleware.go`) (Updated for Scopes).
- [x] Register auth routes and remove protected example routes in API v1 (`internal/api/v1/routes.go`).
- [x] Integrate UserStore and AuthHandler into server startup (`cmd/server/main.go`).
- [x] Ensure `users.json` creation is handled by bootstrap initialization.
- [x] Remove verbose "Creating..." logs during initialization.
- [x] Add initial auth API tests (`/test/auth_api_test.go`).
- [x] Add `functions.themes` boolean flag to `config.toml`.
- [x] Enhance debug mode: Force port 32768 and add `/debug` API endpoint.
- [x] Fix routing conflict between root static file handler and entry-specific handler.
- [x] Implement dynamic HTML template rendering instead of pre-loading.
- [x] Prevent direct access to `/web/<entry>/templates/` via URL.
- [x] Adjust logging level for file not found errors.
- [x] Add configuration for trusted proxy.
- [x] Add ID generator utility (`pkg/utils/id_generator.go`).
- [x] Update user creation (`JSONUserStore`) to use ID generator.
- [x] Ensure `created_at` in `users.json` uses RFC3339 format.
- [x] Implement hierarchical scope-based authorization middleware (`map[string]interface{}`).
- [x] Update JWT generation/validation for hierarchical scopes and `aud`/`jti`/`name` claims.
- [x] Apply fine-grained scope checks to user management routes (`users:*`).
- [x] Implement initial admin user creation with full scopes in bootstrap.
- [x] Implement default scope assignment for newly created users in CreateUser handler.
- [x] Implement `/account` routes for self-management (profile, password).
- [x] Design and implement basic API Token management (create, list, delete for self; stored in users.json).
- [x] Implement graceful shutdown handling.
- [x] Implement configurable auth rules (require old password, allow self-delete, prevent admin delete via User ID) in `config.toml`.
- [x] Implement default scope assignment based on `config.toml` for new users.
- [ ] Implement JTI validation for API tokens (requires DB or store modification).
- [ ] Implement routes and scope checks for `themes`, `plugins`, `commands`.
- [ ] Write comprehensive API tests for authorization scenarios.

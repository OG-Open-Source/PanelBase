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
  - `/web`: Frontend assets, including static files (`/web/static`) and HTML templates (`/web/templates`).
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
- [x] Refactor initialization script `cmd/init/main.go` to handle mandatory dirs (`configs`, `themes`, `web`) and conditional creation based on `configs/config.toml` (`plugins`, `commands`, `users.json`).
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

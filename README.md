# PanelBase

A Go-based web panel.

## Setup

Project initialization (directory/file creation) is now handled automatically when starting the server.

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

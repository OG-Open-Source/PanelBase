# PanelBase

A flexible and extensible panel system built with Go.

## Project Goals

*   Load custom panels defined in local files.
*   Provide a versioned API for interaction (e.g., `/api/v1`).
*   Utilize Server-Sent Events (SSE) for real-time updates.
*   Support content personalization using Go templates (`text/template`).

## Directory Structure

```
PanelBase/
├── cmd/panelbase/       # Main application entry point (placeholder)
├── frontend/            # Static frontend assets (HTML, CSS, JS - placeholder)
├── internal/
│   ├── api/             # API version handlers
│   │   └── v1/          # Version 1 API implementation (router, handlers, SSE, templates)
│   ├── panel/           # Panel definition, loading logic, and interface
│   └── server/          # HTTP server setup and routing
├── go.mod               # Go module definition
├── go.sum               # Go module checksums (will be generated)
├── main.go              # Application entry point (will call server start)
└── README.md            # This file
```

## Setup

1.  **Install Go:** Ensure you have Go 1.23 or later installed.
2.  **Clone Repository:** `git clone https://github.com/OG-Open-Source/PanelBase.git` (Replace with actual URL if different)
3.  **Navigate to Directory:** `cd PanelBase`
4.  **Install Dependencies:** `go mod tidy` (This will download necessary modules once code uses them)

## Build

Build the application executable:

```bash
# For Linux/macOS
go build -o panelbase ./cmd/panelbase

# For Windows
go build -o panelbase.exe ./cmd/panelbase
```

*(Note: The `cmd/panelbase` directory currently only contains a `.gitkeep` file. The build command assumes `main.go` will eventually be moved or logic placed within `cmd/panelbase/main.go` for a typical Go project structure. For now, you can build from the root using `go build -o panelbase main.go`)*

## Features

*   **Web Server:** Provides a basic HTTP server.
*   **API v1:** Basic API structure under `/api/v1`.
*   **Plugin System (Basic):**
    *   Loads plugins from the `./ext/plugins` directory at startup (currently placeholder logic).
    *   Defines a `Plugin` interface in `internal/plugin/plugin.go`.
    *   Includes a basic plugin loader in `internal/plugin/loader.go`.

## Running

Start the PanelBase server (after building):

```bash
# For Linux/macOS
./panelbase

# For Windows
.\panelbase.exe
```

*(Note: The server currently starts and attempts to load plugins from `./ext/plugins`, but the actual plugin loading logic is not yet implemented.)*
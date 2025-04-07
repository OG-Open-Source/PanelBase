# PanelBase

A lightweight Go-based management panel for Linux servers.

## Features

_(Add key features here as they are developed)_

- JWT-based Authentication (Web session & API tokens)
- API Token Management (Create, List, Update, Delete)
- Basic Permission System
- Persistent Token Storage (BoltDB)
- Customizable Logging

## Version

Current version: 0.9.0 (2025-04-08)

## Getting Started

_(Add instructions on how to build and run the project)_

```bash
# Example
go run cmd/panelbase/main.go
```

## Configuration

Configuration is managed via `configs/config.toml` and `configs/users.json`.

## Changelog

### [0.9.0] - 2025-04-08 Logging Overhaul and Refinements

- Centralized and structured logging (`logger.Printf`, etc. with module/action).
- Log output detail now depends on `server.mode` (`debug` vs `release`).
- Standardized log timestamps to RFC3339.
- Fixed startup order issue (`bootstrap` now runs before config load).
- Centralized middleware context keys.

See [CHANGELOG.md](CHANGELOG.md) for detailed history.

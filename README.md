# PanelBase

A lightweight, secure, and cross-platform panel system built with Go.

## Features

- Cross-platform support (Windows, Linux, macOS)
- Secure access with custom entry point
- JWT-based authentication
- Comprehensive logging system
- No database dependency
- Built with Echo framework

## Project Structure

```
.
├── cmd/
│   └── panelbase/
│       └── main.go
├── configs/
│   └── config.toml
├── internal/
│   ├── config/
│   │   └── config.go
│   └── logger/
│       └── logger.go
├── web/
│   └── static/
├── logs/
├── go.mod
└── README.md
```

## Configuration

Edit `configs/config.toml` to configure:

- Server settings (host, port, entry point)
- Security settings (JWT secret, expiration)
- User credentials
- Logging configuration

## Building

```bash
go build -o panelbase cmd/panelbase/main.go
```

## Running

```bash
./panelbase
```

## Access

The panel will be accessible at:
```
http://<IP>:<PORT>/<ENTRY>
```

## Security

- All API endpoints are protected by JWT authentication
- Custom entry point for additional security
- No database dependency to minimize attack surface
- Comprehensive logging for security monitoring

## License

MIT License 
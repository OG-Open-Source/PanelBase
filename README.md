# PanelBase

PanelBase is a secure, lightweight control panel that runs on Linux, Windows, and macOS. It provides a web interface and API for managing tasks with multi-user support and role-based permissions.

## Features

- **Secure Access**: Accessible only through a specific entry point (`IP:PORT/ENTRY`)
- **Cross-Platform**: Runs on Linux, Windows, and macOS
- **Comprehensive Logging**: Detailed log storage for all operations
- **Multi-User System**: Role-based permissions with 4 user levels:
  - ROOT: Complete system control
  - ADMIN: Manage tasks but limited user management
  - USER: Create and run tasks
  - GUEST: View-only access
- **Task Management**: Create, run, monitor, and stop tasks
- **API Access**: Full API support for programmatic access
- **No Database Required**: Uses file-based storage for simplicity and portability
- **Lightweight**: Minimal dependencies and small footprint

## Getting Started

### Prerequisites

- Go 1.19 or later

### Installation

1. Clone the repository:

```bash
git clone https://github.com/OG-Open-Source/PanelBase.git
cd PanelBase
```

2. Configure the application by editing the `.env` file:

```
IP=0.0.0.0              # IP to bind to (0.0.0.0 for all interfaces)
PORT=8080               # Port to listen on
ENTRY=panel             # Custom entry point for added security
JWT_SECRET=your_secret  # Secret for JWT token generation (change this!)
WORK_DIR=/              # Default working directory for tasks
PANEL_TRUSTED_IPS=      # Comma-separated list of trusted IPs (empty for all)
```

3. Build the application:

```bash
go build -o panelbase ./cmd/panelbase
```

4. Run the application:

```bash
./panelbase
```

### Default Account

Upon first run, a default admin account is created:

- Username: `admin`
- Password: `admin`

**IMPORTANT**: Change the default password immediately after first login for security.

## Directory Structure

The PanelBase project follows a standard Go project layout:

```
.
├── cmd/                  # Application entry points
│   └── panelbase/        # Main application
│       └── main.go       # Main entry point
├── internal/             # Private application code
│   ├── config/           # Configuration management
│   ├── executor/         # Task execution system
│   ├── logger/           # Logging system
│   ├── middleware/       # HTTP middleware
│   ├── server/           # HTTP server implementation
│   │   ├── api/          # API handlers
│   │   └── ...
│   ├── user/             # User management system
│   └── utils/            # Utility functions
├── web/                  # Web frontend
│   ├── index.html        # Main HTML file
│   └── static/           # Static assets
│       ├── css/          # Stylesheets
│       ├── js/           # JavaScript files
│       └── img/          # Images and icons
├── data/                 # Application data (created at runtime)
│   └── users.json        # User data storage
├── logs/                 # Log files (created at runtime)
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
├── .env                  # Environment configuration
└── README.md             # This file
```

## Deployment

### Running as a Service

#### Linux (systemd)

1. Create a systemd service file:

```bash
sudo nano /etc/systemd/system/panelbase.service
```

2. Add the following content:

```
[Unit]
Description=PanelBase Control Panel
After=network.target

[Service]
User=your_user
WorkingDirectory=/path/to/panelbase
ExecStart=/path/to/panelbase/panelbase
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

3. Enable and start the service:

```bash
sudo systemctl enable panelbase
sudo systemctl start panelbase
```

#### Windows

1. Install [NSSM (Non-Sucking Service Manager)](https://nssm.cc/download)
2. Open Command Prompt as Administrator
3. Run:

```
nssm install PanelBase
```

4. Configure the service:

   - Path: `C:\path\to\panelbase.exe`
   - Startup directory: `C:\path\to\panelbase`
   - Service name: `PanelBase`

5. Start the service:

```
nssm start PanelBase
```

### Reverse Proxy with HTTPS

For production, it's recommended to setup a reverse proxy with HTTPS:

#### Example with Nginx

```
server {
		listen 443 ssl;
		server_name your-domain.com;

		ssl_certificate /path/to/cert.pem;
		ssl_certificate_key /path/to/key.pem;

		location /panel/ {
				proxy_pass http://localhost:8080/panel/;
				proxy_http_version 1.1;
				proxy_set_header Upgrade $http_upgrade;
				proxy_set_header Connection 'upgrade';
				proxy_set_header Host $host;
				proxy_cache_bypass $http_upgrade;
		}
}
```

## API Documentation

### Authentication

- Login: `POST /{ENTRY}/api/auth/login`
- All other API endpoints require JWT authentication via the `Authorization: Bearer {token}` header
- API key authentication is also supported via the `X-API-Key` header

### User Management

- List Users: `GET /{ENTRY}/api/users`
- Get User: `GET /{ENTRY}/api/users/{id}`
- Create User: `POST /{ENTRY}/api/users`
- Update User: `PUT /{ENTRY}/api/users/{id}`
- Delete User: `DELETE /{ENTRY}/api/users/{id}`
- Generate API Key: `POST /{ENTRY}/api/users/{id}/api-key`

### Task Management

- List Tasks: `GET /{ENTRY}/api/tasks`
- Get Task: `GET /{ENTRY}/api/tasks/{id}`
- Create Task: `POST /{ENTRY}/api/tasks`
- Start Task: `POST /{ENTRY}/api/tasks/{id}/start`
- Stop Task: `POST /{ENTRY}/api/tasks/{id}/stop`

## Security Considerations

- Always change the default admin password
- Use a strong, random JWT secret
- Configure trusted IPs to restrict access
- Use HTTPS in production environments
- Regularly rotate API keys

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

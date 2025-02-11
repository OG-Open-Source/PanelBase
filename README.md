# PanelBase - Secure Remote Panel Control System

A secure, token-based remote panel control system that allows remote access to Linux systems through a web interface without installing a full panel UI on the target host.

## Features

- Triple-key authentication system for enhanced security
- Token-based authorization with automatic rotation
- Temporary access sharing with customizable permissions
- CGI-based backend for lightweight deployment
- GitHub Pages hosted frontend
- Request rate limiting and source binding
- Temporary token sharing mechanism

## Architecture

### Frontend (GitHub Pages)
- Hosted at `panel.ogtt.tk`
- Implements UI for key input and theme selection
- Handles temporary access sharing UI
- Manages token refresh mechanism

### Backend (CGI)
- Lightweight CGI implementation
- Implements triple-key verification
- Handles token generation and rotation
- Manages rate limiting and security measures

## Security Features

### Authentication
- Triple parallel key mechanism
- Token-based session management
- Automatic token rotation via cron

### Access Control
- Custom permission levels
- Temporary access sharing
- Configurable user limits
- Source IP binding

### Security Measures
- Rate limiting on CGI endpoint
- Token rotation mechanism
- Request source validation
- Temporary token mechanism

## Components

### CGI Backend
- `/cgi-bin/panel.cgi` - Main CGI endpoint
- `/cgi-bin/token.cgi` - Token management endpoint
- `/cgi-bin/share.cgi` - Sharing management endpoint

### Frontend
- Single page application
- JWT-based permission encoding
- Theme support
- Command interface

## Setup

### Target Host Requirements
1. CGI-capable web server
2. Cron service for token rotation
3. Basic system utilities

### Installation
[To be added]

## Security Considerations
- All communication is currently over HTTP
- IP-based security measures
- Token rotation for enhanced security
- Rate limiting implementation

## Development Status
Under active development

## License
See LICENSE file for details

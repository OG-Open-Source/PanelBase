# PanelBase API Reference

This document lists the currently available API endpoints for PanelBase, along with example PowerShell requests using `Invoke-RestMethod`. Replace `$PORT` and `$ENTRY` with your actual server port and entry path as needed.

---

## Variables

- `$PORT` : The port your PanelBase server is running on (e.g., `8080`)
- `$ENTRY` : The entry path if configured (e.g., `admin`). If not set, leave it empty in the URL.

---

## Authentication

### Register

- **Endpoint:** `POST /$ENTRY/api/v1/auth/register`
- **Description:** Register a new user (if registration is enabled).
- **Example:**

```powershell
$body = @{ username = "newuser1"; password = "yourpassword"; name = "User Name"; email = "user@example.com" }
$response = Invoke-RestMethod -Uri "http://localhost:$PORT/$ENTRY/api/v1/auth/register" -Method Post -Body ($body | ConvertTo-Json) -ContentType 'application/json'
$token = $response.data.token
```

### Login

- **Endpoint:** `POST /$ENTRY/api/v1/auth/login`
- **Description:** Authenticate and receive a JWT token.
- **Example:**

```powershell
$body = @{ username = "newuser"; password = "yourpassword" }
$response = Invoke-RestMethod -Uri "http://localhost:$PORT/$ENTRY/api/v1/auth/login" -Method Post -Body ($body | ConvertTo-Json) -ContentType 'application/json'
$token = $response.data.token
```

### Admin Login

- **Endpoint:** `POST /$ENTRY/api/v1/auth/login`
- **Description:** Login as an administrator to obtain a JWT token for admin operations.
- **Example:**

```powershell
$body = @{ username = "admin"; password = "fXOp4DEB2qCe0ghj" }
$response = Invoke-RestMethod -Uri "http://localhost:$PORT/$ENTRY/api/v1/auth/login" -Method Post -Body ($body | ConvertTo-Json) -ContentType 'application/json'
$token = $response.data.token
```

---

## User Management

> Requires admin or appropriate scopes.

### List Users

- **Endpoint:** `GET /$ENTRY/api/v1/users`
- **Description:** Get a list of all users.
- **Example:**

```powershell
$headers = @{ Authorization = "Bearer $token" }
Invoke-RestMethod -Uri "http://localhost:$PORT/$ENTRY/api/v1/users" -Headers $headers -Method Get
```

### Create User

- **Endpoint:** `POST /$ENTRY/api/v1/users`
- **Description:** Create a new user.
- **Example:**

```powershell
$headers = @{ Authorization = "Bearer $token" }
$body = @{ username = "anotheruser"; password = "pass1234"; name = "Another User"; email = "another@example.com" }
Invoke-RestMethod -Uri "http://localhost:$PORT/$ENTRY/api/v1/users" -Headers $headers -Method Post -Body ($body | ConvertTo-Json) -ContentType 'application/json'
```

### Get User by ID

- **Endpoint:** `GET /$ENTRY/api/v1/users/{id}`
- **Description:** Get details for a specific user.
- **Example:**

```powershell
$headers = @{ Authorization = "Bearer $token" }
$userId = "user_xxxxx"
Invoke-RestMethod -Uri "http://localhost:$PORT/$ENTRY/api/v1/users/$userId" -Headers $headers -Method Get
```

### Update User

- **Endpoint:** `PUT /$ENTRY/api/v1/users/{id}`
- **Description:** Update user details.
- **Example:**

```powershell
$headers = @{ Authorization = "Bearer $token" }
$userId = "user_xxxxx"
$body = @{ name = "Updated Name" }
Invoke-RestMethod -Uri "http://localhost:$PORT/$ENTRY/api/v1/users/$userId" -Headers $headers -Method Put -Body ($body | ConvertTo-Json) -ContentType 'application/json'
```

### Delete User

- **Endpoint:** `DELETE /$ENTRY/api/v1/users/{id}`
- **Description:** Delete a user.
- **Example:**

```powershell
$headers = @{ Authorization = "Bearer $token" }
$userId = "user_xxxxx"
Invoke-RestMethod -Uri "http://localhost:$PORT/$ENTRY/api/v1/users/$userId" -Headers $headers -Method Delete
```

---

## Account Management (Self-Service)

### Get Profile

- **Endpoint:** `GET /$ENTRY/api/v1/account/profile`
- **Description:** Get your own account profile.
- **Example:**

```powershell
$headers = @{ Authorization = "Bearer $token" }
Invoke-RestMethod -Uri "http://localhost:$PORT/$ENTRY/api/v1/account/profile" -Headers $headers -Method Get
```

### Update Profile

- **Endpoint:** `PUT /$ENTRY/api/v1/account/profile`
- **Description:** Update your own profile (name, email).
- **Example:**

```powershell
$headers = @{ Authorization = "Bearer $token" }
$body = @{ name = "New Name"; email = "new@example.com" }
Invoke-RestMethod -Uri "http://localhost:$PORT/$ENTRY/api/v1/account/profile" -Headers $headers -Method Put -Body ($body | ConvertTo-Json) -ContentType 'application/json'
```

### Change Password

- **Endpoint:** `PUT /$ENTRY/api/v1/account/password`
- **Description:** Change your account password.
- **Example:**

```powershell
$headers = @{ Authorization = "Bearer $token" }
$body = @{ old_password = "oldpass"; new_password = "newpass" }
Invoke-RestMethod -Uri "http://localhost:$PORT/$ENTRY/api/v1/account/password" -Headers $headers -Method Put -Body ($body | ConvertTo-Json) -ContentType 'application/json'
```

### Delete Own Account

- **Endpoint:** `DELETE /$ENTRY/api/v1/account`
- **Description:** Delete your own account (if allowed).
- **Example:**

```powershell
$headers = @{ Authorization = "Bearer $token" }
Invoke-RestMethod -Uri "http://localhost:$PORT/$ENTRY/api/v1/account" -Headers $headers -Method Delete
```

---

## Debug (Debug Mode Only)

### Ping

- **Endpoint:** `GET /debug/ping`
- **Description:** Simple health check (only available in debug mode).
- **Example:**

```powershell
Invoke-RestMethod -Uri "http://localhost:$PORT/debug/ping" -Method Get
```

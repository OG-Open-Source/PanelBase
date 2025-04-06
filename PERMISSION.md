# Permissions

This document outlines the permission system used in PanelBase.
Permissions are defined in the `users.json` file for each user under the `scopes` field.
They follow a `resource:action[:modifier]` format.

## Permission Format

- **Resource**: The type of object the permission applies to (e.g., `account`, `users`, `api`, `settings`, `commands`, `plugins`, `themes`).
  - `account`: Refers to actions the user performs on their _own_ account data (profile, password, etc.).
  - `users`: Refers to administrative actions performed on _any_ user's account (listing, creating, updating, deleting other users).
- **Action**: The operation being permitted (e.g., `read`, `create`, `update`, `delete`, `install`, `execute`).
  - The `update` action is typically associated with specific HTTP methods depending on the resource:
    - **PATCH** is preferred for `account`, `users`, `api`, `settings` to allow partial updates.
    - **PUT** might be used for `commands`, `plugins`, `themes` if updates imply replacing the entire resource configuration.
- **Modifier (Optional)**: Further specifies the scope of the action (e.g., `list` for reading multiple items, `item` for a single item, `all` for administrative access over other users' resources).

## Default Admin Permissions

The default `admin` user created during the initial bootstrap has the following permissions:

```json
{
  "scopes": {
    "account": [
      "read", // Read own profile
      "update", // Update own profile (e.g., name, email, password via PATCH)
      "delete" // Delete own account
    ],
    "users": [
      "read:list", // List all users
      "read:item", // Get specific user details
      "create", // Create new users
      "update", // Update any user's details (excluding password changes via this scope)
      "delete" // Delete any user
    ],
    "api": [
      "read:list", // List own API tokens
      "read:item", // Get own specific API token
      "create", // Create own API token
      "update", // Update own API token
      "delete", // Delete own API token
      "read:list:all", // List ANY user's API tokens
      "read:item:all", // Get ANY user's specific API token
      "create:all", // Create API token for ANY user
      "update:all", // Update ANY user's API token
      "delete:all" // Delete ANY user's API token
    ],
    "settings": [
      "read", // Read global UI settings
      "update" // Update global UI settings
    ],
    "commands": [
      "read:list",
      "read:item",
      "install",
      "execute",
      "update",
      "delete"
    ],
    "plugins": ["read:list", "read:item", "install", "update", "delete"],
    "themes": ["read:list", "read:item", "install", "update", "delete"]
  }
}
```

## API Endpoint Permissions

The following API endpoints require specific permissions:

### Authentication (`/api/v1/auth`)

- `POST /login`: No permission required (public).
- `POST /register`: No permission required (public).
- `POST /token` (Refresh): Requires a valid `web_session` token (implicit permission).

### Account Management (Self - _Intended Route: `/api/v1/account`_)

_(Note: These routes/permissions are defined but corresponding handlers might not be fully implemented yet)_

- `GET /`: Requires `account:read` (View own profile).
- `PATCH /`: Requires `account:update` (Update own profile details like `{"name": "..."}` or `{"password": "..."}`).
- `DELETE /`: Requires `account:delete` (Delete own account).

### User Management (Admin - `/api/v1/users`)

- `GET /`: Permission checks (`users:read:list` or `users:read:item`) are **intended** within handler (Currently placeholder).
- `POST /`: Requires `users:create`.
- `PATCH /`: Requires `users:update` (Partially update user details; non-standard to use PATCH on collection endpoint - consider `PUT /users/{id}` or `PATCH /users/{id}`).
- `DELETE /users/{id}`: Requires `users:delete` (Delete specific user; route example, actual TBD).

### API Token Management (`/api/v1/users/token`)

_(Note: Specific permission checks like `api:create` vs `api:create:all` are handled **inside** the respective handlers. Admin actions (`_:all`) are intended to target users via `user_id`in the request body, not`username`.)\*

- `GET /`: Requires `api:read:list` or `api:read:list:all` (checked internally).
- `POST /`: Requires `api:create` or `api:create:all` (checked internally).
- `PATCH /`: Requires `api:update` or `api:update:all` (Partially update token; checked internally). Requires `token_id` in body. For admin, also requires `user_id`.
- `DELETE /`: Requires `api:delete` or `api:delete:all` (checked internally). Requires `token_id` in body. For admin, also requires `user_id`.

### Settings (`/api/v1/settings`)

- `GET /ui`: Requires `settings:read`.
- `PUT /ui`: Requires `settings:update` (Typically uses PUT as it often replaces the whole settings structure, but PATCH with `settings:update` could also be supported for partial UI changes).

### Commands, Plugins, Themes (`/api/v1/...`)

- Permissions (`commands:*`, `plugins:*`, `themes:*`) are defined but not yet enforced. Update actions might typically use PUT.

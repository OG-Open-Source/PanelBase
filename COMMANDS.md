```bash
PORT=23160 # Or your actual configured port
TOKEN=$(curl -s -X POST http://localhost:$PORT/api/v1/auth/login -H "Content-Type: application/json" -d '{"username":"admin", "password":"admin"}' | jq -r '.data.token')
echo "Using Token: $TOKEN"

# Example: Generate a token for testing (if needed)
ADMIN_TOKEN_INFO=$(curl -s -X POST http://localhost:$PORT/api/v1/users/token) \
#     -H "Authorization: Bearer $TOKEN" \
#     -H "Content-Type: application/json" \
#     -d '{"name":"Test Token","duration":"1h","scopes":{"api":["read:list"]}}')
# TEST_TOKEN_ID=$(echo $ADMIN_TOKEN_INFO | jq -r '.data.id')
# echo "Test Token ID: $TEST_TOKEN_ID"
TEST_TOKEN_ID="tok_REPLACE_WITH_ACTUAL_ID" # <-- Replace with an actual token ID
```

## API v1 Endpoints

### Authentication (`/api/v1/auth`)

**Login**

- **Method:** `POST`
- **Path:** `/api/v1/auth/login`
- **Description:** Authenticates a user and returns a JWT (`web_session` audience) set as an HTTP-only cookie and also in the response body.
- **Request Body:**
  ```json
  {
    "username": "admin",
    "password": "admin"
  }
  ```
- **Response Body (Success):**
  ```json
  {
    "status": "success",
    "message": "Login successful",
    "data": {
      "token": "eyJhbGciOi...",
      "expires_at": "2025-04-06T12:00:00Z"
    }
  }
  ```

**Register**

- **Method:** `POST`
- **Path:** `/api/v1/auth/register`
- **Description:** Creates a new user account. Default permissions are assigned.
- **Request Body:**
  ```json
  {
    "username": "newuser",
    "password": "password123",
    "email": "newuser@example.com",
    "name": "New User" // Optional, defaults to username if omitted
  }
  ```
- **Response Body (Success):** User object (excluding password).

**Refresh Token**

- **Method:** `POST`
- **Path:** `/api/v1/auth/token`
- **Requires Auth:** Yes (Valid `web_session` token via Cookie or Header)
- **Description:** Refreshes an existing `web_session` token, returning a new token and revoking the old one.
- **Response Body (Success):** Same structure as Login response.

### Account Management (Self) (`/api/v1/account`)

_(Note: These endpoints manage the logged-in user's own account)_

**Get Own Profile**

- **Method:** `GET`
- **Path:** `/api/v1/account`
- **Requires Auth:** Yes
- **Permissions:** `account:read`
- **Description:** Retrieves the profile information of the currently authenticated user.

**Update Own Profile**

- **Method:** `PATCH`
- **Path:** `/api/v1/account`
- **Requires Auth:** Yes
- **Permissions:** `account:update`
- **Description:** Partially updates the profile information (e.g., name, email) or password of the currently authenticated user. Only include fields to be updated.
- **Request Body Examples:**
  ```json
  // Update name
  { "name": "Updated Name" }
  ```
  ```json
  // Update password
  { "password": "newS3cureP@ssword" }
  ```

**Delete Own Account**

- **Method:** `DELETE`
- **Path:** `/api/v1/account`
- **Requires Auth:** Yes
- **Permissions:** `account:delete`
- **Description:** Deletes the account of the currently authenticated user.

### User Management (Admin) (`/api/v1/users`)

_(Note: These endpoints are for administrative management of user accounts)_

**List/Get Users**

- **Method:** `GET`
- **Path:** `/api/v1/users`
- **Requires Auth:** Yes
- **Permissions:** `users:read:list` or `users:read:item` (Handler needs implementation)
- **Description:** Lists all users (requires `users:read:list`). Can potentially be used to get a specific user by ID (requires `users:read:item`), but details TBD.

**Create User**

- **Method:** `POST`
- **Path:** `/api/v1/users`
- **Requires Auth:** Yes
- **Permissions:** `users:create`
- **Description:** Creates a new user account (Handler needs implementation).
- **Request Body:** Similar to `/auth/register`.

**Update User**

- **Method:** `PATCH`
- **Path:** `/api/v1/users` (or potentially `/api/v1/users/{id}` - TBD)
- **Requires Auth:** Yes
- **Permissions:** `users:update`
- **Description:** Partially updates a user's details (Handler needs implementation). Using PATCH on the collection is non-standard; a route like `/users/{id}` might be preferable.
- **Request Body:** `{"id": "usr_...", "name": "...", ...}`

**Delete User**

- **Method:** `DELETE`
- **Path:** `/api/v1/users/{id}` (Example route, actual TBD)
- **Requires Auth:** Yes
- **Permissions:** `users:delete`
- **Description:** Deletes a specific user account (Handler needs implementation).

### API Token Management (`/api/v1/users/token`)

**List/Get API Tokens**

- **Method:** `GET`
- **Path:** `/api/v1/users/token`
- **Requires Auth:** Yes
- **Permissions:** `api:read:list` (for self) or `api:read:list:all` (for admin, requires `user_id` in body - TBD)
- **Description:** Lists API tokens for the current user or, if admin, for a specified user (requires `user_id`). Can potentially get a specific token by `token_id`.
- **Request Body (Admin/Get Specific):** `{"user_id": "usr_...", "token_id": "tkn_..."}` (Optional fields)

**Create API Token**

- **Method:** `POST`
- **Path:** `/api/v1/users/token`
- **Requires Auth:** Yes
- **Permissions:** `api:create` (for self) or `api:create:all` (for admin, requires `user_id`)
- **Description:** Creates a new API token for the current user or a specified user.
- **Request Body:**
  ```json
  {
    "user_id": "usr_...", // Required only for admin action (api:create:all)
    "name": "My Script Token",
    "description": "Token for automation script", // Optional
    "duration": "P30D", // Optional (ISO 8601 Duration), defaults based on config
    "scopes": ["commands:execute"] // Optional, defaults to user's scopes if omitted
  }
  ```
- **Response Body (Success):** Token details including the JWT string.

**Update API Token**

- **Method:** `PATCH`
- **Path:** `/api/v1/users/token`
- **Requires Auth:** Yes
- **Permissions:** `api:update` (for self) or `api:update:all` (for admin, requires `user_id`)
- **Description:** Partially updates an API token's metadata (e.g., name, description, scopes). Requires `token_id`.
- **Request Body:**
  ```json
  {
    "token_id": "tkn_...", // Required
    "user_id": "usr_...", // Required only for admin action (api:update:all)
    "name": "Updated Token Name", // Optional field to update
    "description": "New desc" // Optional field to update
    // Cannot update scopes or duration via PATCH currently
  }
  ```

**Delete API Token**

- **Method:** `DELETE`
- **Path:** `/api/v1/users/token`
- **Requires Auth:** Yes
- **Permissions:** `api:delete` (for self) or `api:delete:all` (for admin, requires `user_id`)
- **Description:** Deletes (revokes) an API token. Requires `token_id`.
- **Request Body:**
  ```json
  {
    "token_id": "tkn_...", // Required
    "user_id": "usr_..." // Required only for admin action (api:delete:all)
  }
  ```

### Settings (`/api/v1/settings`)

**Get UI Settings**

- **Method:** `GET`
- **Path:** `/api/v1/settings/ui`
- **Requires Auth:** Yes
- **Permissions:** `settings:read`
- **Description:** Retrieves global UI settings.

**Update UI Settings**

- **Method:** `PUT` (or potentially PATCH)
- **Path:** `/api/v1/settings/ui`
- **Requires Auth:** Yes
- **Permissions:** `settings:update`
- **Description:** Updates global UI settings. PUT typically replaces the entire structure, while PATCH could allow partial updates. Only provide fields to be updated.
- **Request Body Examples (PATCH):**
  ```json
  { "title": "My Custom Title" }
  ```
  ```json
  {
    "logo_url": "/assets/new_logo.png",
    "custom_css": "body { background: #eee; }"
  }
  ```

### Commands, Plugins, Themes

_(Endpoints exist but handlers are placeholders. Permissions defined but not enforced yet. Updates might typically use PUT.)_

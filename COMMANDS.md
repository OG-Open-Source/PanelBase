```bash
PORT=48685 # Or your actual configured port
TOKEN=$(curl -s -X POST http://localhost:$PORT/api/v1/auth/login -H "Content-Type: application/json" -d '{"username":"admin", "password":"admin"}' | jq -r '.data.token')
echo "Using Token: $TOKEN"
```

## API v1 Endpoints

**General Update Conventions:**

- Resources like `account`, `users`, `api` tokens, and `settings` typically use the `PATCH` method for partial updates.
- Resources like `commands`, `plugins`, and `themes` typically use the `PUT` method for updates (often involving fetching from a source link).

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
      "token": "eyJhbGciOi..."
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
  ```json
  {
    "status": "success",
    "message": "User registered successfully",
    "data": {
      "active": true,
      "created_at": "2025-04-06T12:41:09Z",
      "email": "random6759@example.com",
      "id": "usr_f4b14d9f079a",
      "name": "GUK853H7ox",
      "username": "GUK853H7ox"
    }
  }
  ```

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

### API Token Management (Self) (`/api/v1/account/token`)

### `GET /api/v1/account/token` (List)

- **Description:** Retrieves metadata for all API tokens belonging to the user.
- **Requires Auth:** Yes
- **Permissions:** `api:read:list` (for self) or `api:read:list:all` (for admin)

**Query Parameters (Admin Only):**

- `user_id=usr_...`: Specify for admin action (`api:read:list:all`) to list another user's tokens.

**Success Response:** `200 OK` with an array of token metadata objects.

### `GET /api/v1/account/token/:token_id` (Get Specific)

- **Description:** Retrieves metadata for a specific API token by its ID.
- **Requires Auth:** Yes
- **Permissions:** `api:read:item` (for self) or `api:read:item:all` (for admin)
- **Path Parameters:**
  - `:token_id`: The ID (JTI starting with `tok_`) of the token to retrieve.

**Query Parameters (Admin Only):**

- `user_id=usr_...`: Specify for admin action (`api:read:item:all`) to get a token belonging to another user.

**Success Response:** `200 OK` with a single token metadata object.

**Create API Token**

- **Method:** `POST`
- **Path:** `/api/v1/account/token`
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
- **Response Body (Success):** Token details including the JWT string and ID (e.g., `tok_...`).

**Update API Token**

- **Method:** `PATCH`
- **Path:** `/api/v1/account/token`
- **Requires Auth:** Yes
- **Permissions:** `api:update` (for self) or `api:update:all` (for admin, requires `user_id`)
- **Description:** Partially updates an API token's metadata (currently only `name`). Requires `token_id`.
- **Request Body (Only `name` update supported):**
  ```json
  {
    "token_id": "tok_...", // Required
    "user_id": "usr_...", // Required only for admin action (api:update:all)
    "name": "Updated Token Name" // Optional field to update
    // Description, scopes, and duration updates are not supported via PATCH currently
  }
  ```

**Delete API Token**

- **Method:** `DELETE`
- **Path:** `/api/v1/account/token`
- **Requires Auth:** Yes
- **Permissions:** `api:delete` (for self) or `api:delete:all` (for admin, requires `user_id`)
- **Description:** Deletes (revokes) an API token. Requires `token_id`.
- **Request Body:**
  ```json
  {
    "token_id": "tok_...", // Required, Changed tkn_ to tok_
    "user_id": "usr_..." // Required only for admin action (api:delete:all)
  }
  ```

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

### Settings (`/api/v1/settings`)

**Get UI Settings**

- **Method:** `GET`
- **Path:** `/api/v1/settings/ui`
- **Requires Auth:** Yes
- **Permissions:** `settings:read`
- **Description:** Retrieves global UI settings.

**Update UI Settings**

- **Method:** `PATCH`
- **Path:** `/api/v1/settings/ui`
- **Requires Auth:** Yes
- **Permissions:** `settings:update`
- **Description:** Partially updates global UI settings. Only provide fields to be updated.
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

_(Endpoints exist but handlers are placeholders. Updates typically use PUT, potentially involving a `source_link`.)_

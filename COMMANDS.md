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
- **Description:** Authenticates a user and returns a JWT token (valid for web sessions by default).
- **Permissions:** None (Public)
- **Request Body:**
  ```json
  {
    "username": "user",
    "password": "password"
  }
  ```
- **Example:** (Used in header to get `$TOKEN`)

**Register**

- **Method:** `POST`
- **Path:** `/api/v1/auth/register`
- **Description:** Registers a new user.
- **Permissions:** None (Public)
- **Request Body:**
  ```json
  {
    "username": "newuser",
    "password": "complexpassword",
    "email": "user@example.com",
    "name": "Optional Name" // Optional
  }
  ```
- **Example:**
  ```bash
  curl -X POST http://localhost:$PORT/api/v1/auth/register \
       -H "Content-Type: application/json" \
       -d '{"username":"testuser","password":"testpassword123","email":"test@example.com"}'
  ```

**Refresh Token**

- **Method:** `POST`
- **Path:** `/api/v1/auth/token`
- **Description:** Refreshes an existing web session JWT token. Requires a valid token passed via Cookie (`panelbase_jwt`).
- **Permissions:** Requires valid web session token.
- **Request Body:** None
- **Example:** (Requires browser context or manually setting the cookie)

### API Token Management (`/api/v1/users/token`)

Requires Authorization Header: `-H "Authorization: Bearer $TOKEN"`

**Get API Tokens**

- **Method:** `GET`
- **Path:** `/api/v1/users/token`
- **Description:** Retrieves API token metadata.
  - No request body: Lists tokens for the current user.
  - Body `{"id": "TOKEN_JTI"}`: Gets details for a specific token.
- **Permissions:**
  - List self: `read:list`
  - Get self item: `read:item`
  - Get others' item: `read:item:all` (Admin only, based on token ownership)
- **Request Body (Optional for specific token):**
  ```json
  {
    "id": "tok_..."
  }
  ```
- **Examples:**

  ```bash
  # List own tokens (No Request Body)
  curl -X GET http://localhost:$PORT/api/v1/users/token \
       -H "Authorization: Bearer $TOKEN"

  # Get own token with JTI (replace TEST_TOKEN_ID)
  curl -X GET http://localhost:$PORT/api/v1/users/token \
       -H "Authorization: Bearer $TOKEN" \
       -H "Content-Type: application/json" \
       -d "{\"id\": \"$TEST_TOKEN_ID\"}"

  # Get someone else's token (Admin, replace TEST_TOKEN_ID)
  curl -X GET http://localhost:$PORT/api/v1/users/token \
       -H "Authorization: Bearer $TOKEN" \
       -H "Content-Type: application/json" \
       -d "{\"id\": \"tok_SOME_OTHER_USERS_TOKEN_ID\"}" # Requires read:item:all
  ```

**Create API Token**

- **Method:** `POST`
- **Path:** `/api/v1/users/token`
- **Description:** Creates a new API token.
- **Permissions:**
  - Create for self: `api:create`
  - Create for other user: `api:create:all`
- **Request Body:**

  ```json
  // For self
  {
    "name": "My New Token",
    "duration": "P7D", // ISO 8601 Date Duration (e.g., P7D, P1M, P1Y6M)
    "description": "Optional description", // Optional
    "scopes": { "api": ["read:list"] } // Optional, defaults to user scopes if omitted.
  }

  // For another user (Admin)
  {
    "username": "testuser",
    "name": "Token For TestUser",
    "duration": "P30D", // ISO 8601 Date Duration
    "scopes": { "api": ["read:list"] } // Optional
  }
  ```

- **Examples:**

  ```bash
  # Create token for self (scopes optional, duration P90D)
  curl -X POST http://localhost:$PORT/api/v1/users/token \
       -H "Authorization: Bearer $TOKEN" \
       -H "Content-Type: application/json" \
       -d '{"name":"MyCLI Token","duration":"P90D"}'

  # Create token for user 'testuser' (Admin, duration P1H -> use P1D instead)
  curl -X POST http://localhost:$PORT/api/v1/users/token \
       -H "Authorization: Bearer $TOKEN" \
       -H "Content-Type: application/json" \
       -d '{"username":"testuser","name":"Test User Token","duration":"P1D"}' # Using P1D for 1 day
  ```

**Update API Token**

- **Method:** `PUT`
- **Path:** `/api/v1/users/token`
- **Description:** Updates the metadata (name, description, scopes) of an existing API token.
- **Permissions:**
  - Update own token: `api:update`
  - Update other's token: `api:update:all`
- **Request Body:**

  ```json
  // Update own token
  {
    "id": "tok_...", // JTI of the token to update
    "name": "Updated Token Name", // Optional
    "description": "New optional description", // Optional
    "scopes": { "api": ["read:list"] } // Optional
  }

  // Update other's token (Admin)
  {
    "username": "testuser",
    "id": "tok_...", // JTI of the token to update
    "name": "Updated TestUser Token Name" // Optional
  }
  ```

- **Examples:**

  ```bash
  # Update own token 'tok_...'
  curl -X PUT http://localhost:$PORT/api/v1/users/token \
       -H "Authorization: Bearer $TOKEN" \
       -H "Content-Type: application/json" \
       -d '{"id":"tok_...","name":"Renamed My Token"}'

  # Update user 'testuser' token 'tok_...' (Admin)
  curl -X PUT http://localhost:$PORT/api/v1/users/token \
       -H "Authorization: Bearer $TOKEN" \
       -H "Content-Type: application/json" \
       -d '{"username":"testuser","id":"tok_...","description":"Admin updated description"}'
  ```

**Delete API Token**

- **Method:** `DELETE`
- **Path:** `/api/v1/users/token`
- **Description:** Deletes (revokes) an API token.
- **Permissions:**
  - Delete own token: `api:delete`
  - Delete other's token: `api:delete:all`
- **Request Body:**

  ```json
  // Delete own token
  {
    "token_id": "tok_..." // JTI of the token to delete
  }

  // Delete other's token (Admin)
  {
    "username": "testuser",
    "token_id": "tok_..." // JTI of the token to delete
  }
  ```

- **Examples:**

  ```bash
  # Delete own token 'tok_...'
  curl -X DELETE http://localhost:$PORT/api/v1/users/token \
       -H "Authorization: Bearer $TOKEN" \
       -H "Content-Type: application/json" \
       -d '{"token_id":"tok_..."}'

  # Delete user 'testuser' token 'tok_...' (Admin)
  curl -X DELETE http://localhost:$PORT/api/v1/users/token \
       -H "Authorization: Bearer $TOKEN" \
       -H "Content-Type: application/json" \
       -d '{"username":"testuser","token_id":"tok_..."}'
  ```

### Settings (`/api/v1/settings`)

Requires Authorization Header: `-H "Authorization: Bearer $TOKEN"`

**Get UI Settings**

- **Method:** `GET`
- **Path:** `/api/v1/settings/ui`
- **Description:** Retrieves global UI settings (title, logo, etc.).
- **Permissions:** `settings:read`
- **Example:**
  ```bash
  curl -X GET http://localhost:$PORT/api/v1/settings/ui \
       -H "Authorization: Bearer $TOKEN"
  ```

**Update UI Settings**

- **Method:** `PUT`
- **Path:** `/api/v1/settings/ui`
- **Description:** Updates global UI settings. Only provided fields are updated.
- **Permissions:** `settings:update`
- **Request Body:**
  ```json
  {
    "title": "My Custom Panel Title", // Optional
    "logo_url": "/assets/images/custom_logo.png", // Optional
    "favicon_url": "/assets/images/favicon.ico", // Optional
    "custom_css": "body { background-color: #f0f0f0; }", // Optional
    "custom_js": "console.log('UI settings updated!');" // Optional
  }
  ```
- **Example:**

  ```bash
  # Update only the title
  curl -X PUT http://localhost:$PORT/api/v1/settings/ui \
       -H "Authorization: Bearer $TOKEN" \
       -H "Content-Type: application/json" \
       -d '{"title":"My Awesome Panel"}'

  # Update multiple fields
  curl -X PUT http://localhost:$PORT/api/v1/settings/ui \
       -H "Authorization: Bearer $TOKEN" \
       -H "Content-Type: application/json" \
       -d '{"logo_url":"/img/new_logo.svg", "custom_css":"h1 { color: blue; }"}'
  ```

### User Management (`/api/v1/users`)

Requires Authorization Header: `-H "Authorization: Bearer $TOKEN"`
_(Note: Handlers for these are mostly placeholders and need implementation)_

- **GET /users**: List/Get users (Requires `users:read:list`/`users:read:item`, handler needs implementation)
- **POST /users**: Create user (Requires `users:create`, handler needs implementation)
- **PUT /users**: Update user (Requires `users:update`, handler needs implementation + ownership check)
- **DELETE /users**: Delete user (Requires `users:delete`, handler needs implementation + ownership check)

### Other Resources (Placeholders)

Requires Authorization Header: `-H "Authorization: Bearer $TOKEN"`
_(Note: These are placeholders and need implementation and permission checks)_

- `/api/v1/commands` (GET, POST, PUT, DELETE)
- `/api/v1/plugins` (GET, POST, PUT, DELETE)
- `/api/v1/themes` (GET, POST, PUT, DELETE)

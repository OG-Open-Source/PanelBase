#!/bin/bash

# --- Configuration ---
CONFIG_FILE="configs/config.toml"
DEFAULT_USERNAME="admin"
DEFAULT_PASSWORD="admin"
TEST_REG_USERNAME="testuser_$(date +%s)"
TEST_REG_PASSWORD="testpassword123"
TEST_REG_EMAIL="${TEST_REG_USERNAME}@example.com"

# --- State Variables ---
ADMIN_SESSION_TOKEN=""
ADMIN_USER_ID=""
CREATED_API_TOKEN=""
CREATED_API_TOKEN_ID="" # Changed from tkn_ to tok_ prefix internally now
TOKEN_TYPE="None"       # Track current token type for logging

# --- Helper Functions ---

check_command() {
	if ! command -v "$1" &>/dev/null; then
		echo "Error: Required command '$1' not found. Please install it." >&2
		exit 1
	fi
}

get_api_port() {
	if [[ ! -f "$CONFIG_FILE" ]]; then
		echo "Error: Config file '$CONFIG_FILE' not found." >&2
		exit 1
	fi
	PORT=$(grep -E '^\s*port\s*=' "$CONFIG_FILE" | head -n 1 | awk -F '=' '{gsub(/ /,"",$2); print $2}')
	if [[ -z "$PORT" ]]; then
		echo "Error: Could not extract port from '$CONFIG_FILE'." >&2
		exit 1
	fi
	echo "$PORT"
}

decode_jwt_payload() {
	local jwt=$1
	if [[ -z "$jwt" ]]; then
		echo "JWT is empty." >&2
		return "" # Return empty string on failure
	fi
	local payload_base64=$(echo "$jwt" | cut -d '.' -f 2)
	payload_base64=${payload_base64//-/+}
	payload_base64=${payload_base64//_//}
	case $((${#payload_base64} % 4)) in
	1) payload_base64="${payload_base64}===" ;;
	2) payload_base64="${payload_base64}==" ;;
	3) payload_base64="${payload_base64}=" ;;
	esac
	decoded_payload=$(echo "$payload_base64" | base64 --decode 2>/dev/null | jq -c '.' 2>/dev/null)
	if [[ $? -ne 0 ]]; then
		echo "Error decoding Base64 or invalid JSON in payload for JWT: $jwt" >&2
		return "" # Return empty string on failure
	fi
	echo "--- Decoded JWT Payload ---" >&2 # Log to stderr
	echo "$decoded_payload" | jq '.' >&2   # Pretty print to stderr
	echo "--------------------------" >&2  # Log to stderr
	echo "$decoded_payload"                # Return the compact JSON string to stdout
}

# Function to make API requests
# Usage: make_request METHOD PATH [DATA] [TOKEN] [DESCRIPTION]
# Prints informational logs to stderr, prints raw response body to stdout
make_request() {
	local method=$1
	local path=$2
	local data=$3
	local token=$4
	local description=$5
	local url="${BASE_URL}${path}"
	local headers=("-H" "Content-Type: application/json") # Use array for headers
	local curl_opts=("-s" "-X" "${method}")               # Basic curl options

	# Print info logs to stderr
	echo "-----------------------------------------------------" >&2
	echo ">> Test: ${description}" >&2
	echo ">> Request: ${method} ${path}" >&2
	[[ -n "$data" ]] && echo ">> Data: ${data}" >&2
	[[ -n "$token" ]] && echo ">> Using Token: YES (type: ${TOKEN_TYPE})" >&2 || echo ">> Using Token: NO" >&2

	if [[ -n "$token" ]]; then
		headers+=("-H" "Authorization: Bearer ${token}")
	fi

	curl_opts+=("${headers[@]}") # Add headers to curl options

	# Construct the command string for display only
	local display_cmd="curl ${curl_opts[*]} \"${url}\""
	if [[ -n "$data" ]]; then
		curl_opts+=("-d" "${data}") # Add data option
		display_cmd+=" -d '${data}'"
	fi

	echo ">> Curl Command:" >&2
	echo "   ${display_cmd}" >&2 # Show the command being run

	# Execute curl and capture response body to stdout
	# Errors from curl itself (like connection refused) will go to stderr
	response_body=$(curl "${curl_opts[@]}" "${url}")
	local curl_exit_code=$?

	echo ">> Response:" >&2
	# Pretty print if valid JSON to stderr, otherwise print raw to stderr
	if jq -e . >/dev/null 2>&1 <<<"$response_body"; then
		echo "$response_body" | jq '.' >&2
	else
		echo "$response_body" >&2
		if [[ $curl_exit_code -ne 0 ]]; then
			echo "[Curl Error: Exit code $curl_exit_code]" >&2
		fi
	fi

	echo "-----------------------------------------------------" >&2
	echo "" >&2
	sleep 0.5 # Small delay

	# Output the raw response body to stdout for capture
	echo "$response_body"
}

# --- Main Script ---

check_command curl
check_command jq
check_command base64
check_command grep
check_command awk

API_PORT=$(get_api_port)
BASE_URL="http://localhost:${API_PORT}/api/v1"
echo "Testing API at: ${BASE_URL}" >&2 # Log setup info to stderr
echo "" >&2

echo "=== Running API Tests ===" >&2 # Log section header to stderr

# --- Auth Tests ---
TOKEN_TYPE="None"
# Use make_request and ignore its stdout (response body) for simple calls
make_request POST "/auth/register" "{\"username\":\"${TEST_REG_USERNAME}\",\"password\":\"${TEST_REG_PASSWORD}\",\"email\":\"${TEST_REG_EMAIL}\"}" "" "Registration - Success" >/dev/null
make_request POST "/auth/register" "{\"username\":\"${TEST_REG_USERNAME}\",\"password\":\"${TEST_REG_PASSWORD}\",\"email\":\"${TEST_REG_EMAIL}\"}" "" "Registration - Failure (Username Exists)" >/dev/null
make_request POST "/auth/register" "{\"username\":\"another_${TEST_REG_USERNAME}\",\"email\":\"another@example.com\"}" "" "Registration - Failure (Missing Password)" >/dev/null
make_request POST "/auth/login" "{\"username\":\"${DEFAULT_USERNAME}\",\"password\":\"wrongpassword\"}" "" "Login - Failure (Wrong Password)" >/dev/null

# Login Admin and capture token/user ID - Capture stdout here
echo ">> Test: Login - Success (Default Admin)" >&2
login_response_body=$(make_request POST "/auth/login" "{\"username\":\"${DEFAULT_USERNAME}\",\"password\":\"${DEFAULT_PASSWORD}\"}" "" "Login - Success (Default Admin)")

# Now jq operates only on the response body captured in login_response_body
ADMIN_SESSION_TOKEN=$(echo "$login_response_body" | jq -r '.data.token // empty')
if [[ -z "$ADMIN_SESSION_TOKEN" ]]; then
	echo "FATAL: Could not extract admin session token. Stopping tests." >&2
	exit 1
fi
echo "Admin Session Token captured." >&2
TOKEN_TYPE="Web Session"
# Decode payload and extract User ID (sub)
decoded_payload=$(decode_jwt_payload "$ADMIN_SESSION_TOKEN")
if [[ -n "$decoded_payload" ]]; then
	ADMIN_USER_ID=$(echo "$decoded_payload" | jq -r '.sub // empty')
	if [[ -n "$ADMIN_USER_ID" ]]; then
		echo "Admin User ID captured: ${ADMIN_USER_ID}" >&2
	else
		echo "Error: Could not extract User ID (sub) from admin token payload." >&2
	fi
else
	echo "Error: Could not decode admin token payload." >&2
fi
echo "-----------------------------------------------------" >&2 # Add separator manually after capture step
echo "" >&2

# --- Account/Token Tests (Self Actions using Session Token) ---
make_request GET "/settings/ui" "" "" "Get Settings - Failure (No Token)" >/dev/null
make_request GET "/settings/ui" "" "${ADMIN_SESSION_TOKEN}" "Get Settings - Success (Using Admin Session Token)" >/dev/null

# Create API Token for Admin Self
create_token_data="{\"name\":\"Admin API Token 1\",\"duration\":\"P7D\"}"
create_response_body=$(make_request POST "/account/token" "${create_token_data}" "${ADMIN_SESSION_TOKEN}" "Create API Token - Success (Admin Self)")
CREATED_API_TOKEN=$(echo "$create_response_body" | jq -r '.data.token // empty')
CREATED_API_TOKEN_ID=$(echo "$create_response_body" | jq -r '.data.id // empty')
if [[ -z "$CREATED_API_TOKEN" || -z "$CREATED_API_TOKEN_ID" ]]; then
	echo "Error: Could not extract API token or ID from creation response. Token tests may fail." >&2
else
	echo "API Token created. ID: ${CREATED_API_TOKEN_ID}" >&2
	decoded_api_payload=$(decode_jwt_payload "$CREATED_API_TOKEN") # Decode API token
	if [[ -z "$decoded_api_payload" ]]; then
		echo "Error: Could not decode created API token payload." >&2
	fi
fi
echo "-----------------------------------------------------" >&2 # Add separator manually after capture step
echo "" >&2

# List API Tokens for Admin Self (using Session Token)
make_request GET "/account/token" "" "${ADMIN_SESSION_TOKEN}" "List API Tokens - Success (Admin Self, using Session Token)" >/dev/null

# Get Specific API Token for Admin Self (using Session Token)
if [[ -n "$CREATED_API_TOKEN_ID" ]]; then
	specific_token_path="/account/token/${CREATED_API_TOKEN_ID}"
	make_request GET "${specific_token_path}" "" "${ADMIN_SESSION_TOKEN}" "Get Specific API Token - Success (Admin Self, using Session Token)" >/dev/null
else
	echo "Skipping Get Specific Token test - Token ID not captured." >&2
fi

# Update API Token for Admin Self (using Session Token)
if [[ -n "$CREATED_API_TOKEN_ID" ]]; then
	update_token_path="/account/token/${CREATED_API_TOKEN_ID}" # Use ID in path
	update_token_data='{"name":"Admin API Token UPDATED"}' # Only send fields to update
	make_request PATCH "${update_token_path}" "${update_token_data}" "${ADMIN_SESSION_TOKEN}" "Update API Token Name - Success (Admin Self, using Session Token)" >/dev/null
else
	echo "Skipping Update Token test - Token ID not captured." >&2
fi

# List API Tokens again to see update (using Session Token)
make_request GET "/account/token" "" "${ADMIN_SESSION_TOKEN}" "List API Tokens After Update - Success (Admin Self, using Session Token)" >/dev/null

# Delete API Token for Admin Self (using Session Token)
if [[ -n "$CREATED_API_TOKEN_ID" ]]; then
	delete_token_path="/account/token/${CREATED_API_TOKEN_ID}" # Use ID in path
	make_request DELETE "${delete_token_path}" "" "${ADMIN_SESSION_TOKEN}" "Delete API Token - Success (Admin Self, using Session Token)" >/dev/null # No request body needed
else
	echo "Skipping Delete Token test - Token ID not captured." >&2
fi

# List API Tokens one last time to confirm deletion (using Session Token)
make_request GET "/account/token" "" "${ADMIN_SESSION_TOKEN}" "List API Tokens After Delete - Success (Admin Self, using Session Token)" >/dev/null

# --- Optional: Test using the created API Token ---
if [[ -n "$CREATED_API_TOKEN" ]]; then
	echo "" >&2
	echo "=== Testing using the created API Token ===" >&2
	TOKEN_TYPE="API Token"
	# Example: Try getting settings using the API token
	# Note: The token might have been deleted already if tests run sequentially
	make_request GET "/settings/ui" "" "${CREATED_API_TOKEN}" "Get Settings - Attempt using API Token" >/dev/null
	# Example: Try listing tokens using the API token
	make_request GET "/account/token" "" "${CREATED_API_TOKEN}" "List API Tokens - Attempt using API Token" >/dev/null
else
	echo "Skipping tests using the created API token as it wasn't captured." >&2
fi

echo "=== API Tests Completed ===" >&2

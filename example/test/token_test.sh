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
CREATED_API_TOKEN_ID=""

# --- Helper Functions ---

check_command() {
	if ! command -v "$1" &>/dev/null; then
		echo "Error: Required command '$1' not found. Please install it."
		exit 1
	fi
}

get_api_port() {
	if [[ ! -f "$CONFIG_FILE" ]]; then
		echo "Error: Config file '$CONFIG_FILE' not found."
		exit 1
	fi
	PORT=$(grep -E '^\s*port\s*=' "$CONFIG_FILE" | head -n 1 | awk -F '=' '{gsub(/ /,"",$2); print $2}')
	if [[ -z "$PORT" ]]; then
		echo "Error: Could not extract port from '$CONFIG_FILE'."
		exit 1
	fi
	echo "$PORT"
}

decode_jwt_payload() {
	local jwt=$1
	if [[ -z "$jwt" ]]; then
		echo "JWT is empty."
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
		echo "Error decoding Base64 or invalid JSON in payload." >&2
		return "" # Return empty string on failure
	fi
	echo "--- Decoded JWT Payload ---"
	echo "$decoded_payload" | jq '.' # Pretty print for display
	echo "--------------------------"
	echo "$decoded_payload" # Return the compact JSON string
}

# Function to make API requests
# Usage: make_request METHOD PATH [DATA] [TOKEN] [DESCRIPTION]
make_request() {
	local method=$1
	local path=$2
	local data=$3
	local token=$4
	local description=$5
	local url="${BASE_URL}${path}"
	local headers=(-H "Content-Type: application/json")
	local data_arg=""

	echo "-----------------------------------------------------"
	echo ">> Test: ${description}"
	echo ">> Request: ${method} ${path}"
	[[ -n "$data" ]] && echo ">> Data: ${data}"
	[[ -n "$token" ]] && echo ">> Using Token: YES (type: ${TOKEN_TYPE})" || echo ">> Using Token: NO"

	if [[ -n "$token" ]]; then
		headers+=(-H "Authorization: Bearer ${token}")
	fi
	if [[ -n "$data" ]]; then
		data_arg="-d '${data}'"
	fi

	local curl_cmd="curl -s -X ${method} \"${url}\" ${headers[*]} ${data_arg}"

	echo ">> Curl Command:"
	echo "   ${curl_cmd}"

	echo ">> Response:"
	response=$(eval "${curl_cmd}")

	if jq -e . >/dev/null 2>&1 <<<"$response"; then
		echo "$response" | jq '.'
	else
		echo "$response"
	fi

	# Try to extract and decode token from response data if exists
	local resp_token=$(echo "$response" | jq -r '.data.token // empty')
	if [[ -n "$resp_token" ]]; then
		echo ""
		decode_jwt_payload "$resp_token" >/dev/null # Decode but don't print payload here, rely on caller
	fi

	echo "-----------------------------------------------------"
	echo ""
	sleep 0.5        # Small delay
	echo "$response" # Return response body for capture
}

# --- Main Script ---

check_command curl
check_command jq
check_command base64
check_command grep
check_command awk

API_PORT=$(get_api_port)
BASE_URL="http://localhost:${API_PORT}/api/v1"
echo "Testing API at: ${BASE_URL}"
echo ""

echo "=== Running API Tests ==="

# --- Auth Tests ---
TOKEN_TYPE="None"
make_request POST "/auth/register" "{\"username\":\"${TEST_REG_USERNAME}\",\"password\":\"${TEST_REG_PASSWORD}\",\"email\":\"${TEST_REG_EMAIL}\"}" "" "Registration - Success"
make_request POST "/auth/register" "{\"username\":\"${TEST_REG_USERNAME}\",\"password\":\"${TEST_REG_PASSWORD}\",\"email\":\"${TEST_REG_EMAIL}\"}" "" "Registration - Failure (Username Exists)"
make_request POST "/auth/register" "{\"username\":\"another_${TEST_REG_USERNAME}\",\"email\":\"another@example.com\"}" "" "Registration - Failure (Missing Password)"
make_request POST "/auth/login" "{\"username\":\"${DEFAULT_USERNAME}\",\"password\":\"wrongpassword\"}" "" "Login - Failure (Wrong Password)"

# Login Admin and capture token/user ID
echo ">> Test: Login - Success (Default Admin)"
login_response=$(make_request POST "/auth/login" "{\"username\":\"${DEFAULT_USERNAME}\",\"password\":\"${DEFAULT_PASSWORD}\"}" "" "Login - Success (Default Admin)")
ADMIN_SESSION_TOKEN=$(echo "$login_response" | jq -r '.data.token // empty')
if [[ -z "$ADMIN_SESSION_TOKEN" ]]; then
	echo "FATAL: Could not extract admin session token. Stopping tests."
	exit 1
fi
echo "Admin Session Token captured."
TOKEN_TYPE="Web Session"
# Decode payload and extract User ID (sub)
decoded_payload=$(decode_jwt_payload "$ADMIN_SESSION_TOKEN")
if [[ -n "$decoded_payload" ]]; then
	ADMIN_USER_ID=$(echo "$decoded_payload" | jq -r '.sub // empty')
	if [[ -n "$ADMIN_USER_ID" ]]; then
		echo "Admin User ID captured: ${ADMIN_USER_ID}"
	else
		echo "Error: Could not extract User ID (sub) from admin token payload."
	fi
else
	echo "Error: Could not decode admin token payload."
fi
echo "-----------------------------------------------------"
echo ""

# --- Account/Token Tests (Self Actions using Session Token) ---
# Note: Using session token for API actions after middleware update

make_request GET "/settings/ui" "" "" "Get Settings - Failure (No Token)"
make_request GET "/settings/ui" "" "${ADMIN_SESSION_TOKEN}" "Get Settings - Success (Using Admin Session Token)"

# Create API Token for Admin Self
create_token_data="{\"name\":\"Admin API Token 1\",\"duration\":\"P7D\"}" # Create for self, no user_id needed
create_response=$(make_request POST "/account/token" "${create_token_data}" "${ADMIN_SESSION_TOKEN}" "Create API Token - Success (Admin Self)")
CREATED_API_TOKEN=$(echo "$create_response" | jq -r '.data.token // empty')
CREATED_API_TOKEN_ID=$(echo "$create_response" | jq -r '.data.id // empty')
if [[ -z "$CREATED_API_TOKEN" || -z "$CREATED_API_TOKEN_ID" ]]; then
	echo "Error: Could not extract API token or ID from creation response. Token tests may fail."
else
	echo "API Token created. ID: ${CREATED_API_TOKEN_ID}"
	echo ""
	decode_jwt_payload "$CREATED_API_TOKEN" >/dev/null # Decode but don't print payload
fi

# List API Tokens for Admin Self (using Session Token)
make_request GET "/account/token" "" "${ADMIN_SESSION_TOKEN}" "List API Tokens - Success (Admin Self, using Session Token)"

# Get Specific API Token for Admin Self (using Session Token)
if [[ -n "$CREATED_API_TOKEN_ID" ]]; then
	get_token_data="{\"token_id\":\"${CREATED_API_TOKEN_ID}\"}"
	make_request GET "/account/token" "${get_token_data}" "${ADMIN_SESSION_TOKEN}" "Get Specific API Token - Success (Admin Self, using Session Token)"
else
	echo "Skipping Get Specific Token test - Token ID not captured."
fi

# Update API Token for Admin Self (using Session Token)
if [[ -n "$CREATED_API_TOKEN_ID" ]]; then
	update_token_data="{\"token_id\":\"${CREATED_API_TOKEN_ID}\", \"name\":\"Admin API Token UPDATED\", \"description\":\"Updated Desc\"}"
	make_request PATCH "/account/token" "${update_token_data}" "${ADMIN_SESSION_TOKEN}" "Update API Token - Success (Admin Self, using Session Token)"
else
	echo "Skipping Update Token test - Token ID not captured."
fi

# List API Tokens again to see update (using Session Token)
make_request GET "/account/token" "" "${ADMIN_SESSION_TOKEN}" "List API Tokens After Update - Success (Admin Self, using Session Token)"

# Delete API Token for Admin Self (using Session Token)
if [[ -n "$CREATED_API_TOKEN_ID" ]]; then
	delete_token_data="{\"token_id\":\"${CREATED_API_TOKEN_ID}\"}" # Delete the one we created
	make_request DELETE "/account/token" "${delete_token_data}" "${ADMIN_SESSION_TOKEN}" "Delete API Token - Success (Admin Self, using Session Token)"
else
	echo "Skipping Delete Token test - Token ID not captured."
fi

# List API Tokens one last time to confirm deletion (using Session Token)
make_request GET "/account/token" "" "${ADMIN_SESSION_TOKEN}" "List API Tokens After Delete - Success (Admin Self, using Session Token)"

# --- Optional: Test using the created API Token ---
if [[ -n "$CREATED_API_TOKEN" ]]; then
	echo ""
	echo "=== Testing using the created API Token ==="
	TOKEN_TYPE="API Token"
	# Example: Try getting settings using the API token
	make_request GET "/settings/ui" "" "${CREATED_API_TOKEN}" "Get Settings - Success (Using API Token)"
	# Example: Try listing tokens using the API token (should work if scope allows)
	make_request GET "/account/token" "" "${CREATED_API_TOKEN}" "List API Tokens - Success (Using API Token)"
	# IMPORTANT: The created API token might be deleted by previous tests.
	# These tests might fail if run after the DELETE test above.
else
	echo "Skipping tests using the created API token as it wasn't captured."
fi

echo "=== API Tests Completed ==="

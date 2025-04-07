#!/bin/bash

[ -f ~/utilkit.sh ] && source ~/utilkit.sh || bash <(curl -sL utilkit.ogtt.tk) && source ~/utilkit.sh

# --- Configuration ---
CONFIG_FILE="configs/config.toml"
# Use default admin credentials or change as needed for testing
DEFAULT_USERNAME="admin"
DEFAULT_PASSWORD="admin"
# Credentials for registration test
TEST_REG_USERNAME="testuser_$(date +%s)" # Add timestamp for uniqueness
TEST_REG_PASSWORD="testpassword123"
TEST_REG_EMAIL="${TEST_REG_USERNAME}@example.com"

# --- Helper Functions ---

# Function to extract port from config.toml
get_api_port() {
	if [[ ! -f "$CONFIG_FILE" ]]; then
		echo "Error: Config file '$CONFIG_FILE' not found."
		exit 1
	fi
	# Attempt to parse port= line, handling spaces around '='
	PORT=$(grep -E '^\s*port\s*=' "$CONFIG_FILE" | head -n 1 | awk -F '=' '{gsub(/ /,"",$2); print $2}')
	if [[ -z "$PORT" ]]; then
		echo "Error: Could not extract port from '$CONFIG_FILE'."
		exit 1
	fi
	echo "$PORT"
}

# Function to decode JWT Payload (does not verify signature)
decode_jwt_payload() {
	local jwt=$1
	if [[ -z "$jwt" ]]; then
		echo "JWT is empty."
		return
	fi
	local payload=$(echo "$jwt" | cut -d '.' -f 2)
	# Replace URL-safe base64 chars
	payload=${payload//-/+}
	payload=${payload//_//}
	# Add padding if necessary
	case $((${#payload} % 4)) in
	1) payload="${payload}===" ;;
	2) payload="${payload}==" ;;
	3) payload="${payload}=" ;;
	esac
	echo "--- Decoded JWT Payload ---"
	echo "$payload" | base64 --decode 2>/dev/null | jq '.' || echo "Error decoding Base64 or invalid JSON in payload."
	echo "--------------------------"
}

# Function to make API requests
# Usage: make_request METHOD PATH [DATA] [TOKEN]
make_request() {
	local method=$1
	local path=$2
	local data=$3
	local token=$4
	local url="${BASE_URL}${path}"
	local headers=(-H "Content-Type: application/json")

	echo "-----------------------------------------------------"
	echo ">> Request: ${method} ${path}"
	[[ -n "$data" ]] && echo ">> Data: ${data}"
	[[ -n "$token" ]] && echo ">> Using Token: YES" || echo ">> Using Token: NO"

	if [[ -n "$token" ]]; then
		headers+=(-H "Authorization: Bearer ${token}")
	fi

	local curl_cmd="curl -s -X ${method} \"${url}\" ${headers[*]}"
	[[ -n "$data" ]] && curl_cmd+=" -d '${data}'"

	echo ">> Curl Command:"
	echo "   ${curl_cmd}" # Show the command being run

	echo ">> Response:"
	# Use process substitution to capture body and headers/status if needed later
	response=$(eval "${curl_cmd}")
	# response_code=$(curl -s -o /dev/null -w "%{http_code}" -X ${method} \"${url}\" ${headers[*]} ${data_arg})

	# Pretty print if valid JSON, otherwise print raw
	if jq -e . >/dev/null 2>&1 <<<"$response"; then
		echo "$response" | jq '.'
	else
		echo "$response"
	fi

	# Try to extract and decode token from response data if exists
	local resp_token=$(echo "$response" | jq -r '.data.token // empty')
	if [[ -n "$resp_token" ]]; then
		echo ""
		decode_jwt_payload "$resp_token"
	fi

	echo "-----------------------------------------------------"
	echo ""
	sleep 1 # Small delay between requests
}

# --- Main Script ---

# Check dependencies
deps=(curl jq base64 grep awk)
CHECK_DEPS -a

# Get Port and Set Base URL
API_PORT=$(get_api_port)
BASE_URL="http://localhost:${API_PORT}/api/v1"
echo "Testing API at: ${BASE_URL}"
echo ""

# --- Test Cases ---

echo "=== Running API Tests ==="

# 1. Registration - Success
echo "** Test Case: Registration (Success) **"
reg_data="{\"username\":\"${TEST_REG_USERNAME}\",\"password\":\"${TEST_REG_PASSWORD}\",\"email\":\"${TEST_REG_EMAIL}\"}"
make_request POST "/auth/register" "${reg_data}"

# 2. Registration - Failure (Username already exists)
echo "** Test Case: Registration (Failure - Username Exists) **"
make_request POST "/auth/register" "${reg_data}" # Use the same data

# 3. Registration - Failure (Missing required field - password)
echo "** Test Case: Registration (Failure - Missing Password) **"
reg_fail_data="{\"username\":\"anotheruser_${TEST_REG_USERNAME}\",\"email\":\"another@example.com\"}"
make_request POST "/auth/register" "${reg_fail_data}"

# 4. Login - Failure (Wrong Password)
echo "** Test Case: Login (Failure - Wrong Password) **"
login_fail_data="{\"username\":\"${DEFAULT_USERNAME}\",\"password\":\"wrongpassword\"}"
make_request POST "/auth/login" "${login_fail_data}"

# 5. Login - Success (Default Admin)
echo "** Test Case: Login (Success - Default Admin) **"
login_data="{\"username\":\"${DEFAULT_USERNAME}\",\"password\":\"${DEFAULT_PASSWORD}\"}"
# Capture the response to extract the token
login_response=$(curl -s -X POST "${BASE_URL}/auth/login" -H "Content-Type: application/json" -d "${login_data}")
echo ">> Response:"
echo "$login_response" | jq '.'
ADMIN_SESSION_TOKEN=$(echo "$login_response" | jq -r '.data.token // empty')
if [[ -z "$ADMIN_SESSION_TOKEN" ]]; then
	echo "Error: Could not extract admin session token from login response."
	# exit 1 # Decide if you want to stop tests here
else
	echo ""
	decode_jwt_payload "$ADMIN_SESSION_TOKEN"
fi
echo "-----------------------------------------------------"
echo ""
sleep 1

# 6. Get Settings - Failure (No Token)
echo "** Test Case: Get Settings (Failure - No Token) **"
make_request GET "/settings/ui"

# 7. Get Settings - Success (Using Admin Session Token)
# Note: Technically API endpoints should prefer API tokens, but let's see if this works/fails
echo "** Test Case: Get Settings (Success? - Using Admin Session Token) **"
make_request GET "/settings/ui" "" "${ADMIN_SESSION_TOKEN}"

# 8. Create API Token - Success (For Admin Self)
echo "** Test Case: Create API Token (Success - Admin Self) **"
create_token_data="{\"name\":\"Test Script API Token\",\"duration\":\"P1D\"}"
# Capture response to get the new API token
create_response=$(curl -s -X POST "${BASE_URL}/users/token" -H "Content-Type: application/json" -H "Authorization: Bearer ${ADMIN_SESSION_TOKEN}" -d "${create_token_data}")
echo ">> Response:"
echo "$create_response" | jq '.'
NEW_API_TOKEN=$(echo "$create_response" | jq -r '.data.token // empty')
NEW_API_TOKEN_ID=$(echo "$create_response" | jq -r '.data.id // empty')
if [[ -z "$NEW_API_TOKEN" || -z "$NEW_API_TOKEN_ID" ]]; then
	echo "Error: Could not extract new API token or ID from creation response."
	# exit 1
else
	echo ""
	decode_jwt_payload "$NEW_API_TOKEN"
	echo "   New Token ID: ${NEW_API_TOKEN_ID}"
fi
echo "-----------------------------------------------------"
echo ""
sleep 1

# 9. List API Tokens - Success (Using the New API Token)
echo "** Test Case: List API Tokens (Success - Using New API Token) **"
if [[ -n "$NEW_API_TOKEN" ]]; then
	make_request GET "/users/token" "" "${NEW_API_TOKEN}"
else
	echo "Skipping test - Failed to create API token earlier."
fi

# 10. Delete API Token - Success (Using Admin Session Token)
echo "** Test Case: Delete API Token (Success - Using Admin Session Token) **"
if [[ -n "$ADMIN_SESSION_TOKEN" && -n "$NEW_API_TOKEN_ID" ]]; then
	delete_token_data="{\"token_id\":\"${NEW_API_TOKEN_ID}\"}" # Delete the one we just created
	make_request DELETE "/users/token" "${delete_token_data}" "${ADMIN_SESSION_TOKEN}"
else
	echo "Skipping test - Missing Admin Token or New Token ID."
fi

echo "=== API Tests Completed ==="

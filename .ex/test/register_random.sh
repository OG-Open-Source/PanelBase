#!/bin/bash

# Set the basic URL and port for the API
API_BASE_URL="http://localhost"
API_PORT="$1" # Get the port number from the script arguments, e.g., ./register_random.sh 8080
REGISTER_ENDPOINT="/api/v1/auth/register"
CONTENT_TYPE="application/json"

# Set the length of the username
USERNAME_LENGTH=10

# Check if the port number is provided
if [ -z "$API_PORT" ]; then
	echo "Please provide the API port number as the first argument."
	echo "Usage example: ./register_random.sh 8080"
	exit 1
fi

echo "Continuously trying to register random accounts until you press Ctrl+C..."

while true; do
	# Randomly generate a username (including letters and numbers)
	USERNAME=$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c "$USERNAME_LENGTH")

	# Randomly generate a password
	PASSWORD_LENGTH=12
	PASSWORD=$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c "$PASSWORD_LENGTH")

	# Randomly generate an email address
	EMAIL_PREFIX="random"
	RANDOM_NUMBER_EMAIL=$((RANDOM % 10000))
	EMAIL_DOMAIN="example.com"
	EMAIL="${EMAIL_PREFIX}${RANDOM_NUMBER_EMAIL}@${EMAIL_DOMAIN}"

	# Construct the JSON request body
	JSON_DATA=$(
		cat <<EOF
{
	"username": "${USERNAME}",
	"password": "${PASSWORD}",
	"email": "${EMAIL}"
}
EOF
	)

	# Use curl to send a POST request
	RESPONSE=$(curl -s -X POST "${API_BASE_URL}:${API_PORT}${REGISTER_ENDPOINT}" \
		-H "Content-Type: ${CONTENT_TYPE}" \
		-d "$JSON_DATA")

	echo "$(date '+%Y-%m-%d %H:%M:%S') - Attempting to register user: ${USERNAME}, response: $RESPONSE"

	# You can add a short delay to avoid making requests too frequently
	sleep 0.5
done

exit 0

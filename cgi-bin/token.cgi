#!/bin/bash

# Enable strict mode
set -euo pipefail

# Configuration
TOKEN_FILE="/var/lib/panelbase/tokens.json"
KEY_FILE="/var/lib/panelbase/keys.json"
MACHINE_ID_FILE="/var/lib/panelbase/machine_id"
TOKEN_VALIDITY_HOURS=24

# HTTP Headers
echo "Content-Type: application/json"
echo "Access-Control-Allow-Origin: https://panel.ogtt.tk"
echo "Access-Control-Allow-Methods: POST, OPTIONS"
echo "Access-Control-Allow-Headers: Content-Type, Authorization"
echo ""

# Handle OPTIONS request for CORS
if [ "$REQUEST_METHOD" = "OPTIONS" ]; then
    exit 0
fi

# Function to get system identifiers
get_system_ids() {
    local ids
    ids=$(cat "$KEY_FILE")
    echo "$ids"
}

# Function to generate new token
generate_token() {
    local length=32
    local machine_id=$(cat "$MACHINE_ID_FILE")
    local random_part=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c $length)
    local token="${machine_id:0:8}_${random_part}"
    local expiry=$(date -d "+${TOKEN_VALIDITY_HOURS} hours" +%s)
    
    # Create token entry
    local token_entry="{\"token\":\"$token\",\"created\":$(date +%s),\"expires\":$expiry,\"machine_id\":\"$machine_id\"}"
    
    # Update tokens file
    if [ ! -f "$TOKEN_FILE" ]; then
        echo "{\"tokens\":[$token_entry]}" > "$TOKEN_FILE"
    else
        # Remove expired tokens and add new one
        local current_time=$(date +%s)
        jq --arg current "$current_time" \
           --argjson new "$token_entry" \
           '.tokens = [.tokens[] | select(.expires > ($current | tonumber))] + [$new]' \
           "$TOKEN_FILE" > "${TOKEN_FILE}.tmp" && mv "${TOKEN_FILE}.tmp" "$TOKEN_FILE"
    fi
    
    echo "$token_entry"
}

# Function to validate keys
validate_keys() {
    local key1="$1"
    local key2="$2"
    local key3="$3"
    
    if [ ! -f "$KEY_FILE" ]; then
        echo '{"error":"Key file not found","status":500}'
        exit 1
    fi
    
    # Get stored system identifiers
    local stored_keys=$(get_system_ids)
    
    # Validate all three keys match system identifiers
    if ! echo "$stored_keys" | jq -e --arg k1 "$key1" --arg k2 "$key2" --arg k3 "$key3" \
        '.keys[] | select(.key1 == $k1 and .key2 == $k2 and .key3 == $k3)' > /dev/null; then
        echo '{"error":"Invalid keys","status":401}'
        exit 0
    fi
}

# Read POST data
read -n "$CONTENT_LENGTH" POST_DATA

# Extract keys from request
keys=$(echo "$POST_DATA" | jq -r '.keys // {}')
key1=$(echo "$keys" | jq -r '.key1 // ""')
key2=$(echo "$keys" | jq -r '.key2 // ""')
key3=$(echo "$keys" | jq -r '.key3 // ""')

# Validate keys
if [ -z "$key1" ] || [ -z "$key2" ] || [ -z "$key3" ]; then
    echo '{"error":"Missing keys","status":400}'
    exit 0
fi

validate_keys "$key1" "$key2" "$key3"

# Generate and return new token
token_data=$(generate_token)
echo "{\"status\":200,\"token\":$token_data}" 
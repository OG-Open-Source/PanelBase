#!/bin/bash

# Enable strict mode
set -euo pipefail

# Configuration
TOKEN_FILE="/var/lib/panelbase/tokens.json"
KEY_FILE="/var/lib/panelbase/keys.json"
MACHINE_ID_FILE="/var/lib/panelbase/machine_id"
RATE_LIMIT_DIR="/var/lib/panelbase/ratelimit"
MAX_REQUESTS_PER_MINUTE=60

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

# Function to validate security code
validate_security_code() {
    local security_code="$1"
    local stored_code
    
    if [ ! -f "$KEY_FILE" ]; then
        echo '{"error":"Security configuration not found","status":500}'
        exit 1
    fi
    
    stored_code=$(jq -r '.security_code' "$KEY_FILE")
    
    if [ "$security_code" != "$stored_code" ]; then
        echo '{"error":"Invalid security code","status":401}'
        exit 0
    fi
}

# Function to get system identifiers
get_system_ids() {
    local ids
    ids=$(cat "$KEY_FILE")
    echo "$ids"
}

# Function to check rate limit
check_rate_limit() {
    local ip_address="$1"
    local current_time=$(date +%s)
    local rate_file="${RATE_LIMIT_DIR}/${ip_address}"
    
    mkdir -p "$RATE_LIMIT_DIR"
    
    if [ -f "$rate_file" ]; then
        local count=0
        while IFS= read -r timestamp; do
            if [ $((current_time - timestamp)) -lt 60 ]; then
                ((count++))
            fi
        done < "$rate_file"
        
        if [ "$count" -ge "$MAX_REQUESTS_PER_MINUTE" ]; then
            echo '{"error":"Rate limit exceeded","status":429}'
            exit 0
        fi
    fi
    
    echo "$current_time" >> "$rate_file"
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

# Function to validate token
validate_token() {
    local token="$1"
    local machine_id=$(cat "$MACHINE_ID_FILE")
    
    if [ ! -f "$TOKEN_FILE" ]; then
        echo '{"error":"Token file not found","status":500}'
        exit 1
    fi
    
    # Validate token matches machine ID and is not expired
    if ! jq -e --arg token "$token" --arg mid "$machine_id" \
        '.tokens[] | select(.token == $token and .machine_id == $mid and .expires > now)' "$TOKEN_FILE" > /dev/null; then
        echo '{"error":"Invalid or expired token","status":401}'
        exit 0
    fi
}

# Read POST data
read -n "$CONTENT_LENGTH" POST_DATA

# Extract request data
keys=$(echo "$POST_DATA" | jq -r '.keys // {}')
token=$(echo "$POST_DATA" | jq -r '.token // ""')
command=$(echo "$POST_DATA" | jq -r '.command // ""')
security_code=$(echo "$POST_DATA" | jq -r '.security_code // ""')

# Check rate limit
check_rate_limit "$REMOTE_ADDR"

# Validate security code
if [ -n "$security_code" ]; then
    validate_security_code "$security_code"
fi

# Get panel user from machine ID
PANEL_USER="panel_$(head -c 12 "$MACHINE_ID_FILE")"

# Validate authentication
if [ -n "$token" ]; then
    validate_token "$token"
else
    key1=$(echo "$keys" | jq -r '.key1 // ""')
    key2=$(echo "$keys" | jq -r '.key2 // ""')
    key3=$(echo "$keys" | jq -r '.key3 // ""')
    
    if [ -z "$key1" ] || [ -z "$key2" ] || [ -z "$key3" ]; then
        echo '{"error":"Missing keys","status":400}'
        exit 0
    fi
    
    validate_keys "$key1" "$key2" "$key3"
fi

# Execute command with restricted privileges
if [ -n "$command" ]; then
    result=$(sudo -u "$PANEL_USER" bash -c "$command" 2>&1) || true
    echo "{\"output\":\"$result\",\"status\":200}"
else
    echo '{"error":"No command specified","status":400}'
fi 
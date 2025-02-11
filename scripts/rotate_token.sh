#!/bin/bash

# Enable strict mode
set -euo pipefail

# Configuration
TOKEN_FILE="/var/lib/panelbase/tokens.json"
SHARE_TOKEN_FILE="/var/lib/panelbase/share_tokens.json"
JWT_SECRET_FILE="/var/lib/panelbase/jwt_secret"

# Function to generate new JWT secret
generate_jwt_secret() {
    head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' > "$JWT_SECRET_FILE"
}

# Function to clean expired tokens
clean_expired_tokens() {
    local current_time=$(date +%s)
    
    # Clean main tokens
    if [ -f "$TOKEN_FILE" ]; then
        jq --arg current "$current_time" \
           '.tokens = [.tokens[] | select(.expires > ($current | tonumber))]' \
           "$TOKEN_FILE" > "${TOKEN_FILE}.tmp" && mv "${TOKEN_FILE}.tmp" "$TOKEN_FILE"
    fi
    
    # Clean share tokens
    if [ -f "$SHARE_TOKEN_FILE" ]; then
        jq --arg current "$current_time" \
           '.shares = [.shares[] | 
            select(
                (.token | split(".")[1] | @base64d | fromjson).expires > ($current | tonumber)
            )]' \
           "$SHARE_TOKEN_FILE" > "${SHARE_TOKEN_FILE}.tmp" && mv "${SHARE_TOKEN_FILE}.tmp" "$SHARE_TOKEN_FILE"
    fi
}

# Generate new JWT secret
generate_jwt_secret

# Clean expired tokens
clean_expired_tokens

# Set proper permissions
chown www-data:www-data "$JWT_SECRET_FILE" "$TOKEN_FILE" "$SHARE_TOKEN_FILE"
chmod 600 "$JWT_SECRET_FILE" "$TOKEN_FILE" "$SHARE_TOKEN_FILE" 
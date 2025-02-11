#!/bin/bash

# Enable strict mode
set -euo pipefail

# Configuration
TOKEN_FILE="/var/lib/panelbase/tokens.json"
SHARE_TOKEN_FILE="/var/lib/panelbase/share_tokens.json"
JWT_SECRET_FILE="/var/lib/panelbase/jwt_secret"

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

# Function to validate main token
validate_token() {
    local token="$1"
    
    if [ ! -f "$TOKEN_FILE" ]; then
        echo '{"error":"Token file not found","status":500}'
        exit 1
    fi
    
    if ! jq -e --arg token "$token" '.tokens[] | select(.token == $token and .expires > now)' "$TOKEN_FILE" > /dev/null; then
        echo '{"error":"Invalid or expired token","status":401}'
        exit 0
    fi
}

# Function to generate JWT
generate_jwt() {
    local payload="$1"
    local secret=$(cat "$JWT_SECRET_FILE")
    
    # Create JWT header
    local header='{"alg":"HS256","typ":"JWT"}'
    local header_base64=$(echo -n "$header" | base64 | tr -d '=' | tr '/+' '_-')
    
    # Create JWT payload
    local payload_base64=$(echo -n "$payload" | base64 | tr -d '=' | tr '/+' '_-')
    
    # Create signature
    local signature=$(echo -n "${header_base64}.${payload_base64}" | \
        openssl dgst -binary -sha256 -hmac "$secret" | \
        base64 | tr -d '=' | tr '/+' '_-')
    
    # Return complete JWT
    echo "${header_base64}.${payload_base64}.${signature}"
}

# Function to create share token
create_share_token() {
    local max_users="$1"
    local permissions="$2"
    local expiry="$3"
    
    local payload="{
        \"max_users\": $max_users,
        \"permissions\": $permissions,
        \"created\": $(date +%s),
        \"expires\": $expiry
    }"
    
    local jwt=$(generate_jwt "$payload")
    
    # Store share token info
    if [ ! -f "$SHARE_TOKEN_FILE" ]; then
        echo "{\"shares\":[{\"token\":\"$jwt\",\"current_users\":0}]}" > "$SHARE_TOKEN_FILE"
    else
        jq --arg jwt "$jwt" \
           '.shares += [{"token":$jwt,"current_users":0}]' \
           "$SHARE_TOKEN_FILE" > "${SHARE_TOKEN_FILE}.tmp" && mv "${SHARE_TOKEN_FILE}.tmp" "$SHARE_TOKEN_FILE"
    fi
    
    echo "$jwt"
}

# Read POST data
read -n "$CONTENT_LENGTH" POST_DATA

# Extract request data
token=$(echo "$POST_DATA" | jq -r '.token // ""')
action=$(echo "$POST_DATA" | jq -r '.action // ""')

# Validate main token
if [ -z "$token" ]; then
    echo '{"error":"Missing token","status":400}'
    exit 0
fi

validate_token "$token"

case "$action" in
    "create")
        max_users=$(echo "$POST_DATA" | jq -r '.max_users // 1')
        permissions=$(echo "$POST_DATA" | jq -r '.permissions // []')
        expiry=$(echo "$POST_DATA" | jq -r '.expiry // 0')
        
        if [ "$expiry" -eq 0 ]; then
            expiry=$(($(date +%s) + 3600)) # Default 1 hour
        fi
        
        share_token=$(create_share_token "$max_users" "$permissions" "$expiry")
        echo "{\"status\":200,\"share_token\":\"$share_token\"}"
        ;;
        
    "validate")
        share_token=$(echo "$POST_DATA" | jq -r '.share_token // ""')
        
        if [ -z "$share_token" ]; then
            echo '{"error":"Missing share token","status":400}'
            exit 0
        fi
        
        # Validate share token and update usage count
        if jq -e --arg token "$share_token" \
            '.shares[] | select(.token == $token and .current_users < fromjson($token | split(".")[1] | @base64d).max_users)' \
            "$SHARE_TOKEN_FILE" > /dev/null; then
            
            # Increment current users
            jq --arg token "$share_token" \
               '.shares = [.shares[] | if .token == $token then .current_users += 1 else . end]' \
               "$SHARE_TOKEN_FILE" > "${SHARE_TOKEN_FILE}.tmp" && mv "${SHARE_TOKEN_FILE}.tmp" "$SHARE_TOKEN_FILE"
               
            echo "{\"status\":200,\"valid\":true}"
        else
            echo "{\"status\":200,\"valid\":false}"
        fi
        ;;
        
    *)
        echo '{"error":"Invalid action","status":400}'
        ;;
esac 
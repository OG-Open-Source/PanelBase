#!/bin/bash

# Enable strict mode
set -euo pipefail

# Configuration
INSTALL_DIR="/opt/panelbase"
DATA_DIR="/var/lib/panelbase"
CGI_DIR="/usr/lib/cgi-bin"

# Generate machine-specific identifiers
KEY1=$(hostid 2>/dev/null || echo "")
KEY2=$(cat /etc/machine-id 2>/dev/null || echo "")
KEY3=$(dmidecode -s system-uuid 2>/dev/null || echo "")

# Validate that we have at least one valid identifier
if [ -z "$KEY1" ] && [ -z "$KEY2" ] && [ -z "$KEY3" ]; then
    echo "Error: Could not generate any system identifiers"
    exit 1
fi

# Use fallbacks if any key is empty
[ -z "$KEY1" ] && KEY1=$KEY2
[ -z "$KEY2" ] && KEY2=$KEY1
[ -z "$KEY3" ] && KEY3=$KEY1

# Generate panel user based on combined hash
COMBINED_HASH=$(echo "${KEY1}${KEY2}${KEY3}" | sha256sum | cut -d' ' -f1)
PANEL_USER="panel_${COMBINED_HASH:0:12}"

# Generate 8-digit security code
SECURITY_CODE=$(head -c 4 /dev/urandom | od -An -tu4 | tr -d ' ' | cut -c1-8)

# Install required packages
if command -v apt-get &> /dev/null; then
    apt-get update
    apt-get install -y lighttpd jq openssl
elif command -v yum &> /dev/null; then
    yum install -y lighttpd jq openssl
else
    echo "Error: Unsupported package manager"
    exit 1
fi

# Create directories
mkdir -p "$INSTALL_DIR"
mkdir -p "$DATA_DIR"
mkdir -p "$CGI_DIR"

# Create panel user
if ! id "$PANEL_USER" &>/dev/null; then
    useradd -r -s /bin/false "$PANEL_USER"
fi

# Configure lighttpd
cat > /etc/lighttpd/conf-available/10-cgi.conf << EOF
server.modules += ( "mod_cgi" )

\$HTTP["url"] =~ "^/cgi-bin/" {
    cgi.assign = ( ".cgi" => "" )
    alias.url += ( "/cgi-bin/" => "$CGI_DIR/" )
}

# Security headers
setenv.add-response-header = (
    "X-Frame-Options" => "DENY",
    "X-Content-Type-Options" => "nosniff",
    "X-XSS-Protection" => "1; mode=block",
    "Content-Security-Policy" => "default-src 'self'; connect-src *; script-src 'self' 'unsafe-inline' cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' cdn.jsdelivr.net; font-src 'self' cdn.jsdelivr.net;"
)

# CORS settings
setenv.add-response-header += (
    "Access-Control-Allow-Origin" => "https://panel.ogtt.tk",
    "Access-Control-Allow-Methods" => "POST, OPTIONS",
    "Access-Control-Allow-Headers" => "Content-Type, Authorization"
)

# Rate limiting (using mod_evasive)
evasive.max-conns-per-ip = 50
evasive.window-size = 10
EOF

# Enable CGI module
ln -sf /etc/lighttpd/conf-available/10-cgi.conf /etc/lighttpd/conf-enabled/
lighttpd-enable-mod cgi

# Copy files
cp -r cgi-bin/* "$CGI_DIR/"
cp -r scripts/* "$INSTALL_DIR/"

# Set up data directory
touch "$DATA_DIR/tokens.json"
touch "$DATA_DIR/share_tokens.json"
head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' > "$DATA_DIR/jwt_secret"

# Store the three keys and security code
cat > "$DATA_DIR/keys.json" << EOF
{
    "keys": [{
        "key1": "$KEY1",
        "key2": "$KEY2",
        "key3": "$KEY3"
    }],
    "security_code": "$SECURITY_CODE"
}
EOF

# Store combined hash for token generation
echo "$COMBINED_HASH" > "$DATA_DIR/machine_id"

# Set permissions
chown -R www-data:www-data "$DATA_DIR"
chmod 700 "$DATA_DIR"
chmod 600 "$DATA_DIR"/*

chown -R www-data:www-data "$CGI_DIR"
chmod 755 "$CGI_DIR"
chmod 755 "$CGI_DIR"/*

chown -R root:root "$INSTALL_DIR"
chmod 755 "$INSTALL_DIR"
chmod 755 "$INSTALL_DIR"/*

# Set up sudo permissions for panel user
cat > /etc/sudoers.d/panelbase << EOF
# Allow panel user to execute commands with elevated privileges
www-data ALL=($PANEL_USER) NOPASSWD: ALL
EOF
chmod 440 /etc/sudoers.d/panelbase

# Set up cron job for token rotation
cat > /etc/cron.d/panelbase << EOF
# Rotate tokens every hour
0 * * * * root $INSTALL_DIR/rotate_token.sh
EOF
chmod 644 /etc/cron.d/panelbase

# Restart lighttpd
systemctl restart lighttpd

echo "Installation completed successfully!"
echo "System Identifiers:"
echo "Key 1 (hostid): $KEY1"
echo "Key 2 (machine-id): $KEY2"
echo "Key 3 (system-uuid): $KEY3"
echo "Security Code: $SECURITY_CODE"
echo "Panel User: $PANEL_USER"
echo "Please configure your web server to handle CGI requests at $CGI_DIR"
echo "IMPORTANT: Save the security code. It will be required for panel access." 
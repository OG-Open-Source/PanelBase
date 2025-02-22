#!/bin/bash
set -e

get_port() {
	while true; do
		port=$(((RANDOM % (49151 - 1024 + 1)) + 1024))
		if ! nc -z localhost $port &>/dev/null; then
			echo $port
			return
		fi
		sleep 1
	done
}

get_entry() {
	head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9!@#$%^&*()_+-=' | head -c 16
}

if ! command -v nc &>/dev/null; then
	echo "Installing network tools..."
	apt update && apt install -y netcat-openbsd || yum install -y nc
fi

PORT=$(get_port)
ENTRANCE=$(get_entry)
SESSION_KEY=$(openssl rand -base64 32)

cat >.env <<EOF
PANELBASE_PORT=$PORT
PANELBASE_SECURITY_ENTRY=$ENTRANCE
PANELBASE_SESSION_KEY=$SESSION_KEY
EOF

(
	crontab -l 2>/dev/null
	echo "0 * * * * $(pwd)/ref_entry.sh"
) | crontab -

cat >ref_entry.sh <<'EOF'
#!/bin/bash
cd "$(dirname "$0")"

new_entrance=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9!@#$%^&*()_+-=' | head -c 16)
sed -i "s/PANELBASE_SECURITY_ENTRY=.*/PANELBASE_SECURITY_ENTRY=$new_entrance/" .env

pkill -HUP -f panelbase
EOF

chmod +x ref_entry.sh

echo "================================"
echo " Deployment completed"
echo " Proxy port: $PORT"
echo " Security entry: $ENTRANCE"
echo " Session key: $SESSION_KEY"
echo " Automatic refresh: Security entry updated every hour"
echo "================================"

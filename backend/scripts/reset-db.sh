#!/bin/bash
VPS_HOST=${VPS_HOST:-"relay.erenceh.dev"}

read -p "Are you sure you want to wipe the database? (y/N): " confirm
if [[ "$confirm" != "y" ]]; then
    echo "Aborted."
    exit 1
fi

echo "stopping service..."
ssh ubuntu@$VPS_HOST "sudo systemctl stop relay-go" || { echo "failed to stop service"; exit 1; }

echo "wiping db..."
ssh ubuntu@$VPS_HOST "docker-compose down -v" || { echo "failed to wipe db"; exit 1; }

echo "rebuilding db..."
ssh ubuntu@$VPS_HOST "docker-compose --env-file .env up -d" || { echo "failed to rebuild db"; exit 1; }

echo "restarting service...:"
ssh ubuntu@$VPS_HOST "sudo systemctl start relay-go" || { echo "failed to restart service"; exit 1; }

echo "done."
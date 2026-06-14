#!/bin/bash
VPS_HOST=${VPS_HOST:-"relay.erenceh.dev"}

echo "building..."
GOOS=linux GOARCH=amd64 go build -o relay-server ./cmd/server

echo "stopping service..."
ssh ubuntu@$VPS_HOST "sudo systemctl stop relay-go"

echo "copying to VPS..."
scp relay-server ubuntu@$VPS_HOST:/tmp/relay-server
ssh ubuntu@$VPS_HOST "sudo mv /tmp/relay-server /usr/local/bin/relay-server"

echo "restarting service...:"
ssh ubuntu@$VPS_HOST "sudo systemctl start relay-go"

echo "done."
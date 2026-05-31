# relay-go

A self-hosted real-time communication server built on raw TCP with a terminal client.

## Overview

`relay-go` is a real-time communication server written in Go, designed to run
on a self-hosted VPS or local machine. Clients connect over raw TCP using a
custom length-prefix framing protocol. The project is intentionally built close
to the metal; no frameworks, no managed messaging layers — to explore real
networking and systems programming.

The architecture is designed to scale from a modular monolith (v1) to a full
microservice mesh (v4+), with gRPC for internal service communication and
WebSocket support for browser clients planned in v5.

**This project is actively in development. A live demo will be available in v5.**

## Features (v2)

- JWT authentication with bcrypt password hashing
- Opaque refresh tokens with rotation
- Auto-reconnect with exponential backoff
- PostgreSQL message persistence
- Message history on room join
- Per-IP registration rate limiting (3/hour)
- Per-user message rate limiting (10 burst, 2/sec)
- Input validation on usernames and room names
- Connection timeouts (30s auth, 5min idle)

## Rate Limiting

- Registration: max 3 accounts per IP per hour
- Messages: 10 message burst, sustained 2 messages/second per user

## Planned

- gRPC-based microservice split (auth, messaging, presence)
- Message broker (NATS/Redis) for event-driven architecture
- API gateway with WebSocket transport for browser clients
- relay-web browser client

## Getting Started

### Requirements

- Go 1.25+
- Docker (for PostgreSQL)

### Run locally

```bash
# start PostgreSQL
docker compose --env-file .env up -d

# terminal 1
go run ./cmd/server

# terminal 2
go run ./cmd/client -addr localhost:8080
```

### Configuration

```bash
cp .env.example .env
```

### Available commands

/register Create a new account
/login Login to existing account
/join <room> Join a room
/leave Leave current room
/rooms List active rooms and members
/who List online users
/help Show available commands
/quit Disconnect

## Deployment

The live server is currently offline. It will be redeployed on AWS for v5 alongside the web client.

## License

MIT

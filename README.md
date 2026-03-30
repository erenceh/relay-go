# relay-go

A self-hosted real-time communication server built on raw TCP with a terminal client.

## Overview

`relay-go` is a real-time communication server written in Go, designed to run
on a self-hosted VPS or local machine. Clients connect over raw TCP using a
custom length-prefix framing protocol. The project is intentionally built close
to the metal; no frameworks, no managed messaging layers to explore real
networking and systems programming.

The architecture is designed to scale from a modular monolith (v1) to a full
microservice mesh (v4+), with gRPC for internal service communication and
WebSocket support for browser clients planned in v5.

**This project is still in development**

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

### Change Log

- Removed `/dm` from commands

## Rate Limiting

- Registration: max 3 accounts per IP per hour
- Messages: 10 message burst, sustained 2 messages/second per user

## Planned

- gRPC-based microservice split (auth, messaging, presence)
- Message broker (NATS/Redis) for event-driven architecture
- API gateway with WebSocket transport for browser clients
- relay-web browser client

## Architecture

```
relay-go/
├── cmd/
│   ├── server/         # Server entrypoint
│   └── client/         # CLI client entrypoint
├── db/
│   └── migrations/
├── internal/
│   ├── auth/           # Token/session logic (v2)
│   ├── db/             # database connection and migrations
│   ├── domain/         # User, Message, Room types
│   ├── messaging/      # Room routing
│   ├── presence/       # Online/offline tracking
│   ├── protocol/       # Message framing
│   ├── ratelimit/      # Rate limiting
│   ├── repository/     # PostgreSQL repositories
├── scripts/
│   └── deploy.sh       # VPS deployment script
├── go.mod
└── README.md
```

The server and CLI client communicate over raw TCP. Messages are framed using
a length-prefix protocol, each message is preceded by a 4-byte uint32 header
so the receiver knows exactly how many bytes to read. This avoids the partial
read problem inherent to TCP streams.

As features are added, internal packages will be extracted into standalone
microservices communicating over gRPC.

## Getting Started

### Requirements

- Go 1.25+
- Docker (for PostgreSQL)

### Connect to the public server

```bash
go run ./cmd/client
```

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

```
/register            Create a new account
/login               Login to existing account
/join <room>         Join a room
/leave               Leave current room
/rooms               List active rooms and members
/who                 List online users
/help                Show available commands
/quit                Disconnect
```

## Deployment

The public server runs on Oracle Cloud Free Tier at `relay.erenceh.dev:8080` managed as a systemd service.

## License

MIT

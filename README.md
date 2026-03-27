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

## Features (v1)

- [x] Project scaffold and folder structure
- [x] Length-prefix framing protocol (4-byte uint32 header)
- [x] Raw TCP server with goroutine-per-client concurrency
- [x] Connection registry with mutex-protected shared state
- [x] CLI client with concurrent read/write goroutines
- [x] Online/offline presence tracking with `/who`
- [x] Group chat rooms with `/join`, `/leave`, `/rooms`
- [x] Direct messaging with `/dm`
- [x] Graceful disconnect and room cleanup
- [x] Deployed on Oracle Cloud Free Tier VPS with systemd

## Planned

- User authentication (JWT) and message persistence (PostgreSQL)
- gRPC-based microservice split (auth, messaging, presence)
- Message broker (NATS/Redis) for event-driven architecture
- API gateway with WebSocket transport for browser clients
- Rust rewrite of presence service ([relay-rs](https://github.com/erenceh/relay-rs))

## Architecture

```
relay-go/
├── cmd/
│   ├── server/         # Server entrypoint
│   └── client/         # CLI client entrypoint
├── internal/
│   ├── auth/           # Token/session logic (v2)
│   ├── messaging/      # Room and DM routing
│   ├── presence/       # Online/offline tracking
│   └── protocol/       # Message framing and parsing
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

### Connect to the public server

```bash
go run ./cmd/client
```

### Run locally

```bash
# terminal 1
go run ./cmd/server

# terminal 2
go run ./cmd/client -addr localhost:8080
```

### Available commands

```
/register            Create a new account
/login               Login to existing account
/join <room>         Join a room
/leave               Leave current room
/dm <user> <msg>     Send a direct message
/rooms               List active rooms and members
/who                 List online users
/help                Show available commands
/quit                Disconnect
```

## Deployment

The public server runs on Oracle Cloud Free Tier at `relay.erenceh.dev:8080` managed as a systemd service.

## License

MIT

# relay-go

A self-hosted real-time chat server built on raw TCP with a terminal client.

## Overview

`relay-go` is a real-time chat server written in Go, designed to run on a self-hosted VPS or local machine. Clients connect over raw TCP using a lightweight text protocol. The project is intentionally built close to the metal — no frameworks, no managed messaging layers — to explore real networking programming.

## Features (v1)

- [x] Project scaffold and folder structure
- [ ] Raw TCP server and CLI client
- [ ] Direct messages between users
- [ ] Group chat rooms
- [ ] Online/offline presence tracking
- [ ] User authentication

## Planned

- WebSocket transport layer (browser client support)
- Message persistence (PostgreSQL)
- gRPC-based microservice split (auth, messaging, presence)
- Rust rewrite of presence service ([relay-rs](https://github.com/erenceh/relay-rs))

## Architecture

```
relay-go/
├── cmd/
│   ├── server/         # Server entrypoint
│   └── client/         # CLI client entrypoint
├── internal/
│   ├── auth/           # Token/session logic
│   ├── messaging/      # Room and DM routing
│   ├── presence/       # Online/offline tracking
│   └── protocol/       # Message framing and parsing
├── go.mod
└── README.md
```

The server and CLI client communicate over raw TCP. Messages are framed using a length-prefix protocol — each message is preceded by its byte length so the receiver knows exactly how much data to read.

As features are added, internal packages will be extracted into standalone microservices communicating over gRPC.

## Getting Started

### Requirements

- Go 1.22+

### Run the server

```bash
go run ./cmd/server
```

### Run the client

```bash
go run ./cmd/client
```

## Deployment

`relay-go` is designed to be self-hosted. The server runs on an Oracle Cloud Free Tier ARM VM with a single open TCP port. A second instance runs on a local home machine as a demonstration of self-hosted deployment.

## License

MIT

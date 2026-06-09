# relay-go

A self-hosted real-time communication server built on raw TCP with a terminal client.

## Overview

`relay-go` is a real-time communication server written in Go, designed to run
on a self-hosted VPS or local machine. Clients connect over raw TCP using a
custom length-prefix framing protocol. The project is intentionally built close
to the metal; no frameworks, no managed messaging layers - to explore real
networking and systems programming.

The architecture is designed to scale from a modular monolith (v1) to a full
microservice mesh (v4+), with gRPC for internal service communication and
WebSocket support for browser clients planned in v5.

**This project is actively in development. A live demo will be available in v5.**

## Features (v3)

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

## Running Locally

### Requirements

- Go 1.25+
- Docker

### Start infrastructure

```bash
docker compose --env-file .env.db up -d
```

### Start services (each in a separate terminal)

```bash
go run ./cmd/auth-service
go run ./cmd/messaging-service
go run ./cmd/presence-service
go run ./cmd/server
```

### Connect a client

```bash
go run ./cmd/client -addr localhost:8080
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

## Failure Modes

### Auth Service Unavailable

relay-go runs as two separate binaries - `cmd/server` and `cmd/auth-service`. If the auth service becomes unavailable, the following behavior applies:

**During startup**
If the auth service is unreachable when `cmd/server` starts, the messaging server will still start successfully. The gRPC connection is established lazily - it is not verified until the first auth call is made.

**During authentication**
If a client attempts to register, login, or refresh while the auth service is down, `cmd/server` will retry the gRPC call up to 3 times with exponential backoff starting at 1 second. If all attempts fail, the client receives an error message and is prompted to try again. No connection is dropped - the client remains connected and can retry.

**Database unavailable**
If PostgreSQL becomes unreachable, the auth service will fail to process any requests and return errors to the messaging server. The messaging server will surface these as auth failures to the client.

### Recovery

Both services log warnings on each failed retry attempt and an error when retries are exhausted. Monitor logs for `auth service call failed` and `max retry attempts reached` to detect and diagnose connectivity issues between services.

## Deployment

The live server is currently offline. It will be redeployed on AWS for v5 alongside the web client.

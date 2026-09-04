# In-Memory Database

A Redis-like in-memory database server built from scratch in Go.

The goal of this project is to understand how a networked database works internally — from TCP connections and raw bytes, through RESP parsing and command execution, to concurrency, persistence, testing, and cloud deployment.

## Features

- TCP server on port `6379`
- RESP protocol parsing
- TCP stream and fragmentation handling
- Multiple commands in a single TCP read
- Concurrent client connections
- In-memory string storage
- In-memory hash storage
- Command validation
- AOF (Append-Only File) persistence
- Startup recovery from AOF
- Periodic AOF synchronization
- Graceful shutdown
- Docker deployment
- Persistent Docker volume
- Public TCP deployment
- Automated tests
- Go race detector
- GitHub Actions CI

## Supported Commands

### Strings

```text
PING
SET key value
GET key
```

### Hashes

```text
HSET key field value
HGET key field
HGETALL key
HDEL key field
```

### Database

```text
FLUSHDB
```

## Architecture

```text
                    Client
                      │
                      │ TCP / RESP
                      ▼
               TCP Server :6379
                      │
                      ▼
                RESP Parser
                      │
                      ▼
              Command Handler
                 │         │
          ┌──────┘         └──────┐
          ▼                       ▼
   In-Memory Store             AOF
   Strings + Hashes             │
          │                      ▼
          │                  Disk File
          │
          └──────────────► RESP Response
```

## TCP and RESP

TCP is a byte stream rather than a message protocol.

A single command can arrive across multiple network reads. The server therefore maintains an accumulated buffer until a complete RESP value can be parsed.

The parser returns:

1. The parsed `Value`
2. The number of bytes consumed
3. An error

This also allows multiple RESP values to exist inside the same TCP read.

## Concurrency

Each accepted TCP connection is handled independently using a goroutine.

Shared database state is protected using Go mutexes.

```text
Client 1 ──┐
Client 2 ──┼──► TCP Server
Client 3 ──┘       │
                   ▼
              Command Handler
                   │
              ┌────┴────┐
              ▼         ▼
           Strings     Hashes
              │         │
             RWMutex   RWMutex
```

The project is tested with Go's race detector:

```bash
go test -race ./...
```

## Persistence

The database uses an Append-Only File (AOF) for persistence.

Writable commands are appended to the AOF:

```text
SET
HSET
HDEL
```

When the server starts, the AOF is replayed through the command handler to reconstruct the in-memory state.

The AOF is periodically synchronized to disk and synchronized again during shutdown.

### Persistence Flow

```text
Client
  │
  ▼
Command
  │
  ├──────────────► In-Memory Store
  │
  └──────────────► AOF
                       │
                       ▼
                    Disk
                       │
                 Server restart
                       │
                       ▼
                  AOF replay
                       │
                       ▼
                Reconstructed state
```

## Persistence Verification

Persistence was verified on the deployed Docker instance.

Before restarting the container:

```text
SET test survives
HSET user status alive

GET test
→ "survives"

HGET user status
→ "alive"
```

The container was then restarted:

```bash
sudo docker restart in-memory-db
```

The client connection closed as expected.

After reconnecting:

```text
GET test
→ "survives"

HGET user status
→ "alive"
```

This verified:

```text
Remote write
    ↓
AOF
    ↓
Docker container restart
    ↓
AOF replay
    ↓
Data recovered
    ↓
Remote read
```

## Running Locally

### Requirements

- Go
- `redis-cli` or another RESP-compatible client
- Docker (optional)

Run directly:

```bash
go run .
```

The server listens on:

```text
localhost:6379
```

Test:

```bash
redis-cli -p 6379
```

Then:

```text
PING
SET name akash
GET name
```

## Docker

Build the image:

```bash
docker build -t in-memory-db:latest .
```

Run:

```bash
docker run -d \
  --name in-memory-db \
  -p 6379:6379 \
  -v in-memory-db-data:/app/data \
  --restart unless-stopped \
  in-memory-db:latest
```

The Docker volume keeps the AOF outside the container filesystem, allowing database state to survive container recreation.

## Public Deployment

The database is deployed on a VyuhStack Compute Sandbox VM using Docker.

```text
Debian 12
   │
   ▼
Docker Engine
   │
   ▼
in-memory-db container
   │
   ▼
TCP :6379
```

### Cloud Networking Challenge

The VM platform provides public application access through an HTTPS application-port path.

The database communicates using raw TCP/RESP on port `6379`.

Therefore, the database could not simply be exposed through the normal HTTP application endpoint.

Instead, a TCP tunneling layer was used without changing the database protocol or server architecture.

```text
Internet
   │
   │ TCP
   ▼
Portwarp
   │
   ▼
VyuhStack VM
   │
   ▼
Docker :6379
   │
   ▼
In-Memory Database
```

This preserved the server's native TCP/RESP interface on port `6379`.

### Connecting to the Public Database

The active public endpoint is provided by the Portwarp tunnel.

```bash
redis-cli -h elmbv279.free.pwrp.cc -p 12173
```

Example:

```text
PING
→ PONG

SET cloud hello
→ OK

GET cloud
→ "hello"
```

Remote hash operations were also verified:

```text
HSET user name aks
→ OK

HGET user name
→ "aks"

HSET user age 21
→ OK

HGETALL user
→ name / aks
→ age / 21
```

> The public endpoint may change when the tunnel is recreated. Use the currently assigned Portwarp endpoint.

### Public Deployment Flow

```text
Laptop
   │
   │ TCP / RESP
   ▼
Public TCP Endpoint
   │
   ▼
Portwarp TCP Tunnel
   │
   ▼
VyuhStack VM
   │
   ▼
Docker Container
   │
   ▼
Go Database :6379
```

## Deployment Environment

### VM

- VyuhStack Compute Sandbox
- Debian 12
- 1 vCPU
- 1 GB RAM
- 10 GB storage

### Container

- Docker Engine
- `in-memory-db:latest`
- Port `6379`
- Persistent Docker volume
- `restart unless-stopped`

### Public Networking

- Portwarp
- TCP tunnel
- Local port `6379`
- Tunnel configured for automatic reconnect on VM startup

## Testing

Run all tests:

```bash
go test ./...
```

Run with the race detector:

```bash
go test -race ./...
```

Tests cover:

- RESP parsing
- TCP server behavior
- Multiple commands in one read
- Fragmented TCP commands
- Multiple simultaneous clients
- Command execution
- Persistence behavior

## CI

GitHub Actions automatically runs:

```bash
go test ./...
go test -race ./...
```

on pushes and pull requests.

## Engineering Challenges

### TCP Message Boundaries

TCP does not preserve application-level message boundaries.

The server handles both complete and fragmented RESP messages while preserving unconsumed bytes for subsequent reads.

### Concurrent Access

Multiple clients can access the same in-memory data simultaneously.

Separate read/write locks protect the string and hash stores.

### Persistence

The in-memory state cannot survive a process restart by itself.

A custom AOF layer was introduced so commands can be replayed when the server starts.

### Cloud Networking

The selected VM platform did not directly expose a raw public TCP port for the database.

Rather than changing the database protocol or architecture, a TCP tunneling layer was added externally.

This preserved the server's native TCP/RESP interface on port `6379`.

## Project Status

- [x] TCP server
- [x] RESP parser
- [x] TCP fragmentation handling
- [x] Multiple commands per read
- [x] String commands
- [x] Hash commands
- [x] Command validation
- [x] Concurrency protection
- [x] AOF persistence
- [x] AOF recovery
- [x] Periodic synchronization
- [x] Graceful shutdown
- [x] Unit/integration testing
- [x] Race testing
- [x] GitHub Actions CI
- [x] Dockerization
- [x] Persistent Docker volume
- [x] Cloud VM deployment
- [x] Public TCP access
- [x] Remote database testing
- [x] Persistence verification after container restart

## Possible Future Extensions

- TTL / key expiration
- More Redis commands
- Better command dispatch architecture
- Improved error responses
- More extensive AOF crash-recovery tests
- Benchmarks
- Authentication
- More RESP features
- Performance improvements

## Learning Goals

This project is being built to understand:

- TCP networking in Go
- Stream-oriented protocols
- RESP
- Byte buffers
- Protocol parsing
- Command execution
- In-memory data structures
- File persistence
- Concurrency
- Synchronization
- Testing
- Docker
- Cloud deployment
- Systems programming

The emphasis is on understanding why each layer exists and how the pieces interact.

## Tech Stack

- **Language:** Go
- **Networking:** Go `net`
- **Protocol:** RESP
- **Storage:** Custom in-memory data structures
- **Persistence:** Custom AOF
- **Containerization:** Docker
- **Deployment:** VyuhStack VM
- **TCP tunneling:** Portwarp
- **CI:** GitHub Actions
- **Testing:** Go testing + race detector

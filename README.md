# in-memory_db

A Redis-like in-memory database server built from scratch in Go.

The goal is to understand how a networked database works internally — from accepting TCP connections and reading raw bytes to parsing RESP, executing commands, storing data, and eventually persisting it to disk.

## Current Progress

- TCP server listening on port `6379`
- Client connection handling
- Reading raw data from TCP connections
- RESP parser
- Simple Strings (`+`)
- Integers (`:`)
- Bulk Strings (`$`)
- Arrays (`*`)
- Incomplete-data handling
- TCP fragmentation handling
- Persistent accumulation of bytes across reads
- Tracking consumed bytes
- Removing consumed bytes from the input buffer
- Successfully parsing fragmented commands such as:

```text
*1\r\n
$4\r\n
PING\r\n
```

into a structured `Value`.

## Architecture

```text
Client
  │
  │ TCP connection
  ▼
TCP Server
  │
  │ raw bytes
  ▼
Read Buffer
  │
  │ accumulate bytes
  ▼
RESP Parser
  │
  │ Value
  ▼
Command Execution
  │
  ▼
RESP Response
  │
  ▼
Client
```

### Important TCP concept

TCP is a **byte stream**, not a message protocol.

One Redis command may arrive across multiple reads:

```text
READ #1 → "*1\r\n"
READ #2 → "$4\r\n"
READ #3 → "PING\r\n"
```

The server therefore keeps incomplete data until enough bytes have arrived to parse a complete RESP value.

## RESP Parser

The parser has this shape:

```go
func parse(data []byte) (Value, int, error)
```

It returns:

1. The parsed `Value`
2. The number of bytes consumed
3. An error

The parser uses a shared incomplete-data error:

```go
var errIncomplete = errors.New("Incomplete Data")
```

This lets the server distinguish:

```text
Incomplete data
      ↓
wait for another read
```

from:

```text
Invalid data
      ↓
actual parsing error
```

## Buffering

The server maintains a temporary network buffer and an accumulated data buffer:

```go
buffer := make([]byte, 1024)
data := make([]byte, 0)
```

After each read:

```go
data = append(data, buffer[:n]...)
```

When parsing succeeds:

```go
data = data[consumed:]
```

Only the bytes belonging to the parsed value are removed.

This allows multiple RESP values to exist in one TCP read.

For example:

```text
+hello\r\n+world\r\n
```

can be parsed as:

```text
Value 1 → +hello
Value 2 → +world
```

## Running the Server

Run:

```bash
go run .
```

The server listens on:

```text
localhost:6379
```

Expected output:

```text
Server is listening on port 6379...
```

## Testing

### Simple String

```bash
printf '+hello\r\n' | nc localhost 6379
```

### RESP PING command

```bash
printf '*1\r\n$4\r\nPING\r\n' | nc localhost 6379
```

### Deliberately fragmented RESP command

```bash
(printf '*1\r\n'; sleep 1; printf '$4\r\n'; sleep 1; printf 'PING\r\n') | nc localhost 6379
```

The server should accumulate the pieces and eventually parse the complete command.

## Current Limitation

The server can currently **parse** a Redis-style command, but it does not yet execute `PING` and return:

```text
+PONG\r\n
```

The current pipeline ends around:

```text
raw TCP bytes
      ↓
RESP parser
      ↓
Value
      ↓
printed to terminal
```

The next layer is:

```text
Value
  ↓
extract command
  ↓
execute command
  ↓
create RESP response
```

## Roadmap

### Phase 1 — Networking

- [x] TCP server
- [x] Listen on port `6379`
- [x] Accept connections
- [x] Read from connections
- [x] Write responses
- [x] Connection cleanup

### Phase 2 — RESP Protocol

- [x] Simple Strings
- [x] Integers
- [x] Bulk Strings
- [x] Arrays
- [x] Incomplete-data handling
- [x] TCP fragmentation handling
- [x] Multiple values in the input buffer

### Phase 3 — Command Execution

- [ ] Extract command name
- [ ] Implement `PING`
- [ ] Implement `ECHO`
- [ ] Implement `SET`
- [ ] Implement `GET`
- [ ] Implement command errors
- [ ] Return proper RESP responses

### Phase 4 — In-Memory Database

- [ ] Create key/value storage
- [ ] Store strings
- [ ] Retrieve strings
- [ ] Handle missing keys
- [ ] Handle overwrites
- [ ] Add expiration/TTL support

### Phase 5 — Persistence

- [ ] Design on-disk format
- [ ] Write database state to disk
- [ ] Load state when the server starts
- [ ] Handle persistence safely

### Phase 6 — Production Improvements

- [ ] Command dispatch
- [ ] Concurrency improvements
- [ ] Robust error handling
- [ ] Automated tests
- [ ] Benchmarks
- [ ] Graceful shutdown
- [ ] Broader RESP support

## Learning Goals

This project is being built to understand:

- TCP networking in Go
- Byte slices and buffers
- Stream-oriented protocols
- Protocol parsing
- Error handling
- Nested data structures
- Command dispatch
- In-memory data structures
- Database persistence
- Concurrency
- Testing
- Systems programming

The emphasis is on understanding **why each layer exists**, rather than simply copying an implementation.

## Tech Stack

- **Language:** Go
- **Networking:** Go `net` package
- **Protocol:** RESP
- **Testing:** `netcat`
- **Database:** Custom in-memory implementation
- **Persistence:** Planned custom disk format

## Project Status

**Early development — TCP networking and RESP parsing are working.**

The server can receive fragmented RESP data over TCP, accumulate it correctly, parse complete values, and track consumed bytes.

### Next milestone

Turn:

```text
*1\r\n$4\r\nPING\r\n
```

into:

```text
+PONG\r\n
```

and make the first real Redis command work.

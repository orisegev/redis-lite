# redis-lite

A lightweight in-memory key-value store written in Go, inspired by Redis. Supports TCP connections, password authentication, key expiry (TTL), and a background cleanup goroutine that evicts expired keys automatically.

## Features

- `SET` / `GET` / `DEL` / `KEYS` — core key-value operations
- `SET key value EX <seconds>` — optional per-key TTL
- `TTL` — inspect remaining lifetime of a key
- `AUTH` — password-based authentication
- Background expiry sweep — expired keys are removed from memory every second, even without a read
- Graceful shutdown on `SIGINT` / `SIGTERM`
- Docker support

## Getting Started

### Requirements

- Go 1.22+  **or** Docker

### Run with Go

```bash
# clone
git clone https://github.com/orisegev/redis-lite.git
cd redis-lite

# configure (optional — defaults shown)
cp .env.example .env   # edit PORT and AUTH_PASSWORD

# run
go run ./cmd/server/
```

### Run with Docker

```bash
docker compose up --build
```

The server listens on port `6379` by default.

### Configuration

| Variable        | Default           | Description              |
|-----------------|-------------------|--------------------------|
| `PORT`          | `6379`            | TCP port to listen on    |
| `AUTH_PASSWORD` | `defaultpassword` | Password for `AUTH`      |

Set these in a `.env` file or as environment variables.

## Usage

Connect with any TCP client, e.g. `netcat`:

```bash
nc localhost 6379
```

Every connection must authenticate before issuing commands.

---

### AUTH

```
AUTH <password>
```

```
> AUTH mypassword
OK

> AUTH wrongpassword
ERR invalid password
```

---

### SET

Store a key. Add `EX <seconds>` to set an expiry.

```
SET <key> <value>
SET <key> <value> EX <seconds>
```

```
> SET name redis-lite
OK

> SET session abc123 EX 60
OK
```

---

### GET

Retrieve a key. Returns `(nil)` if the key doesn't exist or has expired.

```
GET <key>
```

```
> GET name
redis-lite

> GET session
abc123

> GET unknown
(nil)
```

---

### TTL

Check how many seconds are left on a key.

| Return value | Meaning                     |
|--------------|-----------------------------|
| `>= 0`       | Seconds remaining           |
| `-1`         | Key exists but has no expiry|
| `-2`         | Key does not exist / expired|

```
TTL <key>
```

```
> TTL session
(integer) 47

> TTL name
(integer) -1

> TTL unknown
(integer) -2
```

---

### DEL

Delete a key.

```
DEL <key>
```

```
> DEL name
(integer) 1

> DEL name
(integer) 0
```

---

### KEYS

List all keys currently in the store (excludes expired keys).

```
KEYS
```

```
> KEYS
1) session
2) user:1
```

---

### EXIT

Close the connection.

```
> EXIT
Goodbye!
```

---

## Full Session Example

```
$ nc localhost 6379
Connected to redis-lite. Authenticate with AUTH <password>

> AUTH mypassword
OK

> SET user:1 alice
OK

> SET token:1 xyz EX 30
OK

> KEYS
1) user:1
2) token:1

> GET user:1
alice

> TTL token:1
(integer) 28

> TTL user:1
(integer) -1

> DEL user:1
(integer) 1

> GET user:1
(nil)

> EXIT
Goodbye!
```

## Development

```bash
make build   # compile to bin/server
make run     # go run ./cmd/server/
make test    # go test ./...
make fmt     # gofmt -w .
make lint    # go vet ./...
make clean   # remove bin/
```

## Project Structure

```
cmd/server/          # entry point — wires config, server, graceful shutdown
internal/
  config/            # loads PORT and AUTH_PASSWORD from env
  server/            # TCP listener, connection handler, command dispatcher
  storage/           # in-memory map with RWMutex, TTL, background cleanup
Dockerfile
docker-compose.yml
Makefile
```

# go-faas - Documentation

> Back to [README](../README.md)

## Prerequisites

- Go 1.23 or higher
- Linux operating system (Ubuntu, Debian, Fedora, Arch Linux, Alpine Linux)
- Redis server
- Bubblewrap (`bwrap`)
- Node.js (with npm)
- Python 3
- systemd (for slice resource control)

## Installation

### From Source

```bash
git clone https://github.com/pardnchiu/go-faas.git
cd go-faas
go build -o go-faas cmd/api/main.go
```

### Using go install

```bash
go install github.com/pardnchiu/go-faas/cmd/api@latest
```

### Install TypeScript Dependencies

```bash
npm install
```

> On first launch, the program automatically checks whether `bwrap`, `node`, and `python3` are present. If any are missing, it attempts to install them via the system package manager.

## Configuration

### Environment Variables

Copy `.env.example` and fill in the values:

```bash
cp .env.example .env
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `HTTP_PORT` | No | `8080` | HTTP server port |
| `MAX_CPUS` | No | `1` | Sandbox CPU quota (cores) |
| `MAX_MEMORY` | No | `128M` | Sandbox memory ceiling |
| `CODE_MAX_SIZE` | No | `262144` (256KB) | Maximum allowed code size in bytes |
| `TIMEOUT_SCRIPT` | No | `30` | Script execution timeout in seconds |
| `REDIS_HOST` | No | `localhost` | Redis host address |
| `REDIS_PORT` | No | `6379` | Redis port |
| `REDIS_PASSWORD` | No | empty | Redis password |
| `REDIS_DB` | No | `0` | Redis database number |
| `REDIS_TIMEOUT_SECONDS` | No | `5` | Redis connection timeout in seconds |

## Usage

### Start the Server

```bash
./go-faas
```

### Upload a Script

Store a script in Redis and receive a version number:

```bash
curl -X POST http://localhost:8080/upload \
  -H "Content-Type: application/json" \
  -d '{
    "path": "math/add",
    "language": "python",
    "code": "return event.get(\"a\", 0) + event.get(\"b\", 0)"
  }'
```

Response:

```json
{
  "path": "math/add",
  "language": "python",
  "version": 1739000000
}
```

### Execute a Stored Script

Run the latest version by path:

```bash
curl -X POST http://localhost:8080/run/math/add \
  -H "Content-Type: application/json" \
  -d '{
    "input": "{\"a\": 3, \"b\": 5}"
  }'
```

Run a specific version:

```bash
curl -X POST "http://localhost:8080/run/math/add?version=1739000000" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "{\"a\": 3, \"b\": 5}"
  }'
```

Response:

```json
{
  "data": 8,
  "type": "number"
}
```

### Execute Code Immediately

Submit code for direct execution without storing:

```bash
curl -X POST http://localhost:8080/run-now \
  -H "Content-Type: application/json" \
  -d '{
    "language": "javascript",
    "code": "return { sum: event.a + event.b }",
    "input": "{\"a\": 10, \"b\": 20}"
  }'
```

Response:

```json
{
  "data": { "sum": 30 },
  "type": "json"
}
```

### SSE Streaming Mode

Set `stream: true` to enable Server-Sent Events streaming output:

```bash
curl -X POST http://localhost:8080/run-now \
  -H "Content-Type: application/json" \
  -d '{
    "language": "python",
    "code": "import time\nfor i in range(5):\n    print(i)\n    time.sleep(0.5)\nreturn \"done\"",
    "input": "{}",
    "stream": true
  }'
```

Streaming response format:

```
data: {"event":"log","data":"0","type":"text"}

data: {"event":"log","data":"1","type":"text"}

data: {"event":"result","data":"done","type":"string"}
```

## API Reference

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/upload` | Upload a script to Redis |
| `POST` | `/run/*targetPath` | Execute a stored script |
| `POST` | `/run-now` | Execute submitted code immediately |

### POST /upload

Upload and store a script in Redis, returning a version number.

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | `string` | Yes | Script access path (must not contain `..`) |
| `code` | `string` | Yes | Code content |
| `language` | `string` | Yes | Language (`python`, `javascript`, `typescript`) |

**Response:**

```json
{
  "path": "string",
  "language": "string",
  "version": 1739000000
}
```

### POST /run/*targetPath

Fetch a script from Redis and execute it inside the sandbox.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `version` | `int64` | No | Target version number; defaults to latest |

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `input` | `string` | No | JSON-formatted input data, accessible as `event` in the script |
| `stream` | `bool` | No | Enable SSE streaming output |

### POST /run-now

Submit code for direct sandbox execution without Redis storage.

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `code` | `string` | Yes | Code content |
| `language` | `string` | Yes | Language (`python`, `javascript`, `typescript`) |
| `input` | `string` | No | JSON-formatted input data |
| `stream` | `bool` | No | Enable SSE streaming output |

### Response Format

Standard responses auto-detect the return data type:

| `type` | Description |
|--------|-------------|
| `string` | String value |
| `number` | Numeric value |
| `json` | JSON object or array |
| `text` | Plain text (not valid JSON) |

### SSE Event Format

| `event` | Description |
|---------|-------------|
| `log` | Intermediate script output (`print` / `console.log`) |
| `result` | Final execution result |
| `error` | Execution error message |

### Supported Languages

| Language | Runtime | Extension | Global Variables Available in Script |
|----------|---------|-----------|--------------------------------------|
| Python | `python3` | `.py` | `event`, `input` |
| JavaScript | `node` | `.js` | `event`, `input` |
| TypeScript | `tsx` | `.ts` | `event`, `input` |

***

©️ 2025 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)

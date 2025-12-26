![cover](./cover.png)

> [!NOTE]
> This README was translated by ChatGPT 4.1, get the original version from [here](./README.zh.md).

# Go FaaS

[![pkg](https://pkg.go.dev/badge/github.com/pardnchiu/go-faas.svg)](https://pkg.go.dev/github.com/pardnchiu/go-faas)
[![card](https://goreportcard.com/badge/github.com/pardnchiu/go-faas)](https://goreportcard.com/report/github.com/pardnchiu/go-faas)
[![version](https://img.shields.io/github/v/tag/pardnchiu/go-faas?label=release)](https://github.com/pardnchiu/go-faas/releases)
[![license](https://img.shields.io/github/license/pardnchiu/go-faas)](LICENSE)

> Lightweight Golang FaaS platform providing isolated execution environments for JavaScript, TypeScript, and Python scripts. Supports real-time execution and version management, using Podman/Docker containers to protect host security.

- [Core Features](#core-features)
  - [Multi-language Support](#multi-language-support)
  - [Container Isolation](#container-isolation)
  - [Smart Management](#smart-management)
- [System Architecture](#system-architecture)
- [Dependencies](#dependencies)
- [Requirements](#requirements)
- [Usage](#usage)
  - [Installation](#installation)
  - [Start Service](#start-service)
  - [Container Configuration](#container-configuration)
- [API](#api)
  - [Upload Script](#upload-script)
  - [Run Uploaded Script](#run-uploaded-script)
  - [Run Script Directly](#run-script-directly)
- [Script Types](#script-types)
  - [JavaScript](#javascript)
  - [TypeScript](#typescript)
  - [Python](#python)
- [Configuration](#configuration)
- [License](#license)
- [Author](#author)
- [Star](#star)

## Core Features

### Multi-language Support
Supports execution of JavaScript, TypeScript, and Python scripts, with unified JSON format for parameter passing and result return.

### Container Isolation
Uses Podman container pool to isolate execution environments, protecting the host system. Each request runs in an independent container to avoid interference.

### Smart Management
Automatically detects container health and rebuilds unhealthy containers. Dynamic container pool management and auto-release mechanism ensure high availability.

## System Architecture

```mermaid
flowchart TD
  A[HTTP Request] --> B{Route Dispatch}
  B -->|POST /upload| C[Script Upload]
  B -->|POST /run/*| D[Run Saved Script]
  B -->|POST /run-now| E[Run Immediately]
  
  C --> F[Validate Script]
  F --> G[Redis Version Storage]
  
  D --> H[Get Script from Redis]
  H --> I[Get from Container Pool]
  
  E --> I
  
  I --> J{Container Status Check}
  J -->|Healthy| K[Run Script]
  J -->|Unhealthy| L[Rebuild Container]
  L --> K
  
  K --> M[Return Result]
  M --> N[Return to Container Pool]
  
  O[Health Check Goroutine] -.->|Periodic Check| P[Container Status]
  P -.->|Abnormal| L
```

## Dependencies

- [`github.com/gin-gonic/gin`](https://github.com/gin-gonic/gin)
- [`github.com/redis/go-redis/v9`](https://github.com/redis/go-redis)
- [`github.com/joho/godotenv`](https://github.com/joho/godotenv)

## Requirements

- Go 1.23.0+
- Podman
- Redis 6.0+

## Usage

### Installation

```bash
# Clone the project
git clone https://github.com/pardnchiu/go-faas.git
cd go-faas

# Install dependencies
go mod download
```

### Start Service

```bash
# Start Redis (required)
podman run -d --name redis -p 6379:6379 redis:alpine

# Start service
go run cmd/api/main.go
```

### Container Configuration

```env
MAX_CONTAINERS=4
GPU_ENABLED=false     # Set to true if Nvidia GPU is available
HTTP_PORT=8080

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

## API

### Upload Script

- POST: `/upload` 
- Supported languages: 
  - `javascript`
  - `typescript`
  - `python`
- Request example
  ```json
  {
    "path": "test/calculator",
    "language": "javascript",
    "code": "console.log(JSON.stringify({ sum: event.a + event.b }));"
  }
  ```
- Response example
  ```json
  {
    "path": "test/calculator",
    "language": "javascript",
    "version": 1735286400000
  }
  ```

### Run Uploaded Script
- POST: `/run/{path}`
- Parameters
  - `version` (optional): Specify script version timestamp, default is latest version
- Request example
  ```json
  // Example for `/run/test/calculator`
  {
    "a": 10,
    "b": 5
  }
  ```
- Response example
  ```json
  {
    "sum": 15
  }
  ```

### Run Script Directly
- POST: `/run-now`
- Request example
  ```json
  {
    "language": "python",
    "code": "import json\nresult = {'sum': event['a'] + event['b']}\nprint(json.dumps(result))",
    "input": "{\"a\": 10, \"b\": 5}"
  }
  ```
- Response example
  ```json
  {
    "output": {
    "sum": 15
    },
    "type": "json"
  }
  ```

## Script Types

> [!NOTE]
> All scripts receive input data via the `event` variable

### JavaScript

```javascript
// Input is provided via the event variable
const result = {
  sum: event.a + event.b,
  product: event.a * event.b
};
console.log(JSON.stringify(result));
```

### TypeScript

```typescript
interface Event {
  a: number;
  b: number;
}

const result = {
  sum: event.a + event.b,
  product: event.a * event.b
};
console.log(JSON.stringify(result));
```

### Python

```python
import json

result = {
  'sum': event['a'] + event['b'],
  'product': event['a'] * event['b']
}
print(json.dumps(result))
```

## Configuration

Timeout
- Script execution: 30 seconds

Request limits
- `/run/*`: max 10 MB
- `/run-now`: max 5 MB

## License

This project is licensed under [MIT](LICENSE).

## Author

<img src="https://avatars.githubusercontent.com/u/25631760" align="left" width="96" height="96" style="margin-right: 0.5rem;">

<h4 style="padding-top: 0">邱敬幃 Pardn Chiu</h4>

<a href="mailto:dev@pardn.io" target="_blank">
  <img src="https://pardn.io/image/email.svg" width="48" height="48">
</a> <a href="https://linkedin.com/in/pardnchiu" target="_blank">
  <img src="https://pardn.io/image/linkedin.svg" width="48" height="48">
</a>

## Star

[![Star](https://api.star-history.com/svg?repos=pardnchiu/go-faas&type=Date)](https://www.star-history.com/#pardnchiu/go-faas&Date)

***

©️ 2025 [邱敬幃 Pardn Chiu](https://pardn.io)

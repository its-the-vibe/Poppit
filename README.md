# Poppit
A simple service written in Go that will pop json notifications from a redis list and execute commands listed in the payload - the original purpose is to run a CI/CD pipeline.

## Features

- Connects to Redis and pops messages from a configurable list
- Parses JSON notifications and prints them
- Configurable via environment variables
- Docker Compose setup for easy deployment
- Graceful shutdown support

## Quick Start

### Using Docker Compose

1. Start the services:
```bash
docker compose up -d
```

2. Push a test notification to Redis:
```bash
docker compose exec redis redis-cli LPUSH poppit:notifications '{"command":"echo","args":["Hello, World!"],"meta":{"job_id":"123"}}'
```

3. View the logs:
```bash
docker compose logs -f poppit
```

4. Stop the services:
```bash
docker compose down
```

### Using Go directly

1. Build the application:
```bash
go build -o poppit .
```

2. Run with custom configuration:
```bash
REDIS_ADDR=localhost:6379 REDIS_LIST_NAME=poppit:notifications ./poppit
```

## Configuration

Configuration is done via environment variables:

- `REDIS_ADDR`: Redis server address (default: `localhost:6379`)
- `REDIS_PASSWORD`: Redis password (default: empty)
- `REDIS_LIST_NAME`: Redis list name to pop from (default: `poppit:notifications`)

## Notification Format

Notifications should be JSON objects with the following structure:

```json
{
  "command": "echo",
  "args": ["Hello", "World"],
  "env": {
    "NODE_ENV": "production"
  },
  "meta": {
    "job_id": "123",
    "timestamp": "2024-12-14T20:00:00Z"
  }
}
```

Fields:
- `command` (required): The command to execute
- `args` (optional): Array of command arguments
- `env` (optional): Environment variables for the command
- `meta` (optional): Additional metadata

## Current Status

This is the first iteration that pops messages from Redis and prints them. Future versions will execute the commands listed in the payload for CI/CD automation.


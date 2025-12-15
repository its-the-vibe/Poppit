# Poppit
A service written in Go that pops JSON notifications from a Redis list and executes commands listed in the payload - the original purpose is to run a CI/CD pipeline.

## Features

- Connects to Redis and pops messages from a configurable list
- Parses JSON notifications and executes commands
- Executes commands in specified working directories
- Configurable via environment variables
- Graceful shutdown support

## Quick Start

### Prerequisites

- Go 1.23 or later
- Redis server running and accessible

### Build and Run

1. Build the application:
```bash
go build -o poppit .
```

2. Run with default configuration (assumes Redis at localhost:6379):
```bash
./poppit
```

3. Run with custom configuration:
```bash
REDIS_ADDR=localhost:6379 REDIS_LIST_NAME=poppit:notifications ./poppit
```

### Testing

To test the service, push a notification to Redis:

```bash
redis-cli LPUSH poppit:notifications '{"repo":"its-the-vibe/github-dispatcher","branch":"refs/heads/main","type":"git-webhook","dir":"/tmp","commands":["echo hello","echo world"]}'
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
  "repo": "its-the-vibe/github-dispatcher",
  "branch": "refs/heads/main",
  "type": "git-webhook",
  "dir": "/home/whatever/whereever",
  "commands": [
    "echo hello",
    "echo world"
  ]
}
```

Fields:
- `repo` (required): The repository identifier
- `branch` (required): The branch reference
- `type` (required): The notification type (e.g., "git-webhook")
- `dir` (required): The working directory where commands should be executed
- `commands` (required): Array of shell commands to execute sequentially

## How It Works

1. Poppit connects to Redis and continuously polls for messages on the configured list
2. When a message is received, it parses the JSON notification
3. It verifies the specified working directory exists
4. It executes each command sequentially in the specified directory
5. Command output (stdout/stderr) is logged
6. If a command fails, execution stops and the error is logged

## Security Considerations

- Commands are executed using `sh -c`, so be careful with untrusted input
- Ensure the Redis instance is secured and not publicly accessible
- The working directory must exist before commands are executed
- Commands run with the same permissions as the Poppit process



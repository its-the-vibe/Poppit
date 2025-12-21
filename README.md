# Poppit
A service written in Go that pops JSON notifications from a Redis list and executes commands listed in the payload - the original purpose is to run a CI/CD pipeline.

## Features

- Connects to Redis and pops messages from a configurable list
- Parses JSON notifications and executes commands
- Executes commands in specified working directories
- Publishes completion messages to Redis for Slack integration
- Configurable via environment variables
- Graceful shutdown support

## Quick Start

### Prerequisites

- Go 1.25 or later
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
- `REDIS_PUBLISH_LIST_NAME`: Redis list name to publish completion messages to (default: `slack_messages`)
- `SLACK_CHANNEL`: Slack channel for completion notifications (default: `#ci-cd`)
- `DEFAULT_TTL`: Default TTL (time-to-live) in seconds for completion messages (default: `86400`)

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
7. After all commands complete (success or failure), a completion message is published to Redis, including a TTL field
8. The completion message is formatted for [SlackLiner](https://github.com/its-the-vibe/SlackLiner) to send to Slack

## Completion Messages

After processing a notification, Poppit publishes a completion message to Redis (list: `slack_messages` by default) that can be consumed by SlackLiner for Slack notifications.

**Success Message Format:**
```json
{
  "channel": "#ci-cd",
  "text": "✅ Commands completed successfully for its-the-vibe/repo on branch refs/heads/main",
  "ttl": 86400,
  "metadata": {
    "event_type": "git-webhook",
    "event_payload": {
      "repo": "its-the-vibe/repo",
      "branch": "refs/heads/main",
      "dir": "/path/to/dir"
    }
  }
}
```

**Failure Message Format:**
```json
{
  "channel": "#ci-cd",
  "text": "❌ Commands failed for its-the-vibe/repo on branch refs/heads/main: exit status 1",
  "ttl": 86400,
  "metadata": {
    "event_type": "git-webhook",
    "event_payload": {
      "repo": "its-the-vibe/repo",
      "branch": "refs/heads/main",
      "dir": "/path/to/dir"
    }
  }
}
```

## Security Considerations

**IMPORTANT**: This service is designed for trusted CI/CD pipeline environments with controlled input sources.

- Commands are executed using `sh -c`, which allows full shell features (pipes, redirects, variable expansion) but also means any shell command can be executed
- **Only use this service with trusted notification sources** (e.g., internal Redis instances behind a firewall)
- Ensure the Redis instance is secured and not publicly accessible
- Use authentication (REDIS_PASSWORD) when connecting to Redis
- The working directory must exist before commands are executed
- Commands run with the same permissions as the Poppit process
- Consider running Poppit with limited user permissions (non-root)
- For production deployments:
  - Use firewall rules to restrict Redis access
  - Implement authentication/authorization on the notification source
  - Consider running in an isolated environment (e.g., container, VM, chroot jail)
  - Monitor and log all command executions
  - Validate notification sources before pushing to Redis



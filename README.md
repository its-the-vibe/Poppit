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
POPPIT_SERVICE_REDIS_ADDR=localhost:6379 POPPIT_SERVICE_REDIS_LIST_NAME=poppit:notifications ./poppit
```

### Testing

To test the service, push a notification to Redis:

```bash
redis-cli RPUSH poppit:notifications '{"repo":"its-the-vibe/github-dispatcher","branch":"refs/heads/main","type":"git-webhook","dir":"/tmp","commands":["echo hello","echo world"]}'
```

## Installing as a systemd Service

To run Poppit as a systemd service on Linux systems:

### 1. Build the Application

```bash
go build -o poppit .
```

### 2. Install the Binary

```bash
# Create installation directory
sudo mkdir -p /opt/poppit

# Copy the binary
sudo cp poppit /opt/poppit/

# Set ownership and permissions
sudo chown root:root /opt/poppit/poppit
sudo chmod 755 /opt/poppit/poppit
```

### 3. Create a Service User

It's recommended to run Poppit as a dedicated user for security:

```bash
# Create a system user for running Poppit
sudo useradd -r -s /bin/false -d /opt/poppit poppit
```

### 4. Install the systemd Service File

```bash
# Copy the service file
sudo cp poppit.service /etc/systemd/system/

# Edit the service file to configure your environment variables
sudo nano /etc/systemd/system/poppit.service

# Reload systemd to recognize the new service
sudo systemctl daemon-reload
```

### 5. Configure the Service

Edit `/etc/systemd/system/poppit.service` to set your environment variables:

- `POPPIT_SERVICE_REDIS_ADDR`: Your Redis server address
- `POPPIT_SERVICE_REDIS_PASSWORD`: Your Redis password (if required)
- `POPPIT_SERVICE_REDIS_LIST_NAME`: Redis list name to monitor
- Other configuration variables as needed

For Redis authentication, you can also use systemd's environment file:

```bash
# Create environment file
sudo nano /etc/poppit/poppit.env
```

Add your sensitive variables:
```
POPPIT_SERVICE_REDIS_PASSWORD=your-secret-password
```

Then update the service file to include:
```
EnvironmentFile=/etc/poppit/poppit.env
```

### 6. Start and Enable the Service

```bash
# Start the service
sudo systemctl start poppit

# Check the status
sudo systemctl status poppit

# Enable the service to start on boot
sudo systemctl enable poppit
```

### 7. View Logs

```bash
# View recent logs
sudo journalctl -u poppit -n 50

# Follow logs in real-time
sudo journalctl -u poppit -f

# View logs since boot
sudo journalctl -u poppit -b
```

### 8. Managing the Service

```bash
# Stop the service
sudo systemctl stop poppit

# Restart the service
sudo systemctl restart poppit

# Reload configuration after editing the service file
sudo systemctl daemon-reload
sudo systemctl restart poppit
```

## Configuration

Configuration is done via environment variables. All variables are prefixed with `POPPIT_SERVICE_` to avoid conflicts with environment variables used by executed commands:

- `POPPIT_SERVICE_REDIS_ADDR`: Redis server address (default: `localhost:6379`)
- `POPPIT_SERVICE_REDIS_PASSWORD`: Redis password (default: empty)
- `POPPIT_SERVICE_REDIS_LIST_NAME`: Redis list name to pop from (default: `poppit:notifications`)
- `POPPIT_SERVICE_REDIS_PUBLISH_LIST_NAME`: Redis list name to publish completion messages to (default: `slack_messages`)
- `POPPIT_SERVICE_SLACK_CHANNEL`: Slack channel for completion notifications (default: `#ci-cd`)
- `POPPIT_SERVICE_DEFAULT_TTL`: Default TTL (time-to-live) in seconds for completion messages (default: `86400`)
- `POPPIT_SERVICE_COMMAND_OUTPUT_CHANNEL`: Redis channel to publish command output to when metadata is present (default: `poppit:command-output`)

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
  ],
  "metadata": {
    "taskId": "task-12345",
    "userId": "user-456",
    "source": "github-webhook"
  }
}
```

Fields:
- `repo` (required): The repository identifier
- `branch` (required): The branch reference
- `type` (required): The notification type (e.g., "git-webhook")
- `dir` (required): The working directory where commands should be executed
- `commands` (required): Array of shell commands to execute sequentially
- `metadata` (optional): JSON object containing customizable metadata about the request (e.g., task ID, user ID, source)

## How It Works

1. Poppit connects to Redis and continuously polls for messages on the configured list using `BLPOP` (blocking left pop)
2. When a message is received, it parses the JSON notification
3. It verifies the specified working directory exists
4. It executes each command sequentially in the specified directory
5. Command output (stdout/stderr) is logged
6. If a command fails, execution stops and the error is logged
7. After all commands complete (success or failure), a completion message is published to Redis, including a TTL field
8. The completion message is formatted for [SlackLiner](https://github.com/its-the-vibe/SlackLiner) to send to Slack

### Redis List Pattern

Poppit uses a FIFO (First In, First Out) queue pattern:
- **Clients**: Use `RPUSH` to add notifications to the right end of the list
- **Service**: Uses `BLPOP` to consume notifications from the left end of the list

This ensures that notifications are processed in the order they are received, preventing race conditions and maintaining command execution order.

## Completion Messages

After processing a notification, Poppit publishes a completion message to Redis (list: `slack_messages` by default) that can be consumed by SlackLiner for Slack notifications.

**Success Message Format:**
```json
{
  "channel": "#ci-cd",
  "text": "✅ Commands completed successfully for its-the-vibe/repo on branch refs/heads/main (Duration: 12s)",
  "ttl": 86400,
  "duration": "12s",
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
  "text": "❌ Commands failed for its-the-vibe/repo on branch refs/heads/main: exit status 1 (Duration: 5s)",
  "ttl": 86400,
  "duration": "5s",
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

Fields:
- `channel`: The Slack channel where the message will be posted
- `text`: The formatted message text including operation result and duration
- `ttl`: Time-to-live in seconds for the message
- `duration`: Human-readable duration of the operation (e.g., "12s", "2m 15s", "1h 5m 30s")
- `metadata`: Additional context about the event including type and payload details

## Command Output Publishing

When a notification includes an optional `metadata` field, Poppit will publish the output of each executed command to a Redis channel (default: `poppit:command-output`). This allows callers to receive real-time feedback on command execution. The metadata is returned as-is in the command output.

**Command Output Message Format:**
```json
{
  "metadata": {
    "taskId": "task-12345",
    "userId": "user-456",
    "source": "github-webhook"
  },
  "type": "git-webhook",
  "command": "git pull",
  "output": "remote: Enumerating objects: 7, done.\nremote: Counting objects: 100% (7/7), done.\nremote: Compressing objects: 100% (1/1), done.\nremote: Total 4 (delta 3), reused 4 (delta 3), pack-reused 0 (from 0)\nUnpacking objects: 100% (4/4), 496 bytes | 49.00 KiB/s, done.\nFrom github.com:its-the-vibe/SlackCommandRelay\n   9a394c2..4068c8e  main       -> origin/main\nUpdating 9a394c2..4068c8e\nFast-forward\n docker-compose.yml | 1 +\n main.go            | 9 +++++++--\n 2 files changed, 8 insertions(+), 2 deletions(-)\n",
  "stderr": ""
}
```

Fields:
- `metadata`: The metadata object from the notification, returned as-is
- `type`: The notification type
- `command`: The executed command string
- `output`: The stdout output from the command
- `stderr`: The stderr output from the command

**How to Subscribe to Command Output:**

To receive command output messages, subscribe to the Redis channel using Redis SUBSCRIBE:

```bash
redis-cli SUBSCRIBE poppit:command-output
```

Or in your application using go-redis:
```go
pubsub := rdb.Subscribe(ctx, "poppit:command-output")
ch := pubsub.Channel()

for msg := range ch {
    var cmdOutput CommandOutput
    json.Unmarshal([]byte(msg.Payload), &cmdOutput)
    // Process the command output
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



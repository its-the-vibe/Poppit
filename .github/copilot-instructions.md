# Poppit - GitHub Copilot Instructions

## Project Overview

Poppit is a Go-based service that pops JSON notifications from a Redis list and processes them. The original purpose is to run CI/CD pipelines by executing commands listed in notification payloads.

**Current Status**: The first iteration pops messages from Redis and prints them. Future versions will execute the commands for CI/CD automation.

## Technology Stack

- **Language**: Go 1.23
- **Database**: Redis (using go-redis/v9)
- **Containerization**: Docker and Docker Compose
- **Dependencies**: Managed via Go modules (`go.mod`, `go.sum`)

## Project Structure

- `main.go` - Main application logic (Redis client, message processing, graceful shutdown)
- `Dockerfile` - Container image definition
- `docker-compose.yaml` - Multi-service orchestration (Redis + Poppit)
- `.env.example` - Example environment variables
- `vendor/` - Vendored dependencies

## Code Style and Conventions

### Go Standards
- Follow standard Go formatting using `gofmt` and `goimports`
- Use Go 1.23 features appropriately
- Follow effective Go practices from golang.org/doc/effective_go
- Use meaningful variable names (avoid single letters except for standard idioms like `i` in loops)

### Error Handling
- Always check and handle errors explicitly
- Use `log.Printf` or `log.Fatalf` for error logging
- Include context in error messages (e.g., "Failed to connect to Redis: %v")
- Use `log.Fatalf` for initialization errors, `log.Printf` for runtime errors

### Logging
- Use the standard `log` package
- Include descriptive prefixes in log messages
- Use structured logging patterns where appropriate
- Log important operations: connections, message processing, errors, shutdown

### Concurrency
- Use goroutines with proper synchronization
- Always handle context cancellation
- Use channels for signaling and shutdown coordination
- Avoid race conditions (consider using `go run -race` for testing)

### Redis Patterns
- Use `context.Context` for all Redis operations
- Handle `redis.Nil` separately from other errors (indicates no data available)
- Use blocking operations like `BLPOP` with appropriate timeouts
- Always close Redis connections on shutdown

## Building and Testing

### Local Development
```bash
# Build the application
go build -o poppit .

# Run locally (requires Redis)
REDIS_ADDR=localhost:6379 REDIS_LIST_NAME=poppit:notifications ./poppit
```

### Docker Development
```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f poppit

# Push test notification
docker compose exec redis redis-cli RPUSH poppit:notifications '{"command":"echo","args":["test"]}'

# Stop services
docker compose down
```

### Testing
- No automated tests currently exist
- When adding tests, use Go's standard testing package (`testing`)
- Test files should be named `*_test.go`
- Run tests with `go test ./...`

### Linting
- Use `gofmt` to format code
- Consider using `golangci-lint` for comprehensive linting
- Ensure code passes `go vet` checks

## Configuration

The application is configured via environment variables:

- `REDIS_ADDR` - Redis server address (default: `localhost:6379`)
- `REDIS_PASSWORD` - Redis password (default: empty)
- `REDIS_LIST_NAME` - Redis list name to pop from (default: `poppit:notifications`)

## Data Structures

### Notification JSON Format
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

- `command` (required): The command to execute
- `args` (optional): Array of command arguments
- `env` (optional): Environment variables for the command
- `meta` (optional): Additional metadata

## Documentation Requirements

- Update README.md when adding new features or changing behavior
- Document all environment variables in README.md
- Include examples for new notification formats
- Keep code comments minimal but meaningful
- Document complex algorithms or non-obvious logic

## Security Considerations

- Never commit secrets or credentials to the repository
- Use environment variables for all sensitive configuration
- Validate and sanitize command inputs before execution (important for future command execution feature)
- Be cautious with user-provided input in notifications
- Use context with timeouts to prevent hanging operations

## Development Workflow

- Keep changes focused and minimal
- Test changes with Docker Compose before committing
- Ensure graceful shutdown works correctly
- Verify Redis connection handling and error recovery
- Check that all logging is clear and helpful

## Future Considerations

- Command execution will be added in future iterations
- Security validation will be critical for command execution
- Consider adding metrics and monitoring
- May need to add queue management and retry logic
- Could benefit from structured logging (e.g., using zerolog or zap)

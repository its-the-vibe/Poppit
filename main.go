package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	RedisAddr                  string
	RedisPassword              string
	RedisDB                    int
	ListName                   string
	PublishListName            string
	SlackChannel               string
	DefaultTTL                 int
	CommandOutputChannel       string
	PublishCompletionMessage   bool
}

type Notification struct {
	Repo     string                 `json:"repo"`
	Branch   string                 `json:"branch"`
	Type     string                 `json:"type"`
	Dir      string                 `json:"dir"`
	Commands []string               `json:"commands"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type CompletionMessage struct {
	Channel  string                 `json:"channel"`
	Text     string                 `json:"text"`
	TTL      int                    `json:"ttl,omitempty"`
	Duration string                 `json:"duration,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type CommandOutput struct {
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Type     string                 `json:"type"`
	Command  string                 `json:"command"`
	Output   string                 `json:"output"`
	StdErr   string                 `json:"stderr"`
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func loadConfig() Config {
	redisAddr := os.Getenv("POPPIT_SERVICE_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	listName := os.Getenv("POPPIT_SERVICE_REDIS_LIST_NAME")
	if listName == "" {
		listName = "poppit:notifications"
	}

	publishListName := os.Getenv("POPPIT_SERVICE_REDIS_PUBLISH_LIST_NAME")
	if publishListName == "" {
		publishListName = "slack_messages"
	}

	slackChannel := os.Getenv("POPPIT_SERVICE_SLACK_CHANNEL")
	if slackChannel == "" {
		slackChannel = "#ci-cd"
	}

	commandOutputChannel := os.Getenv("POPPIT_SERVICE_COMMAND_OUTPUT_CHANNEL")
	if commandOutputChannel == "" {
		commandOutputChannel = "poppit:command-output"
	}

	defaultTTL := 24 * 60 * 60
	if ttlStr := os.Getenv("POPPIT_SERVICE_DEFAULT_TTL"); ttlStr != "" {
		if ttlVal, err := strconv.Atoi(ttlStr); err == nil && ttlVal > 0 {
			defaultTTL = ttlVal
		}
	}

	// Parse PUBLISH_COMPLETION_MESSAGE flag (default: true for backward compatibility)
	// Accepts "false" or "0" (case-insensitive) to disable, any other value enables
	publishCompletionMessageEnabled := true
	if publishStr := os.Getenv("POPPIT_SERVICE_PUBLISH_COMPLETION_MESSAGE"); publishStr != "" {
		lowerVal := strings.ToLower(publishStr)
		publishCompletionMessageEnabled = lowerVal != "false" && lowerVal != "0"
	}

	return Config{
		RedisAddr:                redisAddr,
		RedisPassword:            os.Getenv("POPPIT_SERVICE_REDIS_PASSWORD"),
		RedisDB:                  0,
		ListName:                 listName,
		PublishListName:          publishListName,
		SlackChannel:             slackChannel,
		DefaultTTL:               defaultTTL,
		CommandOutputChannel:     commandOutputChannel,
		PublishCompletionMessage: publishCompletionMessageEnabled,
	}
}

func publishCompletionMessage(ctx context.Context, rdb *redis.Client, config Config, notification Notification, success bool, errMsg string, duration time.Duration) error {
	durationStr := formatDuration(duration)
	var messageText string
	if success {
		messageText = fmt.Sprintf("✅ `%s` commands completed successfully for `%s` (Duration: %s)", notification.Type, notification.Repo, durationStr)
	} else {
		messageText = fmt.Sprintf("❌ `%s` commands failed for `%s`: %s (Duration: %s)", notification.Type, notification.Repo, errMsg, durationStr)
	}

	completionMsg := CompletionMessage{
		Channel:  config.SlackChannel,
		Text:     messageText,
		TTL:      config.DefaultTTL,
		Duration: durationStr,
		Metadata: map[string]interface{}{
			"event_type": notification.Type,
			"event_payload": map[string]interface{}{
				"repo":   notification.Repo,
				"branch": notification.Branch,
			},
		},
	}

	msgJSON, err := json.Marshal(completionMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal completion message: %w", err)
	}

	if err := rdb.LPush(ctx, config.PublishListName, msgJSON).Err(); err != nil {
		return fmt.Errorf("failed to push completion message to Redis: %w", err)
	}

	log.Printf("Published completion message to %s", config.PublishListName)
	return nil
}

func publishCommandOutput(ctx context.Context, rdb *redis.Client, config Config, notification Notification, command string, output string, stdErr string) error {
	cmdOutput := CommandOutput{
		Metadata: notification.Metadata,
		Type:     notification.Type,
		Command:  command,
		Output:   output,
		StdErr:   stdErr,
	}

	msgJSON, err := json.Marshal(cmdOutput)
	if err != nil {
		return fmt.Errorf("failed to marshal command output: %w", err)
	}

	if err := rdb.Publish(ctx, config.CommandOutputChannel, msgJSON).Err(); err != nil {
		return fmt.Errorf("failed to publish command output to Redis channel: %w", err)
	}

	log.Printf("Published command output to channel %s", config.CommandOutputChannel)
	return nil
}

func executeCommands(ctx context.Context, rdb *redis.Client, config Config, notification Notification) error {
	log.Printf("Executing commands for repo: %s, branch: %s", notification.Repo, notification.Branch)
	log.Printf("Working directory: %s", notification.Dir)

	// Verify the directory exists
	// Note: This service is designed for trusted CI/CD pipelines. For untrusted
	// environments, additional validation (e.g., path sanitization, whitelisting)
	// should be implemented to prevent path traversal and unauthorized access.
	if _, err := os.Stat(notification.Dir); os.IsNotExist(err) {
		log.Printf("Directory does not exist: %s", notification.Dir)
		return err
	}

	// Execute each command sequentially
	// Note: Commands are executed via shell (sh -c) to support shell features like
	// pipes, redirects, and environment variable expansion. This is intentional for
	// CI/CD flexibility but requires trusted input sources. For untrusted environments,
	// consider command whitelisting or argument parsing instead of shell execution.
	for i, cmdStr := range notification.Commands {
		log.Printf("Executing command %d/%d: %s", i+1, len(notification.Commands), cmdStr)

		cmd := exec.Command("sh", "-c", cmdStr)
		cmd.Dir = notification.Dir

		// If metadata is present, capture output to publish
		if len(notification.Metadata) > 0 {
			// Note: We capture stdout and stderr separately to allow downstream
			// consumers to process them independently. For commands with very large
			// outputs, consider implementing streaming or output limits.
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			stdoutStr := stdout.String()
			stderrStr := stderr.String()

			// Log the output to stdout for observability
			if stdoutStr != "" {
				log.Printf("Command stdout:\n%s", stdoutStr)
			}
			if stderrStr != "" {
				log.Printf("Command stderr:\n%s", stderrStr)
			}

			// Publish command output to Redis channel
			if pubErr := publishCommandOutput(ctx, rdb, config, notification, cmdStr, stdoutStr, stderrStr); pubErr != nil {
				log.Printf("Failed to publish command output: %v", pubErr)
			}

			if err != nil {
				log.Printf("Command failed: %s (error: %v)", cmdStr, err)
				return err
			}
		} else {
			// Original behavior - stream output to stdout/stderr
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				log.Printf("Command failed: %s (error: %v)", cmdStr, err)
				return err
			}
		}
		log.Printf("Command %d completed successfully", i+1)
	}

	log.Println("All commands executed successfully")
	return nil
}

func main() {
	log.Println("Starting Poppit service...")

	config := loadConfig()
	log.Printf("Config: Redis=%s, List=%s", config.RedisAddr, config.ListName)

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	ctx := context.Background()

	// Test Redis connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start popping messages
	go func() {
		for {
			// BLPOP blocks until a message is available or timeout occurs
			result, err := rdb.BLPop(ctx, 5*time.Second, config.ListName).Result()
			if err == redis.Nil {
				// Timeout - no message available
				continue
			} else if err != nil {
				log.Printf("Error popping message: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// result[0] is the list name, result[1] is the message
			if len(result) < 2 {
				log.Println("Received empty result from Redis")
				continue
			}

			message := result[1]
			log.Println("=== New Notification ===")
			log.Printf("Raw message: %s", message)

			// Start timing the notification processing
			startTime := time.Now()

			// Try to parse as JSON
			var notification Notification
			if err := json.Unmarshal([]byte(message), &notification); err != nil {
				log.Printf("Failed to parse JSON: %v", err)
				log.Println("========================")
				continue
			}

			// Pretty print the notification
			prettyJSON, err := json.MarshalIndent(notification, "", "  ")
			if err == nil {
				log.Printf("Parsed notification:\n%s", string(prettyJSON))
			}

			// Execute the commands
			if err := executeCommands(ctx, rdb, config, notification); err != nil {
				duration := time.Since(startTime)
				log.Printf("Failed to execute commands: %v (Duration: %s)", err, formatDuration(duration))
				// Publish failure message if enabled
				if config.PublishCompletionMessage {
					if pubErr := publishCompletionMessage(ctx, rdb, config, notification, false, err.Error(), duration); pubErr != nil {
						log.Printf("Failed to publish completion message: %v", pubErr)
					}
				}
			} else {
				duration := time.Since(startTime)
				log.Printf("All commands completed successfully (Duration: %s)", formatDuration(duration))
				// Publish success message if enabled
				if config.PublishCompletionMessage {
					if pubErr := publishCompletionMessage(ctx, rdb, config, notification, true, "", duration); pubErr != nil {
						log.Printf("Failed to publish completion message: %v", pubErr)
					}
				}
			}

			log.Println("========================")
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("\nShutting down gracefully...")

	// Close Redis connection
	if err := rdb.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}

	log.Println("Shutdown complete")
}

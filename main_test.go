package main

import (
	"os"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{"seconds only", 45 * time.Second, "45s"},
		{"minutes and seconds", 2*time.Minute + 15*time.Second, "2m 15s"},
		{"hours minutes seconds", time.Hour + 5*time.Minute + 30*time.Second, "1h 5m 30s"},
		{"zero duration", 0, "0s"},
		{"sub-second rounds to zero", 400 * time.Millisecond, "0s"},
		{"sub-second rounds up", 600 * time.Millisecond, "1s"},
		{"exactly one minute", 60 * time.Second, "1m 0s"},
		{"exactly one hour", 3600 * time.Second, "1h 0m 0s"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatDuration(tc.input)
			if result != tc.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Unset any env vars that might be set in the environment
	envVars := []string{
		"POPPIT_SERVICE_REDIS_ADDR",
		"POPPIT_SERVICE_REDIS_PASSWORD",
		"POPPIT_SERVICE_REDIS_LIST_NAME",
		"POPPIT_SERVICE_REDIS_PUBLISH_LIST_NAME",
		"POPPIT_SERVICE_SLACK_CHANNEL",
		"POPPIT_SERVICE_DEFAULT_TTL",
		"POPPIT_SERVICE_COMMAND_OUTPUT_CHANNEL",
		"POPPIT_SERVICE_PUBLISH_COMPLETION_MESSAGE",
		"POPPIT_SERVICE_EXECUTION_EVENTS_CHANNEL",
		"POPPIT_SERVICE_CURRENT_COMMAND_KEY",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}

	cfg := loadConfig()

	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("RedisAddr = %q, want %q", cfg.RedisAddr, "localhost:6379")
	}
	if cfg.ListName != "poppit:notifications" {
		t.Errorf("ListName = %q, want %q", cfg.ListName, "poppit:notifications")
	}
	if cfg.PublishListName != "slack_messages" {
		t.Errorf("PublishListName = %q, want %q", cfg.PublishListName, "slack_messages")
	}
	if cfg.SlackChannel != "#ci-cd" {
		t.Errorf("SlackChannel = %q, want %q", cfg.SlackChannel, "#ci-cd")
	}
	if cfg.CommandOutputChannel != "poppit:command-output" {
		t.Errorf("CommandOutputChannel = %q, want %q", cfg.CommandOutputChannel, "poppit:command-output")
	}
	if cfg.DefaultTTL != 86400 {
		t.Errorf("DefaultTTL = %d, want %d", cfg.DefaultTTL, 86400)
	}
	if !cfg.PublishCompletionMessage {
		t.Errorf("PublishCompletionMessage = false, want true")
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	setEnv := func(key, value string) {
		t.Helper()
		os.Setenv(key, value)
		t.Cleanup(func() { os.Unsetenv(key) })
	}
	setEnv("POPPIT_SERVICE_REDIS_ADDR", "redis:6380")
	setEnv("POPPIT_SERVICE_REDIS_PASSWORD", "secret")
	setEnv("POPPIT_SERVICE_REDIS_LIST_NAME", "custom:list")
	setEnv("POPPIT_SERVICE_REDIS_PUBLISH_LIST_NAME", "custom:publish")
	setEnv("POPPIT_SERVICE_SLACK_CHANNEL", "#custom-channel")
	setEnv("POPPIT_SERVICE_DEFAULT_TTL", "3600")
	setEnv("POPPIT_SERVICE_COMMAND_OUTPUT_CHANNEL", "custom:output")
	setEnv("POPPIT_SERVICE_EXECUTION_EVENTS_CHANNEL", "custom:events")
	setEnv("POPPIT_SERVICE_CURRENT_COMMAND_KEY", "custom:command")

	cfg := loadConfig()

	if cfg.RedisAddr != "redis:6380" {
		t.Errorf("RedisAddr = %q, want %q", cfg.RedisAddr, "redis:6380")
	}
	if cfg.RedisPassword != "secret" {
		t.Errorf("RedisPassword = %q, want %q", cfg.RedisPassword, "secret")
	}
	if cfg.ListName != "custom:list" {
		t.Errorf("ListName = %q, want %q", cfg.ListName, "custom:list")
	}
	if cfg.PublishListName != "custom:publish" {
		t.Errorf("PublishListName = %q, want %q", cfg.PublishListName, "custom:publish")
	}
	if cfg.SlackChannel != "#custom-channel" {
		t.Errorf("SlackChannel = %q, want %q", cfg.SlackChannel, "#custom-channel")
	}
	if cfg.DefaultTTL != 3600 {
		t.Errorf("DefaultTTL = %d, want %d", cfg.DefaultTTL, 3600)
	}
	if cfg.CommandOutputChannel != "custom:output" {
		t.Errorf("CommandOutputChannel = %q, want %q", cfg.CommandOutputChannel, "custom:output")
	}
	if cfg.ExecutionEventsChannel != "custom:events" {
		t.Errorf("ExecutionEventsChannel = %q, want %q", cfg.ExecutionEventsChannel, "custom:events")
	}
	if cfg.CurrentCommandKey != "custom:command" {
		t.Errorf("CurrentCommandKey = %q, want %q", cfg.CurrentCommandKey, "custom:command")
	}
}

func TestLoadConfig_PublishCompletionMessageFlag(t *testing.T) {
	tests := []struct {
		envValue string
		expected bool
	}{
		{"", true},
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"0", false},
	}

	for _, tc := range tests {
		t.Run("value="+tc.envValue, func(t *testing.T) {
			if tc.envValue == "" {
				os.Unsetenv("POPPIT_SERVICE_PUBLISH_COMPLETION_MESSAGE")
			} else {
				os.Setenv("POPPIT_SERVICE_PUBLISH_COMPLETION_MESSAGE", tc.envValue)
				defer os.Unsetenv("POPPIT_SERVICE_PUBLISH_COMPLETION_MESSAGE")
			}
			cfg := loadConfig()
			if cfg.PublishCompletionMessage != tc.expected {
				t.Errorf("PublishCompletionMessage with env %q = %v, want %v", tc.envValue, cfg.PublishCompletionMessage, tc.expected)
			}
		})
	}
}

func TestLoadConfig_InvalidTTLIgnored(t *testing.T) {
	os.Setenv("POPPIT_SERVICE_DEFAULT_TTL", "not-a-number")
	defer os.Unsetenv("POPPIT_SERVICE_DEFAULT_TTL")

	cfg := loadConfig()
	if cfg.DefaultTTL != 86400 {
		t.Errorf("DefaultTTL with invalid env = %d, want default 86400", cfg.DefaultTTL)
	}
}

func TestLoadConfig_ZeroTTLIgnored(t *testing.T) {
	os.Setenv("POPPIT_SERVICE_DEFAULT_TTL", "0")
	defer os.Unsetenv("POPPIT_SERVICE_DEFAULT_TTL")

	cfg := loadConfig()
	if cfg.DefaultTTL != 86400 {
		t.Errorf("DefaultTTL with zero env = %d, want default 86400", cfg.DefaultTTL)
	}
}

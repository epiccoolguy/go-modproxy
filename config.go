package modproxy

import "os"

// Config holds the configuration for ModProxy.
type Config struct {
	HostPattern     string
	HostReplacement string
	PathPattern     string
	PathReplacement string
}

// Constants for default pattern and replacement values.
const (
	DefaultHostPattern     = "go.loafoe.dev"   // Default pattern for host matching.
	DefaultHostReplacement = "github.com"      // Default host replacement.
	DefaultPathPattern     = "/"               // Default pattern for path matching.
	DefaultPathReplacement = "/loafoe-dev/go-" // Default path replacement.
)

// getEnvOrDefault retrieves an environment variable by key.
// Returns defaultValue if the environment variable is not set.
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// NewConfig creates a new instance of Config with provided values.
func NewConfig(hostPattern, hostReplacement, pathPattern, pathReplacement string) *Config {
	return &Config{
		HostPattern:     hostPattern,
		HostReplacement: hostReplacement,
		PathPattern:     pathPattern,
		PathReplacement: pathReplacement,
	}
}

// NewConfigFromEnvironment creates a new instance of Config with values from environment variables or default values.
func NewConfigFromEnvironment() *Config {
	return NewConfig(
		getEnvOrDefault("HOST_PATTERN", DefaultHostPattern),
		getEnvOrDefault("HOST_REPLACEMENT", DefaultHostReplacement),
		getEnvOrDefault("PATH_PATTERN", DefaultPathPattern),
		getEnvOrDefault("PATH_REPLACEMENT", DefaultPathReplacement),
	)
}

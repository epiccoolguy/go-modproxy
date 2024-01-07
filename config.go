package modproxy

import "os"

// Config holds the configuration for ModProxy.
type Config struct {
	SchemePattern     string
	SchemeReplacement string
	HostPattern       string
	HostReplacement   string
	PathPattern       string
	PathReplacement   string
}

// Constants for default pattern and replacement values.
const (
	DefaultSchemePattern     = "http"
	DefaultSchemeReplacement = "https"
	DefaultHostPattern       = "go.loafoe.dev"
	DefaultHostReplacement   = "github.com"
	DefaultPathPattern       = "/"
	DefaultPathReplacement   = "/loafoe-dev/go-"
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

// NewConfigFromEnvironment creates a new instance of Config with values from environment variables or default values.
func NewConfigFromEnvironment() *Config {
	return &Config{
		SchemePattern:     getEnvOrDefault("SCHEME_PATTERN", DefaultSchemePattern),
		SchemeReplacement: getEnvOrDefault("SCHEME_REPLACEMENT", DefaultSchemeReplacement),
		HostPattern:       getEnvOrDefault("HOST_PATTERN", DefaultHostPattern),
		HostReplacement:   getEnvOrDefault("HOST_REPLACEMENT", DefaultHostReplacement),
		PathPattern:       getEnvOrDefault("PATH_PATTERN", DefaultPathPattern),
		PathReplacement:   getEnvOrDefault("PATH_REPLACEMENT", DefaultPathReplacement),
	}
}

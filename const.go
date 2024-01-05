package modproxy

import "os"

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

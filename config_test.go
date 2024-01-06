package modproxy

import (
	"os"
	"reflect"
	"testing"
)

// Test case struct
type ConfigFromEnvTestCase struct {
	name           string
	envVars        map[string]string // Environment variables to set
	expectedConfig Config            // Expected Config
}

// Test cases
var configFromEnvTestCases = []ConfigFromEnvTestCase{
	{
		name:    "Default environment variables",
		envVars: map[string]string{},
		expectedConfig: Config{
			HostPattern:     DefaultHostPattern,
			HostReplacement: DefaultHostReplacement,
			PathPattern:     DefaultPathPattern,
			PathReplacement: DefaultPathReplacement,
		},
	},
	{
		name: "Custom environment variables",
		envVars: map[string]string{
			"HOST_PATTERN":     "custom.host.pattern",
			"HOST_REPLACEMENT": "custom.host.replacement",
			"PATH_PATTERN":     "custom/path/pattern",
			"PATH_REPLACEMENT": "custom/path/replacement",
		},
		expectedConfig: Config{
			HostPattern:     "custom.host.pattern",
			HostReplacement: "custom.host.replacement",
			PathPattern:     "custom/path/pattern",
			PathReplacement: "custom/path/replacement",
		},
	},
}

func TestNewConfigFromEnvironment(t *testing.T) {
	for _, tc := range configFromEnvTestCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save current environment variables and defer their restoration
			originalEnvVars := make(map[string]string)
			for key := range tc.envVars {
				originalEnvVars[key] = os.Getenv(key)
				defer os.Setenv(key, originalEnvVars[key])
			}

			// Set environment variables as per test case
			for key, value := range tc.envVars {
				os.Setenv(key, value)
			}
			cfg := NewConfigFromEnvironment()

			// Assert the result is as expected
			if !reflect.DeepEqual(cfg, &tc.expectedConfig) {
				t.Errorf("NewConfigFromEnvironment() = %+v, want %+v", cfg, &tc.expectedConfig)
			}
		})
	}
}

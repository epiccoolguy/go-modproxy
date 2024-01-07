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
			SchemePattern:     DefaultSchemePattern,
			SchemeReplacement: DefaultSchemeReplacement,
			HostPattern:       DefaultHostPattern,
			HostReplacement:   DefaultHostReplacement,
			PathPattern:       DefaultPathPattern,
			PathReplacement:   DefaultPathReplacement,
		},
	},
	{
		name: "Custom environment variables",
		envVars: map[string]string{
			"SCHEME_PATTERN":     "pattern",
			"SCHEME_REPLACEMENT": "replacement",
			"HOST_PATTERN":       "host.pattern",
			"HOST_REPLACEMENT":   "host.replacement",
			"PATH_PATTERN":       "path/pattern",
			"PATH_REPLACEMENT":   "path/replacement",
		},
		expectedConfig: Config{
			SchemePattern:     "pattern",
			SchemeReplacement: "replacement",
			HostPattern:       "host.pattern",
			HostReplacement:   "host.replacement",
			PathPattern:       "path/pattern",
			PathReplacement:   "path/replacement",
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

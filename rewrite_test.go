package modproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test case structs
type GetRequestURLTestCase struct {
	name        string
	request     *http.Request
	expectedURL string
}

type GetPackagePathTestCase struct {
	name         string
	url          string
	expectedPath string
	expectError  bool
}

type RewriteURLTestCase struct {
	name                 string
	originalURL          string
	cfg                  *Config
	expectedRewrittenURL string
	expectError          bool
}

// Test cases
var getRequestURLTestCases = []GetRequestURLTestCase{
	{
		name:        "Generic HTTP URL",
		request:     httptest.NewRequest(http.MethodGet, "http://example.com/path", nil),
		expectedURL: "http://example.com/path",
	},
	{
		name:        "Generic HTTPS URL",
		request:     httptest.NewRequest(http.MethodGet, "https://example.com/path", nil),
		expectedURL: "https://example.com/path",
	},
	{
		name: "No Host header",
		request: func() *http.Request {
			req := httptest.NewRequest(http.MethodGet, "/path", nil)
			req.Host = "" // Manually unset the Host header that httptest.NewRequest sets automatically
			return req
		}(),
		expectedURL: "http://localhost/path",
	},
}

var getPackagePathTestCases = []GetPackagePathTestCase{
	{
		name:         "Generic HTTP URL",
		url:          "http://example.com/path",
		expectedPath: "example.com/path",
	},
	{
		name:        "Malformed URL",
		url:         "http://a b.com/", // Malformed URL
		expectError: true,
	},
}

var rewriteURLTestCases = []RewriteURLTestCase{
	{
		name:        "Default host and path rewrite",
		originalURL: "http://go.loafoe.dev/package",
		cfg: &Config{
			SchemePattern:     "http",
			SchemeReplacement: "https",
			HostPattern:       "go.loafoe.dev",
			HostReplacement:   "github.com",
			PathPattern:       "/",
			PathReplacement:   "/",
		},
		expectedRewrittenURL: "https://github.com/package",
	},
	{
		name:        "Remove ?go-get=1",
		originalURL: "http://go.loafoe.dev/modproxy?go-get=1",
		cfg: &Config{
			SchemePattern:     "http",
			SchemeReplacement: "https",
			HostPattern:       "go.loafoe.dev",
			HostReplacement:   "github.com",
			PathPattern:       "/",
			PathReplacement:   "/loafoe-dev/go-",
		},
		expectedRewrittenURL: "https://github.com/loafoe-dev/go-modproxy",
	},
	{
		name:        "Malformed URL",
		originalURL: "http://%42:8080/", // Malformed URL
		cfg: &Config{
			SchemePattern:     "http",
			SchemeReplacement: "https",
			HostPattern:       "go.loafoe.dev",
			HostReplacement:   "github.com",
			PathPattern:       "/",
			PathReplacement:   "/",
		},
		expectError: true,
	},
}

func TestGetRequestURL(t *testing.T) {
	for _, tc := range getRequestURLTestCases {
		t.Run(tc.name, func(t *testing.T) {
			gotURL := GetRequestURL(tc.request)
			if gotURL != tc.expectedURL {
				t.Errorf("GetRequestURL() = %v, want %v", gotURL, tc.expectedURL)
			}
		})
	}
}

func TestGetPackagePath(t *testing.T) {
	for _, tc := range getPackagePathTestCases {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, err := GetPackagePath(tc.url)

			// Check for error consistency
			if (err != nil) != tc.expectError {
				t.Errorf("GetPackagePath() error = %v, expectError %v", err, tc.expectError)
				return
			}

			// Compare the resulting path with the expected path, if no error is expected
			if !tc.expectError && gotPath != tc.expectedPath {
				t.Errorf("GetPackagePath() got %v, want %v", gotPath, tc.expectedPath)
			}
		})
	}
}

func TestRewriteURL(t *testing.T) {
	for _, tc := range rewriteURLTestCases {
		t.Run(tc.name, func(t *testing.T) {
			rewrittenURL, err := RewriteURL(tc.originalURL, tc.cfg)

			if (err != nil) != tc.expectError {
				t.Errorf("RewriteURL() error = %v, expectError %v", err, tc.expectError)
				return
			}

			if !tc.expectError && rewrittenURL != tc.expectedRewrittenURL {
				t.Errorf("RewriteURL() got %v, want %v", rewrittenURL, tc.expectedRewrittenURL)
			}
		})
	}
}

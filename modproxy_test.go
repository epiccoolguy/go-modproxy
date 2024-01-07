package modproxy

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// Test case structs
type ModProxyTestCase struct {
	name            string
	config          *Config
	mockURLGetter   RequestURLGetter  // Optional mock for RequestURLGetter
	mockPathGetter  PackagePathGetter // Optional mock for PackagePathGetter
	mockURLRewriter URLRewriter       // Optional mock for URLRewriter
	module          string
	expectedCode    int
	expectedRewrite string
}

// Mock implementations
type mockPackagePathGetter struct {
	mockFunc func(url string) (string, error)
}

type mockURLRewriter struct {
	mockFunc func(originalURL string, cfg *Config) (string, error)
}

type mockRequestURLGetter struct {
	mockFunc func(r *http.Request) string
}

func (m mockPackagePathGetter) GetPackagePath(url string) (string, error) {
	return m.mockFunc(url)
}

func (m mockURLRewriter) RewriteURL(originalURL string, cfg *Config) (string, error) {
	return m.mockFunc(originalURL, cfg)
}

func (m mockRequestURLGetter) GetRequestURL(r *http.Request) string {
	return m.mockFunc(r)
}

// Compile-time check to ensure mocks implement interfaces
var _ RequestURLGetter = &mockRequestURLGetter{}
var _ PackagePathGetter = &mockPackagePathGetter{}
var _ URLRewriter = &mockURLRewriter{}

// Test cases
var modProxyTestCases = []ModProxyTestCase{
	{
		name: "Test valid module",
		config: &Config{
			HostPattern:     "go.loafoe.dev",
			HostReplacement: "github.com",
			PathPattern:     "/",
			PathReplacement: "/loafoe-dev/go-",
		},
		module:          "modproxy",
		expectedCode:    http.StatusOK,
		expectedRewrite: "https://github.com/loafoe-dev/go-modproxy",
	},
	{
		name: "Mock error with GetPackagePath",
		config: &Config{
			HostPattern:     "go.loafoe.dev",
			HostReplacement: "github.com",
			PathPattern:     "/",
			PathReplacement: "/loafoe-dev/go-",
		},
		mockPathGetter: &mockPackagePathGetter{
			mockFunc: func(url string) (string, error) {
				return "", errors.New("mock error")
			},
		},
		module:       "modproxy",
		expectedCode: http.StatusInternalServerError, // Expect an internal server error
	},
	{
		name: "Mock error with RewriteURL",
		config: &Config{
			HostPattern:     "go.loafoe.dev",
			HostReplacement: "github.com",
			PathPattern:     "/",
			PathReplacement: "/loafoe-dev/go-",
		},
		mockURLRewriter: &mockURLRewriter{
			mockFunc: func(originalURL string, cfg *Config) (string, error) {
				return "", errors.New("mock error")
			},
		},
		module:       "modproxy",
		expectedCode: http.StatusInternalServerError,
	},
}

// extractMetaTagAttribute is a helper function to extract the content of an attribute of a specified meta tag.
func extractMetaTagAttribute(htmlContent, metaName, attrName string) (string, bool) {
	// Parse the HTML content
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", false
	}

	// Walk the node tree to find the meta tag
	var walker func(*html.Node) (string, bool)
	walker = func(n *html.Node) (string, bool) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			// Walk over the attributes of the meta tag
			for _, a := range n.Attr {
				// Determine whether this is the desired meta tag by name
				if a.Key == "name" && a.Val == metaName {
					// Walk over the attributes again to find the desired attribute
					for _, a := range n.Attr {
						if a.Key == attrName {
							return a.Val, true
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			content, found := walker(c)
			if found {
				return content, true
			}
		}
		return "", false
	}

	return walker(doc)
}

func TestModProxy(t *testing.T) {
	for _, tc := range modProxyTestCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use provided mocks if they're not nil, else use default behavior
			urlGetter := tc.mockURLGetter
			if urlGetter == nil {
				urlGetter = DefaultRequestURLGetter{}
			}
			pathGetter := tc.mockPathGetter
			if pathGetter == nil {
				pathGetter = DefaultPackagePathGetter{}
			}
			urlRewriter := tc.mockURLRewriter
			if urlRewriter == nil {
				urlRewriter = DefaultURLRewriter{}
			}

			// Create an HTTP handler using the configuration
			handler := NewModProxyHandler(tc.config, urlGetter, pathGetter, urlRewriter)

			// Create an HTTP request to the handler
			url := fmt.Sprintf("https://%s%s%s", tc.config.HostPattern, tc.config.PathPattern, tc.module)
			req := httptest.NewRequest(http.MethodGet, url, nil)

			// Record the HTTP response
			w := httptest.NewRecorder()

			// Serve the HTTP request to our handler
			handler.ServeHTTP(w, req)

			// Check the response status code
			if got, want := w.Code, tc.expectedCode; got != want {
				t.Errorf("ModProxy(%q):\n\tgot code %v\n\twant code %v", url, got, want)
			}

			// If an error is expected, don't proceed to check the content
			if tc.expectedCode >= http.StatusBadRequest {
				return
			}

			// Read the HTML response body
			respBody := w.Body.String()

			// Extract the go-import meta tag content
			gotMetaContent, found := extractMetaTagAttribute(respBody, "go-import", "content")
			if !found {
				t.Fatalf("go-import meta tag not found")
			}

			// Expected meta tag content
			packagePath, err := GetPackagePath(url)
			if err != nil {
				t.Fatalf("Error getting package path: %v", err)
			}
			wantMetaContent := fmt.Sprintf("%s git %s", packagePath, tc.expectedRewrite)

			// Check the go-import meta tag content
			if !strings.Contains(gotMetaContent, wantMetaContent) {
				t.Fatalf("ModProxy(%q):\n\tgot meta content %v\n\twant meta content %v", url, gotMetaContent, wantMetaContent)
			}
		})
	}
}

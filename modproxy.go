package modproxy

import (
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

// init registers the ModProxy function as an HTTP-triggered function using environment variables.
func init() {
	// Initialize configuration.
	cfg := NewConfigFromEnvironment()

	// Register the ModProxy handler with the configuration.
	functions.HTTP("ModProxy", NewModProxyHandler(cfg))
}

// generateMetaTags generates the HTML response with the go-import meta tag.
func generateMetaTags(packagePath, rewrittenURL string) string {
	return fmt.Sprintf(`<html><head><meta name="go-import" content="%s git %s"></head><body></body></html>`, packagePath, rewrittenURL)
}

// ModProxy is the main handler for the HTTP function.
// It rewrites the requested URL based on the provided configuration.
func ModProxy(cfg *Config, w http.ResponseWriter, r *http.Request) {
	// Get the complete original request URL.
	originalURL := GetRequestURL(r)

	// Get the package path (host + path) from the request URL
	packagePath, err := GetPackagePath(originalURL)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Rewrite the URL based on the patterns and replacements.
	rewrittenURL, err := RewriteURL(originalURL, cfg.HostPattern, cfg.HostReplacement, cfg.PathPattern, cfg.PathReplacement)
	if err != nil {
		// Handle error, e.g., by sending an HTTP error response
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Generate the HTML response with meta tags
	htmlResponse := generateMetaTags(packagePath, rewrittenURL)

	// Set the Content-Type header and write the HTML response
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, htmlResponse)
}

// NewModProxyHandler creates a new HTTP handler for ModProxy with the provided configuration.
func NewModProxyHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ModProxy(cfg, w, r)
	}
}

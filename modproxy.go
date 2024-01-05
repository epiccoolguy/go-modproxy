package modproxy

import (
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

// init registers the ModProxy function as an HTTP-triggered function.
func init() {
	functions.HTTP("ModProxy", ModProxy)
}

// ModProxy is the main handler for the HTTP function.
// It rewrites the requested URL based on environment variable configurations.
func ModProxy(w http.ResponseWriter, r *http.Request) {
	// Retrieve environment variables or use default values.
	hostPattern := getEnvOrDefault("hostPattern", DefaultHostPattern)
	hostReplacement := getEnvOrDefault("hostReplacement", DefaultHostReplacement)
	pathPattern := getEnvOrDefault("pathPattern", DefaultPathPattern)
	pathReplacement := getEnvOrDefault("pathReplacement", DefaultPathReplacement)

	// Get the complete original request URL.
	originalURL := GetRequestURL(r)
	// Rewrite the URL based on the patterns and replacements.
	rewrittenURL := RewriteURL(originalURL, hostPattern, hostReplacement, pathPattern, pathReplacement)

	// Send the rewritten URL as the response.
	fmt.Fprintln(w, rewrittenURL)
}

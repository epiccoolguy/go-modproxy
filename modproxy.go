package modproxy

import (
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

// Interfaces
type RequestURLGetter interface {
	GetRequestURL(r *http.Request) string
}

type PackagePathGetter interface {
	GetPackagePath(url string) (string, error)
}

type URLRewriter interface {
	RewriteURL(originalURL string, cfg *Config) (string, error)
}

// Concrete implementations
type DefaultRequestURLGetter struct{}
type DefaultPackagePathGetter struct{}
type DefaultURLRewriter struct{}

func (DefaultRequestURLGetter) GetRequestURL(r *http.Request) string {
	return GetRequestURL(r)
}

func (DefaultPackagePathGetter) GetPackagePath(url string) (string, error) {
	return GetPackagePath(url)
}

func (DefaultURLRewriter) RewriteURL(originalURL string, cfg *Config) (string, error) {
	return RewriteURL(originalURL, cfg)
}

// Compile-time check to ensure default implementations correctly implement interfaces
var _ RequestURLGetter = &DefaultRequestURLGetter{}
var _ PackagePathGetter = &DefaultPackagePathGetter{}
var _ URLRewriter = &DefaultURLRewriter{}

// URLManipulator contains the dependencies for the ModProxy function.
type URLManipulator struct {
	Config      *Config
	URLGetter   RequestURLGetter
	PathGetter  PackagePathGetter
	URLRewriter URLRewriter
}

// init registers the ModProxy function as an HTTP-triggered function using environment variables.
func init() {
	// Initialize configuration.
	cfg := NewConfigFromEnvironment()

	// Create default implementations for the interfaces
	urlGetter := DefaultRequestURLGetter{}
	pathGetter := DefaultPackagePathGetter{}
	urlRewriter := DefaultURLRewriter{}

	// Register the ModProxy handler with the configuration.
	functions.HTTP("ModProxy", NewModProxyHandler(cfg, urlGetter, pathGetter, urlRewriter))
}

// generateMetaTags generates the HTML response with the go-import meta tag.
func generateMetaTags(packagePath, rewrittenURL string) string {
	return fmt.Sprintf(`<html><head><meta name="go-import" content="%s git %s"></head><body></body></html>`, packagePath, rewrittenURL)
}

// ModProxy is the main handler for the HTTP function.
// It rewrites the requested URL based on the provided configuration.
func ModProxy(cfg *Config, urlGetter RequestURLGetter, pathGetter PackagePathGetter, urlRewriter URLRewriter, w http.ResponseWriter, r *http.Request) {
	// Get the complete original request URL.
	originalURL := urlGetter.GetRequestURL(r)

	// Get the package path (host + path) from the request URL
	packagePath, err := pathGetter.GetPackagePath(originalURL)
	if err != nil {
		// Handle error, e.g., by sending an HTTP error response
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Rewrite the URL based on the patterns and replacements.
	rewrittenURL, err := urlRewriter.RewriteURL(originalURL, cfg)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Generate the HTML response with meta tags
	htmlResponse := generateMetaTags(packagePath, rewrittenURL)

	// Set the Content-Type header and write the HTML response
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, htmlResponse)
}

// NewModProxyHandler creates a new HTTP handler for ModProxy with the provided configuration and dependencies.
func NewModProxyHandler(cfg *Config, urlGetter RequestURLGetter, pathGetter PackagePathGetter, urlRewriter URLRewriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ModProxy logic using injected dependencies
		ModProxy(cfg, urlGetter, pathGetter, urlRewriter, w, r)
	}
}

package modproxy

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// GetRequestURL constructs the full request URL from an http.Request object.
func GetRequestURL(r *http.Request) string {
	scheme := "http" // Default scheme is HTTP.
	if r.TLS != nil {
		scheme = "https" // Use HTTPS if the request is TLS-secured.
	}

	host := r.Host // Host is obtained from the request's "Host" header.
	if host == "" {
		host = "localhost" // Fallback to 'localhost' if the Host header is not set.
	}

	// Combine scheme, host, and request URI to form the full URL.
	return fmt.Sprintf("%s://%s%s", scheme, host, r.URL.RequestURI())
}

func removeVersionSuffix(path string) string {
	re := regexp.MustCompile(`/v(\d+)$`)
	return re.ReplaceAllString(path, "")
}

// GetPackagePath extracts the host and path from the request URL,
// omitting the scheme. This is used for the go-import meta tag.
func GetPackagePath(r string) (string, error) {
	// Parse the request URL
	parsedURL, err := url.Parse(r)
	if err != nil {
		return "", err
	}

	// Remove any /vX suffix from the path
	parsedURL.Path = removeVersionSuffix(parsedURL.Path)

	// Concatenate the host and path
	packagePath := fmt.Sprintf("%s%s", parsedURL.Host, parsedURL.Path)
	return packagePath, nil
}

// RewriteURL rewrites a given URL based on the provided patterns and replacements configuration.
func RewriteURL(originalURL string, cfg *Config) (string, error) {
	copy, err := url.Parse(originalURL)
	if err != nil {
		return "", err
	}

	// Replace parts of the URL according to the specified patterns and replacements.
	copy.Scheme = strings.Replace(copy.Scheme, cfg.SchemePattern, cfg.SchemeReplacement, 1)
	copy.Host = strings.Replace(copy.Host, cfg.HostPattern, cfg.HostReplacement, 1)
	copy.Path = strings.Replace(copy.Path, cfg.PathPattern, cfg.PathReplacement, 1)

	// Remove any /vX suffix from the path
	copy.Path = removeVersionSuffix(copy.Path)

	// Remove ?go-get=1 from the go get request
	query := copy.Query()
	query.Del("go-get")
	copy.RawQuery = query.Encode()

	return copy.String(), nil // Return the modified URL as a string.
}

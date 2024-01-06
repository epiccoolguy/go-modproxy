package modproxy

import (
	"fmt"
	"net/http"
	"net/url"
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

// GetPackagePath extracts the host and path from the request URL,
// omitting the scheme. This is used for the go-import meta tag.
func GetPackagePath(r string) (string, error) {
	// Parse the request URL
	parsedURL, err := url.Parse(r)
	if err != nil {
		return "", err
	}

	// Concatenate the host and path
	packagePath := fmt.Sprintf("%s%s", parsedURL.Host, parsedURL.Path)
	return packagePath, nil
}

// RewriteURL rewrites a given URL based on the provided patterns and replacements.
func RewriteURL(originalURL, hostPattern, hostReplacement, pathPattern, pathReplacement string) (string, error) {
	copy, err := url.Parse(originalURL)
	if err != nil {
		return "", err
	}

	// Replace the host and path according to the specified patterns and replacements.
	copy.Host = strings.Replace(copy.Host, hostPattern, hostReplacement, 1)
	copy.Path = strings.Replace(copy.Path, pathPattern, pathReplacement, 1)

	return copy.String(), nil // Return the modified URL as a string.
}

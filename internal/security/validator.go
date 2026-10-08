package security

import (
	"errors"
	"fmt"
	"html"
	"net"
	"net/url"
	"strings"
)

var (
	ErrInvalidURLScheme   = errors.New("invalid url scheme: only http and https are allowed")
	ErrInternalIPBlocked  = errors.New("url points to restricted or internal network address (ssrf prevention)")
	ErrEmptyURL           = errors.New("url cannot be empty")
	ErrMalformedURL       = errors.New("malformed url")
)

// ValidateSafeURL checks that a URL is a valid http/https URL and does not target
// internal IP ranges (SSRF prevention).
func ValidateSafeURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ErrEmptyURL
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ErrMalformedURL
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return ErrInvalidURLScheme
	}

	host := u.Hostname()
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".local") {
		return ErrInternalIPBlocked
	}

	// Check if host is direct IP
	if ip := net.ParseIP(host); ip != nil {
		if isRestrictedIP(ip) {
			return ErrInternalIPBlocked
		}
		return nil
	}

	// Resolve hostname to check for internal IP rebinding
	ips, err := net.LookupIP(host)
	if err == nil {
		for _, ip := range ips {
			if isRestrictedIP(ip) {
				return fmt.Errorf("%w: resolved to %s", ErrInternalIPBlocked, ip.String())
			}
		}
	}

	return nil
}

func isRestrictedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}

	// Also block AWS/cloud metadata address: 169.254.169.254
	metadataIP := net.ParseIP("169.254.169.254")
	if ip.Equal(metadataIP) {
		return true
	}

	return false
}

// SanitizeText escapes HTML characters to protect against Stored Cross-Site Scripting (XSS).
func SanitizeText(input string) string {
	return strings.TrimSpace(html.EscapeString(input))
}

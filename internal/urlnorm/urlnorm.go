package urlnorm

import (
	"fmt"
	"net/url"
	"strings"
)

// Normalize resolves raw against base, removes fragments, and returns an
// absolute HTTP(S) URL suitable for deduplication.
func Normalize(raw string, base *url.URL) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty URL")
	}
	lower := strings.ToLower(raw)
	for _, prefix := range []string{"mailto:", "tel:", "javascript:"} {
		if strings.HasPrefix(lower, prefix) {
			return nil, fmt.Errorf("ignored URL scheme")
		}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	if base != nil {
		u = base.ResolveReference(u)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("URL has no host")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	if u.Path == "" {
		u.Path = "/"
	}
	return u, nil
}

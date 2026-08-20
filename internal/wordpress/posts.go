package wordpress

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"pressscout/internal/urlnorm"
)

const (
	postsPerPage = 100
	maxPostPages = 10000
)

type PostSeed struct {
	URL       string
	SourceURL string
}

type postRecord struct {
	Link string `json:"link"`
}

// DiscoverPosts enumerates standard WordPress post permalinks through the
// REST API. The caller's client is reused so authentication cookies apply.
func DiscoverPosts(ctx context.Context, client *http.Client, base *url.URL) ([]PostSeed, error) {
	if client == nil {
		return nil, fmt.Errorf("discover WordPress posts: HTTP client is nil")
	}
	if base == nil {
		return nil, fmt.Errorf("discover WordPress posts: base URL is nil")
	}
	normalizedBase, err := urlnorm.Normalize(base.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("discover WordPress posts: invalid base URL: %w", err)
	}
	base = normalizedBase
	baseOrigin := origin(base)
	seen := make(map[string]bool)
	seeds := make([]PostSeed, 0)
	totalPages := 0

	for page := 1; ; page++ {
		if page > maxPostPages {
			return nil, fmt.Errorf("discover WordPress posts: exceeded maximum of %d API pages", maxPostPages)
		}
		if totalPages > 0 && page > totalPages {
			break
		}
		endpoint := postsEndpoint(base, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("discover WordPress posts from %s: create request: %w", endpoint, err)
		}
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("discover WordPress posts from %s: request failed: %w", endpoint, err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("discover WordPress posts from %s: read response: %w", endpoint, readErr)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return nil, fmt.Errorf("discover WordPress posts from %s: status %s", endpoint, resp.Status)
		}
		var records []postRecord
		if err := json.Unmarshal(body, &records); err != nil {
			return nil, fmt.Errorf("discover WordPress posts from %s: decode JSON: %w", endpoint, err)
		}
		for _, record := range records {
			u, err := urlnorm.Normalize(record.Link, base)
			if err != nil || origin(u) != baseOrigin {
				continue
			}
			key := u.String()
			if seen[key] {
				continue
			}
			seen[key] = true
			seeds = append(seeds, PostSeed{URL: key, SourceURL: endpoint.String()})
		}

		if totalPages == 0 {
			header := strings.TrimSpace(resp.Header.Get("X-WP-TotalPages"))
			if header != "" {
				totalPages, err = strconv.Atoi(header)
				if err != nil || totalPages < 0 {
					return nil, fmt.Errorf("discover WordPress posts from %s: invalid X-WP-TotalPages %q", endpoint, header)
				}
			}
		}
		if len(records) == 0 || (totalPages == 0 && len(records) < postsPerPage) {
			break
		}
		if totalPages > 0 && page >= totalPages {
			break
		}
	}
	return seeds, nil
}

func postsEndpoint(base *url.URL, page int) *url.URL {
	endpoint := *base
	path := strings.TrimSuffix(endpoint.Path, "/")
	endpoint.Path = path + "/wp-json/wp/v2/posts"
	endpoint.RawPath = ""
	query := endpoint.Query()
	query.Set("_fields", "link")
	query.Set("page", strconv.Itoa(page))
	query.Set("per_page", strconv.Itoa(postsPerPage))
	endpoint.RawQuery = query.Encode()
	endpoint.Fragment = ""
	return &endpoint
}

func origin(u *url.URL) string {
	port := u.Port()
	if port == "" {
		if u.Scheme == "http" {
			port = "80"
		} else {
			port = "443"
		}
	}
	return u.Scheme + "://" + u.Hostname() + ":" + port
}

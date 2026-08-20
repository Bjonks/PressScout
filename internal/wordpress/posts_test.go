package wordpress

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDiscoverPostsPaginatesAndFilters(t *testing.T) {
	serverBase := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wp-json/wp/v2/posts" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-WP-TotalPages", "2")
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("per_page") != "100" || r.URL.Query().Get("_fields") != "link" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("page") == "1" {
			fmt.Fprint(w, `[{"link":"`+serverBase+`/first#section"},{"link":"`+serverBase+`/first"},{"link":"https://external.test/no"}]`)
			return
		}
		fmt.Fprint(w, `[{"link":"`+serverBase+`/second?view=full"}]`)
	}))
	defer server.Close()
	serverBase = server.URL
	base, _ := url.Parse(server.URL + "/")
	seeds, err := DiscoverPosts(context.Background(), &http.Client{}, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(seeds) != 2 || seeds[0].URL != server.URL+"/first" || seeds[1].URL != server.URL+"/second?view=full" {
		t.Fatalf("seeds = %+v", seeds)
	}
	if !strings.Contains(seeds[0].SourceURL, "page=1") {
		t.Fatalf("source URL = %s", seeds[0].SourceURL)
	}
}

func TestDiscoverPostsUsesCookiesAndReportsFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "session=ok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, "bad", http.StatusBadGateway)
	}))
	defer server.Close()
	base, _ := url.Parse(server.URL)
	jar, _ := cookiejar.New(nil)
	jar.SetCookies(base, []*http.Cookie{{Name: "session", Value: "ok", Path: "/"}})
	_, err := DiscoverPosts(context.Background(), &http.Client{Jar: jar}, base)
	if err == nil || !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("error = %v", err)
	}
}

func TestDiscoverPostsRejectsInvalidPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-WP-TotalPages", "not-a-number")
		fmt.Fprint(w, `[]`)
	}))
	defer server.Close()
	base, _ := url.Parse(server.URL)
	_, err := DiscoverPosts(context.Background(), &http.Client{}, base)
	if err == nil || !strings.Contains(err.Error(), "invalid X-WP-TotalPages") {
		t.Fatalf("error = %v", err)
	}
}

func TestDiscoverPostsFallsBackToShortPageWithoutHeader(t *testing.T) {
	serverBase := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := 100
		if r.URL.Query().Get("page") == "2" {
			count = 1
		}
		records := make([]postRecord, count)
		for i := range records {
			records[i].Link = fmt.Sprintf("%s/post-%s-%d", serverBase, r.URL.Query().Get("page"), i)
		}
		json.NewEncoder(w).Encode(records)
	}))
	defer server.Close()
	serverBase = server.URL
	base, _ := url.Parse(server.URL)
	seeds, err := DiscoverPosts(context.Background(), &http.Client{}, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(seeds) != 101 {
		t.Fatalf("got %d seeds, want 101", len(seeds))
	}
}

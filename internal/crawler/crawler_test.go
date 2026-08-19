package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"wpgopher/internal/checker"
	"wpgopher/internal/model"
)

func TestCrawlerRecursesInternallyAndChecksExternalOnce(t *testing.T) {
	var mu sync.Mutex
	counts := map[string]int{}
	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		counts["external"+r.URL.Path]++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<a href="/should-not-crawl">no</a>`))
	}))
	defer external.Close()
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		counts[r.URL.Path]++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			w.Write([]byte(`<a href="/page#part">page</a><a href="/page">duplicate</a><a href="/missing">missing</a><a href="` + external.URL + `/outside">external</a>`))
		case "/page":
			w.Write([]byte(`<a href="/">home</a>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer internal.Close()
	base, _ := url.Parse(internal.URL + "/")
	results, err := New(checker.New(&http.Client{}), base, 3).Crawl(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byURL := make(map[string]model.Result)
	for _, result := range results {
		byURL[result.OriginalURL] = result
	}
	if len(byURL) != 4 {
		t.Fatalf("got %d results: %+v", len(byURL), byURL)
	}
	if counts["/page"] != 1 || counts["/missing"] != 1 || counts["external/outside"] != 1 || counts["external/should-not-crawl"] != 0 {
		t.Fatalf("request counts = %v", counts)
	}
	page := byURL[internal.URL+"/page"]
	if len(page.Sources) != 1 || page.Sources[0] != internal.URL+"/" {
		t.Fatalf("page sources = %v", page.Sources)
	}
	if !byURL[external.URL+"/outside"].External {
		t.Fatal("external URL was not marked external")
	}
}

func TestCrawlerExcludesKeywordsFromURLAndAnchorText(t *testing.T) {
	var mu sync.Mutex
	counts := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		counts[r.URL.Path]++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		if r.URL.Path == "/" {
			w.Write([]byte(`<a href="/end-session">Log out</a><a href="/admin">Admin</a><a href="/signoff">Continue</a>`))
			return
		}
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	base, _ := url.Parse(server.URL + "/")
	results, err := NewWithExcludeKeywords(checker.New(&http.Client{}), base, 2, []string{"logout", "signoff"}).Crawl(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || counts["/end-session"] != 0 || counts["/signoff"] != 0 || counts["/admin"] != 1 {
		t.Fatalf("results=%v counts=%v", results, counts)
	}
}

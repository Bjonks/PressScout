package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"wpgopher/internal/model"
)

func TestCheckerClassifiesResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/ok", http.StatusFound)
		case "/forbidden":
			w.WriteHeader(http.StatusForbidden)
		case "/rate":
			w.WriteHeader(http.StatusTooManyRequests)
		case "/server":
			w.WriteHeader(http.StatusInternalServerError)
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()
	client := &http.Client{}
	c := New(client)
	tests := map[string]model.Classification{
		"/ok": model.OK, "/redirect": model.Redirect, "/forbidden": model.Forbidden,
		"/rate": model.RateLimited, "/server": model.ServerError, "/missing": model.Broken,
	}
	for path, want := range tests {
		got := c.Check(context.Background(), server.URL+path, false).Result.Class
		if got != want {
			t.Errorf("%s: got %s, want %s", path, got, want)
		}
	}
}

func TestCheckerTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(100 * time.Millisecond):
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	result := New(&http.Client{}).Check(ctx, server.URL, false).Result
	if result.Class != model.Timeout && !strings.Contains(result.Error, "context deadline exceeded") {
		t.Fatalf("result = %+v", result)
	}
}

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestLoginGetsPageBeforePostingAndRetainsCookies(t *testing.T) {
	var sequence []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sequence = append(sequence, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/wp-login.php":
			http.SetCookie(w, &http.Cookie{Name: "wordpress_test_cookie", Value: "WP"})
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/wp-login.php":
			if err := r.ParseForm(); err != nil || r.Form.Get("log") != "user" || r.Form.Get("pwd") != "pass" || r.Header.Get("Cookie") == "" {
				http.Error(w, "bad login", http.StatusForbidden)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "wordpress_logged_in_test", Value: "yes", Path: "/"})
			http.Redirect(w, r, "/home", http.StatusFound)
		case r.URL.Path == "/home":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	base, _ := url.Parse(server.URL + "/")
	client, err := Login(context.Background(), base, "user", "pass", 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sequence, ",") != "GET /wp-login.php,POST /wp-login.php,GET /home" {
		t.Fatalf("request sequence = %v", sequence)
	}
	resp, err := client.Get(server.URL + "/private")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.SetCookie(w, &http.Cookie{Name: "wordpress_test_cookie", Value: "WP"})
			return
		}
		http.Redirect(w, r, "/wp-login.php?error=1", http.StatusFound)
	}))
	defer server.Close()
	base, _ := url.Parse(server.URL)
	_, err := Login(context.Background(), base, "bad", "bad", 0)
	if err == nil || !strings.Contains(err.Error(), "WordPress authentication failed") {
		t.Fatalf("error = %v", err)
	}
}

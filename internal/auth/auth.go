package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Login creates an authenticated HTTP client using WordPress's form login.
func Login(ctx context.Context, base *url.URL, username, password string, timeout time.Duration) (*http.Client, error) {
	client, err := NewClient(timeout)
	if err != nil {
		return nil, err
	}
	jar := client.Jar
	loginURL := *base
	loginURL.Path = "/wp-login.php"
	loginURL.RawQuery = ""
	loginURL.Fragment = ""

	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, loginURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create login request: %w", err)
	}
	getResp, err := client.Do(getReq)
	if err != nil {
		return nil, fmt.Errorf("load WordPress login page: %w", err)
	}
	getResp.Body.Close()

	form := url.Values{
		"log":         {username},
		"pwd":         {password},
		"wp-submit":   {"Log In"},
		"redirect_to": {base.String()},
		"testcookie":  {"1"},
	}
	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create login submission: %w", err)
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postResp, err := client.Do(postReq)
	if err != nil {
		return nil, fmt.Errorf("submit WordPress login: %w", err)
	}
	postResp.Body.Close()

	loggedIn := false
	for _, cookie := range jar.Cookies(base) {
		if strings.HasPrefix(cookie.Name, "wordpress_logged_in_") {
			loggedIn = true
			break
		}
	}
	var finalURL *url.URL
	if postResp.Request != nil {
		finalURL = postResp.Request.URL
	}
	if !loggedIn || isLoginPage(finalURL) {
		return nil, fmt.Errorf("WordPress authentication failed: invalid credentials or login was rejected")
	}
	return client, nil
}

// NewClient returns a reusable HTTP client with a cookie jar. It is used for
// public sites when authentication is explicitly disabled.
func NewClient(timeout time.Duration) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	return &http.Client{Jar: jar, Timeout: timeout}, nil
}

func isLoginPage(u *url.URL) bool {
	return u != nil && strings.EqualFold(strings.TrimSuffix(u.Path, "/"), "/wp-login.php")
}

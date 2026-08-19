package checker

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"pressscout/internal/model"
)

type Outcome struct {
	Result      model.Result
	Body        []byte
	ContentType string
}

type Checker struct {
	Client *http.Client
}

func New(client *http.Client) *Checker { return &Checker{Client: client} }

func (c *Checker) Check(ctx context.Context, original string, external bool) Outcome {
	out := Outcome{Result: model.Result{OriginalURL: original, FinalURL: original, External: external}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, original, nil)
	if err != nil {
		out.Result.Error = err.Error()
		out.Result.Class = model.NetworkError
		return out
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		out.Result.Error = err.Error()
		if errors.Is(err, context.DeadlineExceeded) || isTimeout(err) {
			out.Result.Class = model.Timeout
		} else {
			out.Result.Class = model.NetworkError
		}
		return out
	}
	defer resp.Body.Close()
	out.Result.Status = resp.StatusCode
	if resp.Request != nil && resp.Request.URL != nil {
		out.Result.FinalURL = resp.Request.URL.String()
	} else {
		out.Result.FinalURL = original
	}
	out.ContentType = resp.Header.Get("Content-Type")
	out.Body, err = io.ReadAll(resp.Body)
	if err != nil {
		out.Result.Error = err.Error()
		out.Result.Class = model.NetworkError
		return out
	}
	out.Result.Class = classify(resp.StatusCode, original, out.Result.FinalURL)
	return out
}

func classify(status int, original, final string) model.Classification {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return model.Forbidden
	case status == http.StatusTooManyRequests:
		return model.RateLimited
	case status >= 500 && status <= 599:
		return model.ServerError
	case status >= 400 && status <= 499:
		return model.Broken
	case status >= 300 && status <= 399:
		return model.Redirect
	case status >= 200 && status <= 299 && final != "" && final != original:
		return model.Redirect
	case status >= 200 && status <= 299:
		return model.OK
	default:
		return model.Broken
	}
}

func isTimeout(err error) bool {
	var timeout interface{ Timeout() bool }
	return errors.As(err, &timeout) && timeout.Timeout()
}

func IsHTML(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "text/html") || contentType == ""
}

func ParseURL(raw string) (*url.URL, error) { return url.Parse(raw) }

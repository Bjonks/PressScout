# PressScout

`PressScout` checks links on an authenticated internal WordPress site. It logs in through `/wp-login.php` using `net/http` and a cookie jar, crawls same-origin HTML pages, checks external links without crawling them, and reports non-OK results.

## Build

```bash
go build ./cmd/pressscout
```

## Usage

Set the WordPress credentials in the environment:

```bash
export WP_USER='username'
export WP_PASS='password'
./pressscout --concurrency 15 --timeout 15s --json results.json https://wordpress.internal/
```

Flags:

- `--concurrency`: maximum concurrent crawl requests; default `15`.
- `--timeout`: per-request timeout; default `15s`.
- `--json FILE`: write the complete report to `FILE` in addition to the concise stdout summary.
- `--no-auth`: skip WordPress authentication for a public site. This is disabled by default.
- `--exclude-keyword WORD`: skip discovered links whose URL or visible anchor text contains `WORD` (case-insensitive); repeat the flag for multiple keywords.

The program first performs a `GET /wp-login.php`, then submits the credentials with `POST /wp-login.php`. It requires a WordPress logged-in cookie before crawling. Credentials are read only from `WP_USER` and `WP_PASS` and are never included in reports.

For a public site, omit the credential variables and use:

```bash
./pressscout --no-auth https://example.com/
```

To avoid following logout links while crawling an authenticated site:

```bash
./pressscout --exclude-keyword logout --exclude-keyword signout https://wordpress.internal/
```

Classifications are `OK`, `REDIRECT`, `BROKEN`, `FORBIDDEN`, `RATE_LIMITED`, `SERVER_ERROR`, `TIMEOUT`, and `NETWORK_ERROR`. Text output includes summary counts and all non-OK URLs with their discovery source pages.

## Development checks

```bash
gofmt -w cmd internal
go mod tidy
go vet ./...
go test ./...
go test -race ./...
```

The tests use local `httptest` servers and do not require a live WordPress site or internet access.

## Current limitations

The crawler does not implement Playwright or other browser automation, SAML, JavaScript rendering, SQLite persistence, or fragment-anchor validation. URL fragments are removed before checks, while query strings are preserved.

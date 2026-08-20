# Changelog

All notable changes to PressScout are documented here.

## [0.1.0] - 2026-08-20

### Added

- Authenticated WordPress login through `/wp-login.php` using `net/http` and a cookie jar.
- Recursive same-origin crawling with external-link checking.
- Bounded concurrent requests with configurable concurrency and timeout settings.
- URL normalization, fragment removal, query preservation, deduplication, and source-page tracking.
- Result classifications for successful, redirected, broken, forbidden, rate-limited, server-error, timeout, and network-error links.
- Concise text reporting and optional JSON output.
- `--no-auth` mode for public sites.
- Repeatable `--exclude-keyword` filtering for URL and anchor text, including logout links.
- Opt-in `--crawl-posts` discovery through the paginated WordPress REST API.
- Unit, integration, and race-detection test coverage.

### Changed

- Renamed the project and executable to `pressscout`.

### Other

- Added the project license.

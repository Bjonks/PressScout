package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"pressscout/internal/auth"
	"pressscout/internal/checker"
	"pressscout/internal/crawler"
	"pressscout/internal/report"
	"pressscout/internal/urlnorm"
)

func main() {
	concurrency := flag.Int("concurrency", 15, "maximum concurrent HTTP requests")
	timeout := flag.Duration("timeout", 15*time.Second, "HTTP request timeout")
	jsonFile := flag.String("json", "", "write a JSON report to this file")
	noAuth := flag.Bool("no-auth", false, "skip WordPress authentication for a public site")
	var excludeKeywords stringList
	flag.Var(&excludeKeywords, "exclude-keyword", "skip links containing this keyword in URL or anchor text (repeatable)")
	flag.Parse()
	if flag.NArg() != 1 {
		fail("usage: pressscout [--concurrency N] [--timeout DURATION] [--json FILE] [--no-auth] BASE_URL")
	}
	if *concurrency < 1 {
		fail("--concurrency must be at least 1")
	}
	if *timeout <= 0 {
		fail("--timeout must be positive")
	}
	base, err := urlnorm.Normalize(flag.Arg(0), nil)
	if err != nil {
		fail("invalid base URL: " + err.Error())
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var client *http.Client
	if *noAuth {
		client, err = auth.NewClient(*timeout)
	} else {
		user, pass := os.Getenv("WP_USER"), os.Getenv("WP_PASS")
		if user == "" || pass == "" {
			fail("WP_USER and WP_PASS environment variables are required (or use --no-auth for a public site)")
		}
		client, err = auth.Login(ctx, base, user, pass, *timeout)
	}
	if err != nil {
		fail(err.Error())
	}
	results, err := crawler.NewWithExcludeKeywords(checker.New(client), base, *concurrency, excludeKeywords).Crawl(ctx)
	if err != nil {
		fail("crawl failed: " + err.Error())
	}
	if err := report.PrintText(os.Stdout, results); err != nil {
		fail("print report: " + err.Error())
	}
	if *jsonFile != "" {
		if err := report.WriteJSON(*jsonFile, results); err != nil {
			fail(err.Error())
		}
	}
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(2)
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

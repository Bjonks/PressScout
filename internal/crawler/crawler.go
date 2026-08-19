package crawler

import (
	"bytes"
	"context"
	"net/url"
	"sort"
	"strings"
	"sync"

	"wpgopher/internal/checker"
	"wpgopher/internal/model"
	"wpgopher/internal/urlnorm"
)

type Crawler struct {
	Checker     *checker.Checker
	Concurrency int
	Base        *url.URL
	Exclude     []string
}

func New(c *checker.Checker, base *url.URL, concurrency int) *Crawler {
	return NewWithExcludeKeywords(c, base, concurrency, nil)
}

func NewWithExcludeKeywords(c *checker.Checker, base *url.URL, concurrency int, keywords []string) *Crawler {
	if concurrency < 1 {
		concurrency = 1
	}
	cleaned := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		if keyword = strings.TrimSpace(strings.ToLower(keyword)); keyword != "" {
			cleaned = append(cleaned, keyword)
		}
	}
	return &Crawler{Checker: c, Base: base, Concurrency: concurrency, Exclude: cleaned}
}

type job struct {
	URL      string
	External bool
}

type workerOutcome struct {
	job job
	out checker.Outcome
}

func (c *Crawler) Crawl(ctx context.Context) ([]model.Result, error) {
	jobs := make(chan job, c.Concurrency)
	outcomes := make(chan workerOutcome, c.Concurrency)
	var wg sync.WaitGroup
	for i := 0; i < c.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				outcomes <- workerOutcome{job: j, out: c.Checker.Check(ctx, j.URL, j.External)}
			}
		}()
	}

	seen := make(map[string]bool)
	sources := make(map[string]map[string]bool)
	results := make(map[string]model.Result)
	pending := []job{{URL: c.Base.String()}}
	seen[c.Base.String()] = true
	inFlight := 0
	baseOrigin := origin(c.Base)

	enqueue := func(raw, source string) {
		u, err := urlnorm.Normalize(raw, c.Base)
		if err != nil {
			return
		}
		key := u.String()
		if sources[key] == nil {
			sources[key] = make(map[string]bool)
		}
		if source != "" {
			sources[key][source] = true
		}
		if seen[key] {
			return
		}
		seen[key] = true
		pending = append(pending, job{URL: key, External: origin(u) != baseOrigin})
	}

	for len(pending) > 0 || inFlight > 0 {
		var send chan job
		var next job
		if len(pending) > 0 && inFlight < c.Concurrency {
			send = jobs
			next = pending[0]
		}
		select {
		case send <- next:
			pending = pending[1:]
			inFlight++
		case item := <-outcomes:
			inFlight--
			result := item.out.Result
			for source := range sources[item.job.URL] {
				result.Sources = append(result.Sources, source)
			}
			sort.Strings(result.Sources)
			results[item.job.URL] = result
			finalURL, err := url.Parse(result.FinalURL)
			if err != nil || item.job.External || result.Class != model.OK && result.Class != model.Redirect || !checker.IsHTML(item.out.ContentType) {
				continue
			}
			if origin(finalURL) != baseOrigin {
				continue
			}
			links, err := ExtractLinksWithTextBytes(item.out.Body, finalURL)
			if err != nil {
				continue
			}
			for _, link := range links {
				if c.excluded(link) {
					continue
				}
				enqueue(link.URL, item.job.URL)
			}
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil, ctx.Err()
		}
	}
	close(jobs)
	wg.Wait()
	answer := make([]model.Result, 0, len(results))
	for _, result := range results {
		answer = append(answer, result)
	}
	sort.Slice(answer, func(i, j int) bool { return answer[i].OriginalURL < answer[j].OriginalURL })
	return answer, nil
}

func ExtractLinksBytes(body []byte, page *url.URL) ([]string, error) {
	return ExtractLinks(bytes.NewReader(body), page)
}

func ExtractLinksWithTextBytes(body []byte, page *url.URL) ([]Link, error) {
	return ExtractLinksWithText(bytes.NewReader(body), page)
}

func (c *Crawler) excluded(link Link) bool {
	value := strings.ToLower(link.URL + "\n" + link.Text)
	compactValue := strings.Join(strings.Fields(value), "")
	for _, keyword := range c.Exclude {
		compactKeyword := strings.Join(strings.Fields(keyword), "")
		if strings.Contains(value, keyword) || strings.Contains(compactValue, compactKeyword) {
			return true
		}
	}
	return false
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

package crawler

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"wpgopher/internal/urlnorm"
)

type Link struct {
	URL  string
	Text string
}

// ExtractLinks returns normalized HTTP(S) links found in anchor elements.
func ExtractLinks(r io.Reader, page *url.URL) ([]string, error) {
	rawLinks, err := ExtractLinksWithText(r, page)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	links := make([]string, 0, len(rawLinks))
	for _, link := range rawLinks {
		if _, ok := seen[link.URL]; ok {
			continue
		}
		seen[link.URL] = struct{}{}
		links = append(links, link.URL)
	}
	return links, nil
}

// ExtractLinksWithText returns normalized links and their visible anchor text.
// Duplicate URLs are retained so callers can filter one anchor without hiding
// another anchor to the same URL.
func ExtractLinksWithText(r io.Reader, page *url.URL) ([]Link, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	links := make([]Link, 0)
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key != "href" {
					continue
				}
				u, err := urlnorm.Normalize(attr.Val, page)
				if err != nil {
					continue
				}
				links = append(links, Link{URL: u.String(), Text: anchorText(n)})
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return links, nil
}

func anchorText(n *html.Node) string {
	var parts []string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			parts = append(parts, node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
}

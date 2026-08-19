package crawler

import (
	"net/url"
	"strings"
	"testing"
)

func TestExtractLinks(t *testing.T) {
	page, _ := url.Parse("https://example.test/docs/index")
	html := `<a href="/one#part">one</a><div><a href="../two?q=1">two</a><a href="mailto:x@y">mail</a><a href="/one">duplicate</a></div>`
	got, err := ExtractLinks(strings.NewReader(html), page)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"https://example.test/one": true, "https://example.test/two?q=1": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, link := range got {
		if !want[link] {
			t.Errorf("unexpected link %q", link)
		}
	}
}

func TestExtractLinksWithText(t *testing.T) {
	page, _ := url.Parse("https://example.test/")
	links, err := ExtractLinksWithText(strings.NewReader(`<a href="/logout"><span>Log</span> out</a>`), page)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].URL != "https://example.test/logout" || links[0].Text != "Log out" {
		t.Fatalf("links = %+v", links)
	}
}

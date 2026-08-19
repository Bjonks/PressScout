package urlnorm

import (
	"net/url"
	"testing"
)

func TestNormalize(t *testing.T) {
	base, _ := url.Parse("https://EXAMPLE.test/docs/page?keep=1#old")
	tests := []struct{ raw, want string }{
		{"../other#section", "https://example.test/other"},
		{"/asset?q=2#x", "https://example.test/asset?q=2"},
		{"https://Other.test/x?b=1&a=2", "https://other.test/x?b=1&a=2"},
	}
	for _, tt := range tests {
		got, err := Normalize(tt.raw, base)
		if err != nil || got.String() != tt.want {
			t.Errorf("Normalize(%q) = %v, %v; want %q", tt.raw, got, err, tt.want)
		}
	}
}

func TestNormalizeIgnoresSchemes(t *testing.T) {
	for _, raw := range []string{"", "mailto:a@example.test", "tel:123", "javascript:void(0)"} {
		if _, err := Normalize(raw, nil); err == nil {
			t.Errorf("Normalize(%q) unexpectedly succeeded", raw)
		}
	}
}

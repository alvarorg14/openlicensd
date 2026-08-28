package clientip_test

import (
	"net/http"
	"testing"

	"github.com/alvarorg14/openlicensd/server/internal/clientip"
)

func TestNewResolverInvalidEntry(t *testing.T) {
	_, err := clientip.NewResolver([]string{"not-an-ip"})
	if err == nil {
		t.Fatal("expected error for invalid trusted proxy entry")
	}
}

func TestResolverFrom(t *testing.T) {
	tests := []struct {
		name     string
		trusted  []string
		remote   string
		xff      string
		expected string
	}{
		{
			name:     "no trusted proxies ignores xff",
			trusted:  nil,
			remote:   "203.0.113.10:12345",
			xff:      "198.51.100.5",
			expected: "203.0.113.10",
		},
		{
			name:     "trusted peer uses rightmost untrusted xff",
			trusted:  []string{"10.0.0.0/8"},
			remote:   "10.1.2.3:8080",
			xff:      "198.51.100.5, 10.1.2.3",
			expected: "198.51.100.5",
		},
		{
			name:     "trusted peer falls back to remote when xff absent",
			trusted:  []string{"10.0.0.0/8"},
			remote:   "10.1.2.3:8080",
			expected: "10.1.2.3",
		},
		{
			name:     "untrusted peer ignores xff",
			trusted:  []string{"10.0.0.0/8"},
			remote:   "203.0.113.10:12345",
			xff:      "198.51.100.5",
			expected: "203.0.113.10",
		},
		{
			name:     "malformed xff falls back to remote",
			trusted:  []string{"10.0.0.0/8"},
			remote:   "10.1.2.3:8080",
			xff:      "not-an-ip",
			expected: "10.1.2.3",
		},
		{
			name:     "ipv6 remote without port",
			trusted:  nil,
			remote:   "2001:db8::1",
			expected: "2001:db8::1",
		},
		{
			name:     "single trusted proxy ip",
			trusted:  []string{"10.1.2.3"},
			remote:   "10.1.2.3:8080",
			xff:      "198.51.100.5",
			expected: "198.51.100.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := clientip.NewResolver(tt.trusted)
			if err != nil {
				t.Fatalf("new resolver: %v", err)
			}

			req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			req.RemoteAddr = tt.remote
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}

			got := resolver.From(req)
			if got != tt.expected {
				t.Fatalf("From()=%q want %q", got, tt.expected)
			}
		})
	}
}

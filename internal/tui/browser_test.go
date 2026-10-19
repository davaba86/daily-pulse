package tui

import "testing"

func TestBrowserCommand(t *testing.T) {
	cases := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{"darwin", "open", []string{"https://example.com"}},
		{"linux", "xdg-open", []string{"https://example.com"}},
		{"freebsd", "xdg-open", []string{"https://example.com"}},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", "https://example.com"}},
	}
	for _, c := range cases {
		name, args := browserCommand(c.goos, "https://example.com")
		if name != c.wantName || len(args) != len(c.wantArgs) {
			t.Errorf("browserCommand(%q, ...) = (%q, %v), want (%q, %v)", c.goos, name, args, c.wantName, c.wantArgs)
			continue
		}
		for i := range args {
			if args[i] != c.wantArgs[i] {
				t.Errorf("browserCommand(%q, ...) args[%d] = %q, want %q", c.goos, i, args[i], c.wantArgs[i])
			}
		}
	}
}

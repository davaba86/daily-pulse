package tui

import (
	"os/exec"
	"runtime"
)

// opener opens a URL in the user's default browser. Tests inject a fake;
// openInBrowser is the real implementation, wired in by Run.
type opener func(url string) error

func browserCommand(goos, url string) (name string, args []string) {
	switch goos {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return "xdg-open", []string{url}
	}
}

func openInBrowser(url string) error {
	name, args := browserCommand(runtime.GOOS, url)
	return exec.Command(name, args...).Start()
}

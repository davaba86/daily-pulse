package render

import (
	"strings"
	"testing"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
	"github.com/fatih/color"
)

func fixedNow() time.Time {
	return time.Date(2026, 10, 18, 8, 2, 0, 0, time.UTC)
}

func TestFormatTable_Sections(t *testing.T) {
	now := fixedNow()
	prs := []model.PullRequest{
		{
			Source: "github", Repo: "davaba86/mc-platform-cloud", Number: 482,
			Title: "fix-auth token refresh", Reviewers: []string{"jsmith"},
			UpdatedAt: now.Add(-12 * time.Minute),
		},
		{
			Source: "gitlab", Repo: "mecha-consulting/mc-product-lifttraq", Number: 91,
			Title: "add SSO login flow", Reviewers: []string{"agupta"},
			UpdatedAt: now.Add(-5 * time.Minute),
		},
	}
	pipelines := []model.PipelineRun{
		{
			Repo: "davaba86/mc-platform-cloud", Status: model.PipelinePass, Branch: "main",
			Duration: 3*time.Minute + 12*time.Second, UpdatedAt: now.Add(-12 * time.Minute),
		},
	}

	out := FormatTable(prs, pipelines, now)

	for _, want := range []string{
		"GITHUB", "open PRs", "482", "fix-auth token refresh", "jsmith", "12m ago",
		"GITLAB", "open MRs", "91", "add SSO login flow", "agupta", "5m ago",
		"PIPELINES", "davaba86/mc-platform-cloud", "pass", "main", "3m12s",
		"2 open PRs/MRs",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatTable() missing %q in output:\n%s", want, out)
		}
	}
}

func TestFormatTable_NoPullRequests(t *testing.T) {
	out := FormatTable(nil, nil, fixedNow())
	if !strings.Contains(out, "0 open PRs/MRs") {
		t.Errorf("FormatTable() with no PRs missing summary line, got:\n%s", out)
	}
}

func TestRelativeTime(t *testing.T) {
	now := fixedNow()
	cases := []struct {
		delta time.Duration
		want  string
	}{
		{30 * time.Second, "30s ago"},
		{5 * time.Minute, "5m ago"},
		{2 * time.Hour, "2h ago"},
		{3 * 24 * time.Hour, "3d ago"},
	}
	for _, c := range cases {
		got := relativeTime(now.Add(-c.delta), now)
		if got != c.want {
			t.Errorf("relativeTime(-%s) = %q, want %q", c.delta, got, c.want)
		}
	}
}

func TestColorize(t *testing.T) {
	color.NoColor = false
	defer func() { color.NoColor = true }()

	if got := Colorize("pass"); !strings.Contains(got, "\x1b[") {
		t.Errorf("Colorize(%q) = %q, want ANSI color codes", "pass", got)
	}
	if got := Colorize("plain text"); got != "plain text" {
		t.Errorf("Colorize(%q) = %q, want unchanged", "plain text", got)
	}
}

func TestMain(m *testing.M) {
	color.NoColor = true // deterministic output in all other tests
	m.Run()
}

# daily-pulse Phase 1 (PR/MR + Pipeline Dashboard) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `daily-pulse` CLI's Phase 1 MVP — a single command that prints a daily overview of open GitHub PRs / GitLab MRs and the latest default-branch pipeline status for a configured set of repos, filterable to "mine" vs. everyone, with aligned/colored static tables and a local SQLite snapshot written on every run for later benchmarking (Phase 3).

**Architecture:** `cmd/root.go` (cobra) loads YAML config, fetches data from either `internal/mockdata` (`--mock` flag, used first to lock down rendering) or `internal/githost` (real `gh`/`glab` CLI calls, injected via a `Runner` interface so fetch logic is unit-testable without live auth), renders via `internal/render` (stdlib `text/tabwriter` for alignment + `fatih/color` for status coloring), and persists the fetched snapshot via `internal/store` (SQLite, `modernc.org/sqlite`, no cgo).

**Tech Stack:** Go, `github.com/spf13/cobra` (CLI), `gopkg.in/yaml.v3` (config), `github.com/fatih/color` (terminal color), `modernc.org/sqlite` + `database/sql` (storage), stdlib `text/tabwriter` (column alignment).

Spec: `docs/superpowers/specs/2026-10-18-daily-pulse-design.md`

---

## File Structure

```
go.mod / go.sum
main.go                              entrypoint, calls cmd.Execute()
LICENSE                               MIT
README.md                             usage, config example, auth prereqs
.gitignore

internal/model/model.go               shared domain types: PullRequest, PipelineRun

internal/config/config.go             YAML schema + Load(path)
internal/config/config_test.go

internal/mockdata/mockdata.go         hardcoded sample data matching the approved mockup
internal/mockdata/mockdata_test.go

internal/render/render.go             FormatTable(prs, pipelines, now) string; Colorize(s) string
internal/render/render_test.go

internal/githost/runner.go            Runner type + ExecRunner (shells to gh/glab)
internal/githost/github.go            GitHub parse + fetch functions (PRs, pipeline status)
internal/githost/github_test.go
internal/githost/gitlab.go            GitLab parse + fetch functions (MRs, pipeline status)
internal/githost/gitlab_test.go
internal/githost/testdata/*.json      fixture JSON files

internal/store/store.go               SQLite schema + Open(path) + SaveSnapshot(db, prs, pipelines, takenAt)
internal/store/store_test.go

cmd/root.go                           cobra root command wiring everything together
cmd/root_test.go
```

---

### Task 1: Repo scaffold

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `LICENSE`
- Create: `README.md`
- Create: `main.go`

- [ ] **Step 1: Initialize the Go module**

Run: `cd /Users/davaba/git-repos/daily-pulse && go mod init github.com/davaba86/daily-pulse`
Expected: `go: creating new go.mod: module github.com/davaba86/daily-pulse`

- [ ] **Step 2: Add `.gitignore`**

```gitignore
/daily-pulse
*.db
*.db-journal
```

- [ ] **Step 3: Add MIT `LICENSE`**

```text
MIT License

Copyright (c) 2026 David Abarca

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 4: Add placeholder `README.md`**

```markdown
# daily-pulse

A CLI daily overview of your open GitHub PRs, GitLab MRs, and repo pipeline
status — one command instead of tab-hopping between web UIs.

Full usage docs land at the end of this plan's implementation.
```

- [ ] **Step 5: Add `main.go`**

```go
package main

import "github.com/davaba86/daily-pulse/cmd"

func main() {
	cmd.Execute()
}
```

(This will fail to compile until Task 7 creates `cmd.Execute` — that's expected at this point, don't run `go build` yet.)

- [ ] **Step 6: Commit**

```bash
git add go.mod .gitignore LICENSE README.md main.go
git commit -m "chore: scaffold daily-pulse Go module"
```

---

### Task 2: Domain model

**Files:**
- Create: `internal/model/model.go`

- [ ] **Step 1: Write the types**

No test needed — this file is plain type declarations with no behavior.

```go
package model

import "time"

type PullRequest struct {
	Source    string // "github" or "gitlab"
	Repo      string // "owner/name" for GitHub, "group/project" path for GitLab
	Number    int
	Title     string
	URL       string
	Author    string
	Reviewers []string
	UpdatedAt time.Time
}

type PipelineStatus string

const (
	PipelinePass    PipelineStatus = "pass"
	PipelineFail    PipelineStatus = "fail"
	PipelineRunning PipelineStatus = "running"
	PipelineNone    PipelineStatus = "none"
)

type PipelineRun struct {
	Repo      string
	Status    PipelineStatus
	Branch    string
	Duration  time.Duration
	UpdatedAt time.Time
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/model/`
Expected: no output (success)

- [ ] **Step 3: Commit**

```bash
git add internal/model/model.go
git commit -m "feat: add shared domain model types"
```

---

### Task 3: Config loader

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	contents := `
github:
  username: davaba86
  repos:
    - owner: davaba86
      name: daily-pulse
gitlab:
  username: david.abarca
  projects:
    - path: mecha-consulting/mc-platform-cloud
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.GitHub.Username != "davaba86" {
		t.Errorf("GitHub.Username = %q, want %q", cfg.GitHub.Username, "davaba86")
	}
	if len(cfg.GitHub.Repos) != 1 || cfg.GitHub.Repos[0].Owner != "davaba86" || cfg.GitHub.Repos[0].Name != "daily-pulse" {
		t.Errorf("GitHub.Repos = %+v, want one repo davaba86/daily-pulse", cfg.GitHub.Repos)
	}
	if cfg.GitLab.Username != "david.abarca" {
		t.Errorf("GitLab.Username = %q, want %q", cfg.GitLab.Username, "david.abarca")
	}
	if len(cfg.GitLab.Projects) != 1 || cfg.GitLab.Projects[0].Path != "mecha-consulting/mc-platform-cloud" {
		t.Errorf("GitLab.Projects = %+v, want one project mecha-consulting/mc-platform-cloud", cfg.GitLab.Projects)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Load() error = nil, want error for missing file")
	}
}

func TestLoad_NoReposConfigured(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("github:\n  username: davaba86\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error when no repos/projects are configured")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -v`
Expected: FAIL — `package config: no Go files` or similar (package doesn't exist yet)

- [ ] **Step 3: Write the implementation**

```go
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub GitHubConfig `yaml:"github"`
	GitLab GitLabConfig `yaml:"gitlab"`
}

type GitHubConfig struct {
	Username string           `yaml:"username"`
	Repos    []GitHubRepoSpec `yaml:"repos"`
}

type GitHubRepoSpec struct {
	Owner string `yaml:"owner"`
	Name  string `yaml:"name"`
}

type GitLabConfig struct {
	Username string             `yaml:"username"`
	Projects []GitLabProjectSpec `yaml:"projects"`
}

type GitLabProjectSpec struct {
	Path string `yaml:"path"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if len(cfg.GitHub.Repos) == 0 && len(cfg.GitLab.Projects) == 0 {
		return nil, fmt.Errorf("config %s: no github.repos or gitlab.projects configured", path)
	}

	return &cfg, nil
}
```

- [ ] **Step 4: Add the yaml dependency**

Run: `go get gopkg.in/yaml.v3`
Expected: `go.mod`/`go.sum` updated with a `gopkg.in/yaml.v3` require entry

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/config/... -v`
Expected: PASS (all three tests)

- [ ] **Step 6: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: add YAML config loader"
```

---

### Task 4: Mock data

**Files:**
- Create: `internal/mockdata/mockdata.go`
- Test: `internal/mockdata/mockdata_test.go`

This fixture backs the `--mock` flag so rendering can be built and visually
checked against the approved mockup before any real `gh`/`glab` calls exist.

- [ ] **Step 1: Write the failing test**

```go
package mockdata

import "testing"

func TestPullRequests(t *testing.T) {
	prs := PullRequests()
	if len(prs) != 3 {
		t.Fatalf("len(PullRequests()) = %d, want 3", len(prs))
	}
}

func TestPipelineRuns(t *testing.T) {
	runs := PipelineRuns()
	if len(runs) != 2 {
		t.Fatalf("len(PipelineRuns()) = %d, want 2", len(runs))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mockdata/... -v`
Expected: FAIL — package/functions don't exist yet

- [ ] **Step 3: Write the implementation**

```go
package mockdata

import (
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

func PullRequests() []model.PullRequest {
	now := time.Now()
	return []model.PullRequest{
		{
			Source:    "github",
			Repo:      "davaba86/mc-platform-cloud",
			Number:    482,
			Title:     "fix-auth token refresh",
			URL:       "https://github.com/davaba86/mc-platform-cloud/pull/482",
			Author:    "davaba86",
			Reviewers: []string{"jsmith"},
			UpdatedAt: now.Add(-12 * time.Minute),
		},
		{
			Source:    "github",
			Repo:      "davaba86/mc-platform-onprem",
			Number:    478,
			Title:     "bump terraform providers",
			URL:       "https://github.com/davaba86/mc-platform-onprem/pull/478",
			Author:    "davaba86",
			Reviewers: nil,
			UpdatedAt: now.Add(-48 * time.Hour),
		},
		{
			Source:    "gitlab",
			Repo:      "mecha-consulting/mc-product-lifttraq",
			Number:    91,
			Title:     "add SSO login flow",
			URL:       "https://gitlab.com/mecha-consulting/mc-product-lifttraq/-/merge_requests/91",
			Author:    "david.abarca",
			Reviewers: []string{"agupta"},
			UpdatedAt: now.Add(-5 * time.Minute),
		},
	}
}

func PipelineRuns() []model.PipelineRun {
	now := time.Now()
	return []model.PipelineRun{
		{
			Repo:      "davaba86/mc-platform-cloud",
			Status:    model.PipelinePass,
			Branch:    "main",
			Duration:  3*time.Minute + 12*time.Second,
			UpdatedAt: now.Add(-12 * time.Minute),
		},
		{
			Repo:      "davaba86/mc-platform-onprem",
			Status:    model.PipelineFail,
			Branch:    "main",
			Duration:  1*time.Minute + 48*time.Second,
			UpdatedAt: now.Add(-34 * time.Minute),
		},
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mockdata/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mockdata/
git commit -m "feat: add mock dataset for rendering development"
```

---

### Task 5: Table rendering

**Files:**
- Create: `internal/render/render.go`
- Test: `internal/render/render_test.go`

- [ ] **Step 1: Write the failing test**

```go
package render

import (
	"strings"
	"testing"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/render/... -v`
Expected: FAIL — package/functions don't exist yet

- [ ] **Step 3: Write the implementation**

```go
package render

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

func relativeTime(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func reviewersCell(reviewers []string) string {
	if len(reviewers) == 0 {
		return "—"
	}
	return strings.Join(reviewers, ", ")
}

func writePRSection(buf *bytes.Buffer, title string, prs []model.PullRequest, numberHeader string, now time.Time) {
	if len(prs) == 0 {
		return
	}
	fmt.Fprintf(buf, "%s\n", title)
	tw := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "REPO\t%s\tTITLE\tREVIEWERS\tUPDATED\n", numberHeader)
	for _, pr := range prs {
		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\n",
			pr.Repo, pr.Number, pr.Title, reviewersCell(pr.Reviewers), relativeTime(pr.UpdatedAt, now))
	}
	tw.Flush()
	buf.WriteString("\n")
}

func writePipelineSection(buf *bytes.Buffer, pipelines []model.PipelineRun, now time.Time) {
	if len(pipelines) == 0 {
		return
	}
	buf.WriteString("PIPELINES  (main/default branch)\n")
	tw := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "REPO\tSTATUS\tBRANCH\tDURATION\tUPDATED\n")
	for _, p := range pipelines {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			p.Repo, string(p.Status), p.Branch, p.Duration.String(), relativeTime(p.UpdatedAt, now))
	}
	tw.Flush()
	buf.WriteString("\n")
}

func FormatTable(prs []model.PullRequest, pipelines []model.PipelineRun, now time.Time) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "daily-pulse · %s\n\n", now.Format("2006-01-02 15:04"))

	var githubPRs, gitlabPRs []model.PullRequest
	for _, pr := range prs {
		switch pr.Source {
		case "github":
			githubPRs = append(githubPRs, pr)
		case "gitlab":
			gitlabPRs = append(gitlabPRs, pr)
		}
	}

	writePRSection(&buf, "GITHUB · open PRs", githubPRs, "#", now)
	writePRSection(&buf, "GITLAB · open MRs", gitlabPRs, "!", now)
	writePipelineSection(&buf, pipelines, now)

	failing := 0
	for _, p := range pipelines {
		if p.Status == model.PipelineFail {
			failing++
		}
	}
	fmt.Fprintf(&buf, "%d open PRs/MRs · %d pipeline failing\n", len(prs), failing)

	return buf.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/render/... -v`
Expected: PASS (all four tests)

- [ ] **Step 5: Add colorization**

Write the failing test first — append to `internal/render/render_test.go`:

```go
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
```

Add the import to the top of `internal/render/render_test.go`:

```go
	"github.com/fatih/color"
```

Run: `go test ./internal/render/... -v`
Expected: FAIL — `Colorize` undefined

Now add to `internal/render/render.go` (new import `"github.com/fatih/color"`):

```go
var (
	greenBold  = color.New(color.FgGreen, color.Bold).SprintFunc()
	redBold    = color.New(color.FgRed, color.Bold).SprintFunc()
	yellowBold = color.New(color.FgYellow, color.Bold).SprintFunc()
)

// Colorize wraps known status tokens with ANSI color. It operates on
// already tabwriter-aligned text: ANSI escapes are zero-width in a
// terminal, so recoloring a token in place never shifts column alignment
// that tabwriter already computed from the plain-text widths.
func Colorize(s string) string {
	s = strings.ReplaceAll(s, "pass", greenBold("pass"))
	s = strings.ReplaceAll(s, "fail", redBold("fail"))
	s = strings.ReplaceAll(s, "running", yellowBold("running"))
	return s
}
```

Run: `go get github.com/fatih/color`
Expected: `go.mod`/`go.sum` updated with a `github.com/fatih/color` require entry

Run: `go test ./internal/render/... -v`
Expected: PASS (all tests)

- [ ] **Step 6: Commit**

```bash
git add internal/render/ go.mod go.sum
git commit -m "feat: add table rendering with aligned columns and status coloring"
```

---

### Task 6: `gh`/`glab` command runner

**Files:**
- Create: `internal/githost/runner.go`

- [ ] **Step 1: Write the implementation**

No dedicated unit test — `ExecRunner` is a one-line pass-through to
`os/exec`, verified by manual smoke test once Task 9 wires real fetches.
The `Runner` type is what makes Tasks 7–8 unit-testable without live auth.

```go
package githost

import "os/exec"

// Runner runs an external CLI (gh or glab) with the given arguments and
// returns its stdout. ExecRunner is the real implementation; tests supply
// a fake to avoid needing live gh/glab authentication.
type Runner func(name string, args ...string) ([]byte, error)

func ExecRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/githost/`
Expected: no output (success)

- [ ] **Step 3: Commit**

```bash
git add internal/githost/runner.go
git commit -m "feat: add injectable command runner for gh/glab calls"
```

---

### Task 7: GitHub fetch (PRs + pipeline status)

**Files:**
- Create: `internal/githost/github.go`
- Test: `internal/githost/github_test.go`
- Create fixtures: `internal/githost/testdata/github_pulls.json`, `internal/githost/testdata/github_repo.json`, `internal/githost/testdata/github_runs.json`, `internal/githost/testdata/github_runs_empty.json`

- [ ] **Step 1: Add fixture files**

`internal/githost/testdata/github_pulls.json`:

```json
[
  {
    "number": 482,
    "title": "fix-auth token refresh",
    "html_url": "https://github.com/davaba86/mc-platform-cloud/pull/482",
    "updated_at": "2026-10-18T07:50:00Z",
    "user": { "login": "davaba86" },
    "requested_reviewers": [{ "login": "jsmith" }]
  }
]
```

`internal/githost/testdata/github_repo.json`:

```json
{ "default_branch": "main" }
```

`internal/githost/testdata/github_runs.json`:

```json
{
  "workflow_runs": [
    {
      "status": "completed",
      "conclusion": "success",
      "head_branch": "main",
      "run_started_at": "2026-10-18T07:48:48Z",
      "updated_at": "2026-10-18T07:52:00Z"
    }
  ]
}
```

`internal/githost/testdata/github_runs_empty.json`:

```json
{ "workflow_runs": [] }
```

- [ ] **Step 2: Write the failing test**

```go
package githost

import (
	"os"
	"testing"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

func fakeRunner(t *testing.T, byArgsPrefix map[string]string) Runner {
	return func(name string, args ...string) ([]byte, error) {
		for prefix, fixture := range byArgsPrefix {
			if len(args) > 0 && args[len(args)-1] == prefix {
				return os.ReadFile(fixture)
			}
		}
		t.Fatalf("fakeRunner: no fixture for %s %v", name, args)
		return nil, nil
	}
}

func TestFetchPullRequests_GitHub(t *testing.T) {
	runner := fakeRunner(t, map[string]string{
		"repos/davaba86/mc-platform-cloud/pulls?state=open": "testdata/github_pulls.json",
	})

	prs, err := FetchGitHubPullRequests(runner, "davaba86", "mc-platform-cloud")
	if err != nil {
		t.Fatalf("FetchGitHubPullRequests() error = %v", err)
	}
	if len(prs) != 1 {
		t.Fatalf("len(prs) = %d, want 1", len(prs))
	}
	pr := prs[0]
	if pr.Source != "github" || pr.Repo != "davaba86/mc-platform-cloud" || pr.Number != 482 ||
		pr.Title != "fix-auth token refresh" || pr.Author != "davaba86" ||
		len(pr.Reviewers) != 1 || pr.Reviewers[0] != "jsmith" {
		t.Errorf("FetchGitHubPullRequests() = %+v, mismatch", pr)
	}
	wantUpdated := time.Date(2026, 10, 18, 7, 50, 0, 0, time.UTC)
	if !pr.UpdatedAt.Equal(wantUpdated) {
		t.Errorf("UpdatedAt = %v, want %v", pr.UpdatedAt, wantUpdated)
	}
}

func TestFetchPipelineStatus_GitHub(t *testing.T) {
	runner := fakeRunner(t, map[string]string{
		"repos/davaba86/mc-platform-cloud":                                      "testdata/github_repo.json",
		"repos/davaba86/mc-platform-cloud/actions/runs?branch=main&per_page=1": "testdata/github_runs.json",
	})

	run, err := FetchGitHubPipelineStatus(runner, "davaba86", "mc-platform-cloud")
	if err != nil {
		t.Fatalf("FetchGitHubPipelineStatus() error = %v", err)
	}
	if run.Repo != "davaba86/mc-platform-cloud" || run.Status != model.PipelinePass || run.Branch != "main" {
		t.Errorf("FetchGitHubPipelineStatus() = %+v, mismatch", run)
	}
	if run.Duration != 3*time.Minute+12*time.Second {
		t.Errorf("Duration = %v, want 3m12s", run.Duration)
	}
}

func TestFetchPipelineStatus_GitHub_NoRuns(t *testing.T) {
	runner := fakeRunner(t, map[string]string{
		"repos/davaba86/mc-platform-cloud":                                      "testdata/github_repo.json",
		"repos/davaba86/mc-platform-cloud/actions/runs?branch=main&per_page=1": "testdata/github_runs_empty.json",
	})

	run, err := FetchGitHubPipelineStatus(runner, "davaba86", "mc-platform-cloud")
	if err != nil {
		t.Fatalf("FetchGitHubPipelineStatus() error = %v", err)
	}
	if run.Status != model.PipelineNone {
		t.Errorf("Status = %v, want %v", run.Status, model.PipelineNone)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/githost/... -v`
Expected: FAIL — `FetchGitHubPullRequests` etc. undefined

- [ ] **Step 4: Write the implementation**

```go
package githost

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

type githubUser struct {
	Login string `json:"login"`
}

type githubPull struct {
	Number              int          `json:"number"`
	Title               string       `json:"title"`
	HTMLURL             string       `json:"html_url"`
	UpdatedAt           time.Time    `json:"updated_at"`
	User                githubUser   `json:"user"`
	RequestedReviewers  []githubUser `json:"requested_reviewers"`
}

func FetchGitHubPullRequests(run Runner, owner, name string) ([]model.PullRequest, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls?state=open", owner, name)
	out, err := run("gh", "api", path)
	if err != nil {
		return nil, fmt.Errorf("gh api %s: %w", path, err)
	}

	var pulls []githubPull
	if err := json.Unmarshal(out, &pulls); err != nil {
		return nil, fmt.Errorf("parse %s response: %w", path, err)
	}

	prs := make([]model.PullRequest, 0, len(pulls))
	for _, p := range pulls {
		reviewers := make([]string, 0, len(p.RequestedReviewers))
		for _, r := range p.RequestedReviewers {
			reviewers = append(reviewers, r.Login)
		}
		prs = append(prs, model.PullRequest{
			Source:    "github",
			Repo:      fmt.Sprintf("%s/%s", owner, name),
			Number:    p.Number,
			Title:     p.Title,
			URL:       p.HTMLURL,
			Author:    p.User.Login,
			Reviewers: reviewers,
			UpdatedAt: p.UpdatedAt,
		})
	}
	return prs, nil
}

type githubRepo struct {
	DefaultBranch string `json:"default_branch"`
}

type githubRun struct {
	Status       string    `json:"status"`
	Conclusion   string    `json:"conclusion"`
	HeadBranch   string    `json:"head_branch"`
	RunStartedAt time.Time `json:"run_started_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type githubRunsResponse struct {
	WorkflowRuns []githubRun `json:"workflow_runs"`
}

func FetchGitHubPipelineStatus(run Runner, owner, name string) (model.PipelineRun, error) {
	repoPath := fmt.Sprintf("repos/%s/%s", owner, name)
	repoOut, err := run("gh", "api", repoPath)
	if err != nil {
		return model.PipelineRun{}, fmt.Errorf("gh api %s: %w", repoPath, err)
	}
	var repo githubRepo
	if err := json.Unmarshal(repoOut, &repo); err != nil {
		return model.PipelineRun{}, fmt.Errorf("parse %s response: %w", repoPath, err)
	}

	runsPath := fmt.Sprintf("repos/%s/%s/actions/runs?branch=%s&per_page=1", owner, name, repo.DefaultBranch)
	runsOut, err := run("gh", "api", runsPath)
	if err != nil {
		return model.PipelineRun{}, fmt.Errorf("gh api %s: %w", runsPath, err)
	}
	var resp githubRunsResponse
	if err := json.Unmarshal(runsOut, &resp); err != nil {
		return model.PipelineRun{}, fmt.Errorf("parse %s response: %w", runsPath, err)
	}

	repoFullName := fmt.Sprintf("%s/%s", owner, name)
	if len(resp.WorkflowRuns) == 0 {
		return model.PipelineRun{Repo: repoFullName, Status: model.PipelineNone, Branch: repo.DefaultBranch}, nil
	}

	r := resp.WorkflowRuns[0]
	status := model.PipelineRunning
	if r.Status == "completed" {
		if r.Conclusion == "success" {
			status = model.PipelinePass
		} else {
			status = model.PipelineFail
		}
	}

	return model.PipelineRun{
		Repo:      repoFullName,
		Status:    status,
		Branch:    repo.DefaultBranch,
		Duration:  r.UpdatedAt.Sub(r.RunStartedAt),
		UpdatedAt: r.UpdatedAt,
	}, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/githost/... -v`
Expected: PASS (all three tests)

- [ ] **Step 6: Commit**

```bash
git add internal/githost/github.go internal/githost/github_test.go internal/githost/testdata/github_*.json
git commit -m "feat: add GitHub PR and pipeline status fetching"
```

---

### Task 8: GitLab fetch (MRs + pipeline status)

**Files:**
- Create: `internal/githost/gitlab.go`
- Test: `internal/githost/gitlab_test.go`
- Create fixtures: `internal/githost/testdata/gitlab_mrs.json`, `internal/githost/testdata/gitlab_project.json`, `internal/githost/testdata/gitlab_pipelines.json`, `internal/githost/testdata/gitlab_pipeline_detail.json`, `internal/githost/testdata/gitlab_pipelines_empty.json`

- [ ] **Step 1: Add fixture files**

`internal/githost/testdata/gitlab_mrs.json`:

```json
[
  {
    "iid": 91,
    "title": "add SSO login flow",
    "web_url": "https://gitlab.com/mecha-consulting/mc-product-lifttraq/-/merge_requests/91",
    "updated_at": "2026-10-18T07:57:00Z",
    "author": { "username": "david.abarca" },
    "reviewers": [{ "username": "agupta" }]
  }
]
```

`internal/githost/testdata/gitlab_project.json`:

```json
{ "default_branch": "main" }
```

`internal/githost/testdata/gitlab_pipelines.json`:

```json
[{ "id": 5501, "status": "success", "ref": "main" }]
```

`internal/githost/testdata/gitlab_pipeline_detail.json`:

```json
{
  "id": 5501,
  "status": "success",
  "ref": "main",
  "duration": 108,
  "updated_at": "2026-10-18T07:34:00Z"
}
```

`internal/githost/testdata/gitlab_pipelines_empty.json`:

```json
[]
```

- [ ] **Step 2: Write the failing test**

```go
package githost

import (
	"testing"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

func TestFetchMergeRequests_GitLab(t *testing.T) {
	runner := fakeRunner(t, map[string]string{
		"projects/mecha-consulting%2Fmc-product-lifttraq/merge_requests?state=opened": "testdata/gitlab_mrs.json",
	})

	mrs, err := FetchGitLabMergeRequests(runner, "mecha-consulting/mc-product-lifttraq")
	if err != nil {
		t.Fatalf("FetchGitLabMergeRequests() error = %v", err)
	}
	if len(mrs) != 1 {
		t.Fatalf("len(mrs) = %d, want 1", len(mrs))
	}
	mr := mrs[0]
	if mr.Source != "gitlab" || mr.Repo != "mecha-consulting/mc-product-lifttraq" || mr.Number != 91 ||
		mr.Title != "add SSO login flow" || mr.Author != "david.abarca" ||
		len(mr.Reviewers) != 1 || mr.Reviewers[0] != "agupta" {
		t.Errorf("FetchGitLabMergeRequests() = %+v, mismatch", mr)
	}
}

func TestFetchPipelineStatus_GitLab(t *testing.T) {
	runner := fakeRunner(t, map[string]string{
		"projects/mecha-consulting%2Fmc-product-lifttraq":                              "testdata/gitlab_project.json",
		"projects/mecha-consulting%2Fmc-product-lifttraq/pipelines?ref=main&per_page=1": "testdata/gitlab_pipelines.json",
		"projects/mecha-consulting%2Fmc-product-lifttraq/pipelines/5501":                "testdata/gitlab_pipeline_detail.json",
	})

	run, err := FetchGitLabPipelineStatus(runner, "mecha-consulting/mc-product-lifttraq")
	if err != nil {
		t.Fatalf("FetchGitLabPipelineStatus() error = %v", err)
	}
	if run.Repo != "mecha-consulting/mc-product-lifttraq" || run.Status != model.PipelinePass || run.Branch != "main" {
		t.Errorf("FetchGitLabPipelineStatus() = %+v, mismatch", run)
	}
	if run.Duration != 108*time.Second {
		t.Errorf("Duration = %v, want 108s", run.Duration)
	}
}

func TestFetchPipelineStatus_GitLab_NoRuns(t *testing.T) {
	runner := fakeRunner(t, map[string]string{
		"projects/mecha-consulting%2Fmc-product-lifttraq":                              "testdata/gitlab_project.json",
		"projects/mecha-consulting%2Fmc-product-lifttraq/pipelines?ref=main&per_page=1": "testdata/gitlab_pipelines_empty.json",
	})

	run, err := FetchGitLabPipelineStatus(runner, "mecha-consulting/mc-product-lifttraq")
	if err != nil {
		t.Fatalf("FetchGitLabPipelineStatus() error = %v", err)
	}
	if run.Status != model.PipelineNone {
		t.Errorf("Status = %v, want %v", run.Status, model.PipelineNone)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/githost/... -v`
Expected: FAIL — `FetchGitLabMergeRequests` etc. undefined

- [ ] **Step 4: Write the implementation**

```go
package githost

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

func encodeProjectPath(path string) string {
	return url.QueryEscape(path)
}

type gitlabUser struct {
	Username string `json:"username"`
}

type gitlabMergeRequest struct {
	IID       int        `json:"iid"`
	Title     string     `json:"title"`
	WebURL    string     `json:"web_url"`
	UpdatedAt time.Time  `json:"updated_at"`
	Author    gitlabUser `json:"author"`
	Reviewers []gitlabUser `json:"reviewers"`
}

func FetchGitLabMergeRequests(run Runner, projectPath string) ([]model.PullRequest, error) {
	path := fmt.Sprintf("projects/%s/merge_requests?state=opened", encodeProjectPath(projectPath))
	out, err := run("glab", "api", path)
	if err != nil {
		return nil, fmt.Errorf("glab api %s: %w", path, err)
	}

	var mrs []gitlabMergeRequest
	if err := json.Unmarshal(out, &mrs); err != nil {
		return nil, fmt.Errorf("parse %s response: %w", path, err)
	}

	prs := make([]model.PullRequest, 0, len(mrs))
	for _, m := range mrs {
		reviewers := make([]string, 0, len(m.Reviewers))
		for _, r := range m.Reviewers {
			reviewers = append(reviewers, r.Username)
		}
		prs = append(prs, model.PullRequest{
			Source:    "gitlab",
			Repo:      projectPath,
			Number:    m.IID,
			Title:     m.Title,
			URL:       m.WebURL,
			Author:    m.Author.Username,
			Reviewers: reviewers,
			UpdatedAt: m.UpdatedAt,
		})
	}
	return prs, nil
}

type gitlabProject struct {
	DefaultBranch string `json:"default_branch"`
}

type gitlabPipelineSummary struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Ref    string `json:"ref"`
}

type gitlabPipelineDetail struct {
	ID        int       `json:"id"`
	Status    string    `json:"status"`
	Ref       string    `json:"ref"`
	Duration  int       `json:"duration"`
	UpdatedAt time.Time `json:"updated_at"`
}

// gitlabPipelineStatus maps GitLab pipeline statuses onto the four
// buckets daily-pulse renders. Only "success" is a pass and only
// "running"/"pending" count as in-progress; every other terminal status
// (failed, canceled, skipped) renders as "fail" so a non-green pipeline
// is never silently hidden.
func gitlabPipelineStatus(status string) model.PipelineStatus {
	switch status {
	case "success":
		return model.PipelinePass
	case "running", "pending":
		return model.PipelineRunning
	default:
		return model.PipelineFail
	}
}

func FetchGitLabPipelineStatus(run Runner, projectPath string) (model.PipelineRun, error) {
	encoded := encodeProjectPath(projectPath)

	projectOut, err := run("glab", "api", fmt.Sprintf("projects/%s", encoded))
	if err != nil {
		return model.PipelineRun{}, fmt.Errorf("glab api projects/%s: %w", encoded, err)
	}
	var project gitlabProject
	if err := json.Unmarshal(projectOut, &project); err != nil {
		return model.PipelineRun{}, fmt.Errorf("parse project response: %w", err)
	}

	listPath := fmt.Sprintf("projects/%s/pipelines?ref=%s&per_page=1", encoded, project.DefaultBranch)
	listOut, err := run("glab", "api", listPath)
	if err != nil {
		return model.PipelineRun{}, fmt.Errorf("glab api %s: %w", listPath, err)
	}
	var summaries []gitlabPipelineSummary
	if err := json.Unmarshal(listOut, &summaries); err != nil {
		return model.PipelineRun{}, fmt.Errorf("parse %s response: %w", listPath, err)
	}

	if len(summaries) == 0 {
		return model.PipelineRun{Repo: projectPath, Status: model.PipelineNone, Branch: project.DefaultBranch}, nil
	}

	detailPath := fmt.Sprintf("projects/%s/pipelines/%d", encoded, summaries[0].ID)
	detailOut, err := run("glab", "api", detailPath)
	if err != nil {
		return model.PipelineRun{}, fmt.Errorf("glab api %s: %w", detailPath, err)
	}
	var detail gitlabPipelineDetail
	if err := json.Unmarshal(detailOut, &detail); err != nil {
		return model.PipelineRun{}, fmt.Errorf("parse %s response: %w", detailPath, err)
	}

	return model.PipelineRun{
		Repo:      projectPath,
		Status:    gitlabPipelineStatus(detail.Status),
		Branch:    detail.Ref,
		Duration:  time.Duration(detail.Duration) * time.Second,
		UpdatedAt: detail.UpdatedAt,
	}, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/githost/... -v`
Expected: PASS (all tests in the package)

- [ ] **Step 6: Commit**

```bash
git add internal/githost/gitlab.go internal/githost/gitlab_test.go internal/githost/testdata/gitlab_*.json
git commit -m "feat: add GitLab MR and pipeline status fetching"
```

---

### Task 9: SQLite snapshot storage

**Files:**
- Create: `internal/store/store.go`
- Test: `internal/store/store_test.go`

- [ ] **Step 1: Write the failing test**

```go
package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

func TestSaveSnapshot_AndReadBack(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pulse.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	takenAt := time.Date(2026, 10, 18, 8, 2, 0, 0, time.UTC)
	prs := []model.PullRequest{
		{Source: "github", Repo: "davaba86/mc-platform-cloud", Number: 482, Title: "fix-auth token refresh", UpdatedAt: takenAt},
	}
	pipelines := []model.PipelineRun{
		{Repo: "davaba86/mc-platform-cloud", Status: model.PipelinePass, Branch: "main", Duration: 192 * time.Second, UpdatedAt: takenAt},
	}

	snapshotID, err := SaveSnapshot(db, prs, pipelines, takenAt)
	if err != nil {
		t.Fatalf("SaveSnapshot() error = %v", err)
	}
	if snapshotID == 0 {
		t.Fatal("SaveSnapshot() returned snapshot id 0, want nonzero")
	}

	var prCount, pipelineCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM pull_requests WHERE snapshot_id = ?", snapshotID).Scan(&prCount); err != nil {
		t.Fatalf("count pull_requests: %v", err)
	}
	if prCount != 1 {
		t.Errorf("pull_requests count = %d, want 1", prCount)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM pipeline_runs WHERE snapshot_id = ?", snapshotID).Scan(&pipelineCount); err != nil {
		t.Fatalf("count pipeline_runs: %v", err)
	}
	if pipelineCount != 1 {
		t.Errorf("pipeline_runs count = %d, want 1", pipelineCount)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store/... -v`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 3: Write the implementation**

```go
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/davaba86/daily-pulse/internal/model"
)

const schema = `
CREATE TABLE IF NOT EXISTS snapshots (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	taken_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pull_requests (
	snapshot_id INTEGER NOT NULL REFERENCES snapshots(id),
	source      TEXT NOT NULL,
	repo        TEXT NOT NULL,
	number      INTEGER NOT NULL,
	title       TEXT NOT NULL,
	author      TEXT NOT NULL,
	updated_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pipeline_runs (
	snapshot_id  INTEGER NOT NULL REFERENCES snapshots(id),
	repo         TEXT NOT NULL,
	status       TEXT NOT NULL,
	branch       TEXT NOT NULL,
	duration_sec INTEGER NOT NULL,
	updated_at   TEXT NOT NULL
);
`

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}
	return db, nil
}

func SaveSnapshot(db *sql.DB, prs []model.PullRequest, pipelines []model.PipelineRun, takenAt time.Time) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec("INSERT INTO snapshots (taken_at) VALUES (?)", takenAt.Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("insert snapshot: %w", err)
	}
	snapshotID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read snapshot id: %w", err)
	}

	for _, pr := range prs {
		_, err := tx.Exec(
			"INSERT INTO pull_requests (snapshot_id, source, repo, number, title, author, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			snapshotID, pr.Source, pr.Repo, pr.Number, pr.Title, pr.Author, pr.UpdatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return 0, fmt.Errorf("insert pull_request %s#%d: %w", pr.Repo, pr.Number, err)
		}
	}

	for _, p := range pipelines {
		_, err := tx.Exec(
			"INSERT INTO pipeline_runs (snapshot_id, repo, status, branch, duration_sec, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			snapshotID, p.Repo, string(p.Status), p.Branch, int(p.Duration.Seconds()), p.UpdatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return 0, fmt.Errorf("insert pipeline_run %s: %w", p.Repo, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}
	return snapshotID, nil
}
```

- [ ] **Step 4: Add the sqlite dependency**

Run: `go get modernc.org/sqlite`
Expected: `go.mod`/`go.sum` updated with a `modernc.org/sqlite` require entry

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/store/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/store/ go.mod go.sum
git commit -m "feat: add SQLite snapshot storage"
```

---

### Task 10: CLI wiring with mock data

**Files:**
- Create: `cmd/root.go`
- Test: `cmd/root_test.go`

This wires the command end-to-end against mock data first, per the
mock-first build order in the spec — no live `gh`/`glab`/SQLite calls yet.

- [ ] **Step 1: Write the failing test**

```go
package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand_Mock(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--mock"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"GITHUB", "GITLAB", "PIPELINES", "open PRs/MRs"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q, got:\n%s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -v`
Expected: FAIL — package/`NewRootCommand` don't exist yet

- [ ] **Step 3: Write the implementation**

```go
package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/davaba86/daily-pulse/internal/mockdata"
	"github.com/davaba86/daily-pulse/internal/render"
)

func NewRootCommand() *cobra.Command {
	var useMock bool
	var showAll bool

	cmd := &cobra.Command{
		Use:   "daily-pulse",
		Short: "Daily overview of your open PRs, MRs, and pipeline status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if useMock {
				prs := mockdata.PullRequests()
				pipelines := mockdata.PipelineRuns()
				_ = showAll // author filtering lands in Task 11 once real fetches exist
				cmd.Print(render.FormatTable(prs, pipelines, time.Now()))
				return nil
			}
			return cmd.Help()
		},
	}

	cmd.Flags().BoolVar(&useMock, "mock", false, "use mock data instead of live GitHub/GitLab calls")
	cmd.Flags().BoolVar(&showAll, "all", false, "show PRs/MRs from everyone, not just you")

	return cmd
}

func Execute() {
	cmd := NewRootCommand()
	cobra.CheckErr(cmd.Execute())
}
```

- [ ] **Step 4: Add the cobra dependency**

Run: `go get github.com/spf13/cobra`
Expected: `go.mod`/`go.sum` updated with a `github.com/spf13/cobra` require entry

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./cmd/... -v`
Expected: PASS

- [ ] **Step 6: Build and manually check the mockup matches**

Run: `go build -o daily-pulse . && ./daily-pulse --mock`
Expected: output matching the approved mockup layout (GITHUB/GITLAB/PIPELINES
sections, aligned columns, summary line)

- [ ] **Step 7: Commit**

```bash
git add cmd/ go.mod go.sum
git commit -m "feat: wire CLI to render mock data via --mock flag"
```

---

### Task 11: Wire real GitHub/GitLab fetches + snapshot persistence

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/root_test.go`

This replaces the `--mock` default path with real `internal/githost` calls
driven by `internal/config`, and persists each run's results via
`internal/store`. `--mock` is kept as an explicit opt-in for development/
demoing without live auth or local state.

- [ ] **Step 1: Write the failing test**

Append to `cmd/root_test.go`:

```go
func TestRootCommand_MissingConfig(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", "/nonexistent/config.yaml"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want error for missing config")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -v`
Expected: FAIL — `--config` flag doesn't exist yet

- [ ] **Step 3: Write the implementation**

Replace the full contents of `cmd/root.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/davaba86/daily-pulse/internal/config"
	"github.com/davaba86/daily-pulse/internal/githost"
	"github.com/davaba86/daily-pulse/internal/mockdata"
	"github.com/davaba86/daily-pulse/internal/model"
	"github.com/davaba86/daily-pulse/internal/render"
	"github.com/davaba86/daily-pulse/internal/store"
)

func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "daily-pulse", "config.yaml")
}

func defaultStorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "daily-pulse", "pulse.db")
}

func fetchAll(cfg *config.Config, showAll bool) ([]model.PullRequest, []model.PipelineRun, error) {
	var prs []model.PullRequest
	var pipelines []model.PipelineRun

	for _, repo := range cfg.GitHub.Repos {
		repoPRs, err := githost.FetchGitHubPullRequests(githost.ExecRunner, repo.Owner, repo.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch github PRs for %s/%s: %w", repo.Owner, repo.Name, err)
		}
		for _, pr := range repoPRs {
			if showAll || pr.Author == cfg.GitHub.Username {
				prs = append(prs, pr)
			}
		}

		run, err := githost.FetchGitHubPipelineStatus(githost.ExecRunner, repo.Owner, repo.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch github pipeline for %s/%s: %w", repo.Owner, repo.Name, err)
		}
		pipelines = append(pipelines, run)
	}

	for _, project := range cfg.GitLab.Projects {
		projectMRs, err := githost.FetchGitLabMergeRequests(githost.ExecRunner, project.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch gitlab MRs for %s: %w", project.Path, err)
		}
		for _, mr := range projectMRs {
			if showAll || mr.Author == cfg.GitLab.Username {
				prs = append(prs, mr)
			}
		}

		run, err := githost.FetchGitLabPipelineStatus(githost.ExecRunner, project.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch gitlab pipeline for %s: %w", project.Path, err)
		}
		pipelines = append(pipelines, run)
	}

	return prs, pipelines, nil
}

func NewRootCommand() *cobra.Command {
	var useMock bool
	var showAll bool
	var configPath string

	cmd := &cobra.Command{
		Use:   "daily-pulse",
		Short: "Daily overview of your open PRs, MRs, and pipeline status",
		RunE: func(cmd *cobra.Command, args []string) error {
			now := time.Now()

			if useMock {
				cmd.Print(render.FormatTable(mockdata.PullRequests(), mockdata.PipelineRuns(), now))
				return nil
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			prs, pipelines, err := fetchAll(cfg, showAll)
			if err != nil {
				return err
			}

			dbPath := defaultStorePath()
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return fmt.Errorf("create store dir: %w", err)
			}
			db, err := store.Open(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()
			if _, err := store.SaveSnapshot(db, prs, pipelines, now); err != nil {
				return err
			}

			cmd.Print(render.FormatTable(prs, pipelines, now))
			return nil
		},
	}

	cmd.Flags().BoolVar(&useMock, "mock", false, "use mock data instead of live GitHub/GitLab calls")
	cmd.Flags().BoolVar(&showAll, "all", false, "show PRs/MRs from everyone, not just you")
	cmd.Flags().StringVar(&configPath, "config", defaultConfigPath(), "path to config.yaml")

	return cmd
}

func Execute() {
	cmd := NewRootCommand()
	cobra.CheckErr(cmd.Execute())
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./... -v`
Expected: PASS across all packages

- [ ] **Step 5: Commit**

```bash
git add cmd/
git commit -m "feat: wire CLI to real GitHub/GitLab fetches and snapshot storage"
```

---

### Task 12: README and manual smoke test

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Write full usage docs**

Replace the full contents of `README.md`:

```markdown
# daily-pulse

A CLI daily overview of your open GitHub PRs, GitLab MRs, and repo pipeline
status — one command instead of tab-hopping between web UIs.

## Install

```bash
go build -o daily-pulse .
```

## Prerequisites

- [`gh`](https://cli.github.com/) authenticated: `gh auth login`
- [`glab`](https://gitlab.com/gitlab-org/cli) authenticated: `glab auth login`

daily-pulse shells out to both CLIs and reuses whatever auth they already
have — it never stores a GitHub/GitLab token itself.

## Configure

Create `~/.config/daily-pulse/config.yaml`:

```yaml
github:
  username: davaba86
  repos:
    - owner: davaba86
      name: daily-pulse
gitlab:
  username: david.abarca
  projects:
    - path: mecha-consulting/mc-platform-cloud
```

## Usage

```bash
daily-pulse            # your open PRs/MRs + pipeline status for configured repos
daily-pulse --all      # everyone's open PRs/MRs, not just yours
daily-pulse --mock      # render against built-in sample data, no live calls
```

Each run also writes a snapshot to `~/.local/share/daily-pulse/pulse.db`,
used by future benchmarking features.
```

- [ ] **Step 2: Manual smoke test against mock data**

Run: `go build -o daily-pulse . && ./daily-pulse --mock`
Expected: full table output with GITHUB/GITLAB/PIPELINES sections

- [ ] **Step 3: Manual smoke test against live data (requires `gh`/`glab` auth and a real config.yaml)**

Run: `./daily-pulse`
Expected: live PR/MR + pipeline data for whatever repos are in your
`~/.config/daily-pulse/config.yaml`, and `~/.local/share/daily-pulse/pulse.db`
created

- [ ] **Step 4: Full test suite + vet + format check**

Run: `go vet ./... && gofmt -l . && go test ./...`
Expected: `gofmt -l .` prints nothing (all files formatted); `go test ./...`
reports `ok` for every package

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "docs: add usage and configuration instructions"
```

---

## Plan self-review notes

- **Spec coverage:** Phase 1's PR/MR dashboard, pipeline view, mock-first
  build order, author-vs-everyone filtering, YAML config, `gh`/`glab`-CLI
  auth reuse, SQLite snapshot writing, and the approved output layout are
  all covered (Tasks 1–12). Jira (Phase 2), `stats` (Phase 3), and inline
  `● NEW` alerts (Phase 4) are intentionally out of scope for this plan —
  each gets its own plan once Phase 1 ships, per the spec's phasing.
- **Type consistency:** `model.PullRequest`/`model.PipelineRun` fields are
  defined once in Task 2 and used with identical names throughout Tasks
  4–11 (`Source`, `Repo`, `Number`, `Title`, `Author`, `Reviewers`,
  `UpdatedAt`, `Status`, `Branch`, `Duration`). `Runner` (Task 6) is used
  with the same signature in Tasks 7, 8, and 11.

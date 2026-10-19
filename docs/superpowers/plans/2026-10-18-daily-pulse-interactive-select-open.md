# daily-pulse Interactive Session, Round 2 (Row Selection + Open in Browser) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** In `daily-pulse --interactive`, turn the OPEN PRs/MRs section into a real selectable table (visible cursor row) and let `Enter` open the selected row's URL in the browser. PIPELINES stays static text beneath it, unchanged.

**Architecture:** Extract the render package's per-column natural-width and per-row formatting logic (already written for the static table) into exported functions so the interactive path reuses them exactly instead of duplicating formatting logic. Replace `internal/tui`'s viewport-over-a-string model with a `bubbles/table.Model` built from that shared logic, plus an injectable `opener` function (same DI pattern as `internal/githost.Runner`) so `Enter`'s behavior is unit-testable without launching a real browser.

**Tech Stack:** `github.com/charmbracelet/bubbles/table` (new), existing `bubbletea`/`bubbles/viewport` dependencies stay (viewport is dropped from `tui.go` itself but the module stays available transitively via `bubbles`).

Spec: `docs/superpowers/specs/2026-10-18-daily-pulse-interactive-select-open-design.md`

---

## File Structure

```
internal/render/render.go       modify: extract prNaturalWidths; export PRColumnWidths, PRRow, SortPullRequests, PipelinesText
internal/render/render_test.go  modify: add tests for the newly exported functions
internal/tui/browser.go         new: browserCommand (pure, testable) + openInBrowser (real opener)
internal/tui/browser_test.go    new: table-driven test over GOOS values
internal/tui/tui.go             modify: replace viewport-based Model with table.Model + opener + status
internal/tui/tui_test.go        modify: replace Round 1's tests with Round 2's (selection, enter, status)
cmd/root.go                     modify: terminalHeight() helper, new tui.Run signature
```

---

### Task 1: Export shared PR formatting/sorting from render

**Files:**
- Modify: `internal/render/render.go`
- Modify: `internal/render/render_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/render/render_test.go`:

```go
func TestSortPullRequests(t *testing.T) {
	now := fixedNow()
	prs := []model.PullRequest{
		{Title: "older", UpdatedAt: now.Add(-time.Hour)},
		{Title: "newer", UpdatedAt: now},
	}

	sorted := SortPullRequests(prs)

	if len(sorted) != 2 || sorted[0].Title != "newer" || sorted[1].Title != "older" {
		t.Errorf("SortPullRequests() = %+v, want newer first", sorted)
	}
	if prs[0].Title != "older" {
		t.Error("SortPullRequests() should not mutate its input")
	}
}

func TestPRRow(t *testing.T) {
	now := fixedNow()
	pr := model.PullRequest{
		Source: "github", Repo: "davaba86/x", Number: 7, Title: "fix thing",
		Reviewers: []string{"jsmith"}, UpdatedAt: now.Add(-5 * time.Minute),
	}

	row := PRRow(pr, now)

	want := []string{"GH", "davaba86/x", "7", "fix thing", "jsmith", "5m ago"}
	if len(row) != len(want) {
		t.Fatalf("PRRow() = %v, want %d cells", row, len(want))
	}
	for i := range want {
		if row[i] != want[i] {
			t.Errorf("PRRow()[%d] = %q, want %q", i, row[i], want[i])
		}
	}
}

func TestPRColumnWidths_Stretches(t *testing.T) {
	now := fixedNow()
	prs := []model.PullRequest{{Source: "github", Repo: "x", Number: 1, Title: "short", UpdatedAt: now}}

	_, _, _, titleNatural, _, _ := PRColumnWidths(prs, now, 0)
	_, _, _, titleStretched, _, _ := PRColumnWidths(prs, now, 200)

	if titleStretched <= titleNatural {
		t.Errorf("PRColumnWidths() TITLE width with width=200 (%d) should exceed natural (%d)", titleStretched, titleNatural)
	}
}

func TestPipelinesText(t *testing.T) {
	now := fixedNow()
	pipelines := []model.PipelineRun{
		{Repo: "davaba86/x", Status: model.PipelinePass, Branch: "main", Duration: 5 * time.Second, UpdatedAt: now},
	}

	out := PipelinesText(pipelines, now)

	for _, want := range []string{"PIPELINES", "davaba86/x", "pass", "main"} {
		if !strings.Contains(out, want) {
			t.Errorf("PipelinesText() missing %q, got:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/render/... -v`
Expected: FAIL — `SortPullRequests`/`PRRow`/`PRColumnWidths`/`PipelinesText` undefined

- [ ] **Step 3: Write the implementation**

Replace `titleStretch` and `writePRSection` in `internal/render/render.go`, and add the new exported functions, as follows:

```go
func prNaturalWidths(prs []model.PullRequest, now time.Time) (source, repo, number, title, reviewers, updated int) {
	source = utf8.RuneCountInString("SOURCE")
	repo = utf8.RuneCountInString("REPO")
	number = utf8.RuneCountInString("#/!")
	title = utf8.RuneCountInString("TITLE")
	reviewers = utf8.RuneCountInString("REVIEWERS")
	updated = utf8.RuneCountInString("UPDATED")

	for _, pr := range prs {
		source = max(source, utf8.RuneCountInString(sourceLabel(pr.Source)))
		repo = max(repo, utf8.RuneCountInString(pr.Repo))
		number = max(number, utf8.RuneCountInString(strconv.Itoa(pr.Number)))
		title = max(title, utf8.RuneCountInString(pr.Title))
		reviewers = max(reviewers, utf8.RuneCountInString(reviewersCell(pr.Reviewers)))
		updated = max(updated, utf8.RuneCountInString(relativeTime(pr.UpdatedAt, now)))
	}
	return
}

// titleStretch returns how many extra spaces the TITLE header needs so the
// table's total width reaches the terminal width, or 0 if the table is
// already at least that wide naturally.
func titleStretch(prs []model.PullRequest, now time.Time, width int) int {
	source, repo, number, title, reviewers, updated := prNaturalWidths(prs, now)
	natural := source + repo + number + title + reviewers + updated + 5*columnGap
	if width <= natural {
		return 0
	}
	return width - natural
}

// PRColumnWidths returns per-column widths for the OPEN PRs/MRs table, in
// SOURCE/REPO/#!/TITLE/REVIEWERS/UPDATED order, with TITLE stretched to
// fill width when width exceeds the natural total — the same rule the
// static table uses. Used by the interactive session to size its
// table.Column definitions.
func PRColumnWidths(prs []model.PullRequest, now time.Time, width int) (source, repo, number, title, reviewers, updated int) {
	source, repo, number, title, reviewers, updated = prNaturalWidths(prs, now)
	title += titleStretch(prs, now, width)
	return
}

// SortPullRequests returns a copy of prs sorted most-recently-updated
// first — the same order the static OPEN PRs/MRs table renders in.
func SortPullRequests(prs []model.PullRequest) []model.PullRequest {
	sorted := make([]model.PullRequest, len(prs))
	copy(sorted, prs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].UpdatedAt.After(sorted[j].UpdatedAt) })
	return sorted
}

// PRRow returns the display cells for one PR/MR row, in the same column
// order as the static table: SOURCE, REPO, #/!, TITLE, REVIEWERS, UPDATED.
func PRRow(pr model.PullRequest, now time.Time) []string {
	return []string{
		sourceLabel(pr.Source), pr.Repo, strconv.Itoa(pr.Number), pr.Title,
		reviewersCell(pr.Reviewers), relativeTime(pr.UpdatedAt, now),
	}
}

func writePRSection(buf *bytes.Buffer, prs []model.PullRequest, now time.Time, width int) {
	if len(prs) == 0 {
		return
	}
	sorted := SortPullRequests(prs)
	titleHeader := "TITLE" + strings.Repeat(" ", titleStretch(sorted, now, width))

	buf.WriteString("OPEN PRs/MRs\n")
	tw := tabwriter.NewWriter(buf, 0, 0, columnGap, ' ', 0)
	fmt.Fprintf(tw, "SOURCE\tREPO\t#/!\t%s\tREVIEWERS\tUPDATED\n", titleHeader)
	for _, pr := range sorted {
		row := PRRow(pr, now)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", row[0], row[1], row[2], row[3], row[4], row[5])
	}
	tw.Flush()
	buf.WriteString("\n")
}

// PipelinesText renders just the PIPELINES section, for the interactive
// session to display alongside its selectable PR/MR table.
func PipelinesText(pipelines []model.PipelineRun, now time.Time) string {
	var buf bytes.Buffer
	writePipelineSection(&buf, pipelines, now)
	return buf.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/render/... -v`
Expected: PASS (all tests, including the pre-existing ones — this step is a pure refactor plus additions, so nothing earlier should change behavior)

- [ ] **Step 5: Commit**

```bash
git add internal/render/
git commit -m "feat: export PR row/sort/width helpers and PipelinesText for reuse"
```

---

### Task 2: Browser opener

**Files:**
- Create: `internal/tui/browser.go`
- Test: `internal/tui/browser_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/... -run TestBrowserCommand -v`
Expected: FAIL — `browserCommand` undefined

- [ ] **Step 3: Write the implementation**

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/... -run TestBrowserCommand -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/browser.go internal/tui/browser_test.go
git commit -m "feat: add cross-platform browser opener for interactive session"
```

---

### Task 3: Selectable table + status line in the TUI model

**Files:**
- Modify: `internal/tui/tui.go`
- Modify: `internal/tui/tui_test.go` (full replacement — Round 1's viewport-based tests no longer apply)

- [ ] **Step 1: Replace the failing/outdated tests**

Replace the full contents of `internal/tui/tui_test.go`:

```go
package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/davaba86/daily-pulse/internal/model"
)

func testPRs() []model.PullRequest {
	now := time.Now()
	return []model.PullRequest{
		{Source: "github", Repo: "davaba86/a", Number: 1, Title: "first", URL: "https://example.com/1", UpdatedAt: now},
		{Source: "gitlab", Repo: "mecha-consulting/b", Number: 2, Title: "second", URL: "https://example.com/2", UpdatedAt: now.Add(-time.Hour)},
	}
}

func TestUpdate_EscQuits(t *testing.T) {
	m := NewModel(testPRs(), time.Now(), "", 80, 24, func(string) error { return nil })

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := updated.(Model)

	if !model.quitting {
		t.Error("Update(Esc) should set quitting = true")
	}
	if cmd == nil {
		t.Error("Update(Esc) should return tea.Quit, got nil cmd")
	}
}

func TestUpdate_QQuits(t *testing.T) {
	m := NewModel(testPRs(), time.Now(), "", 80, 24, func(string) error { return nil })

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	model := updated.(Model)

	if !model.quitting {
		t.Error("Update('q') should set quitting = true")
	}
	if cmd == nil {
		t.Error("Update('q') should return tea.Quit, got nil cmd")
	}
}

func TestUpdate_EnterOpensSelectedURL(t *testing.T) {
	var gotURL string
	open := func(url string) error { gotURL = url; return nil }
	m := NewModel(testPRs(), time.Now(), "", 80, 24, open)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)

	want := model.prs[model.table.Cursor()].URL
	if gotURL == "" {
		t.Fatal("opener was not called")
	}
	if gotURL != want {
		t.Errorf("opener called with %q, want selected row's URL %q", gotURL, want)
	}
}

func TestUpdate_EnterShowsErrorOnOpenFailure(t *testing.T) {
	open := func(string) error { return errors.New("no browser found") }
	m := NewModel(testPRs(), time.Now(), "", 80, 24, open)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)

	if model.status == "" {
		t.Error("Update(Enter) with a failing opener should set a status message")
	}
	if model.quitting {
		t.Error("a failed browser open should not quit the session")
	}
}

func TestUpdate_EnterNoopWhenNoRows(t *testing.T) {
	called := false
	open := func(string) error { called = true; return nil }
	m := NewModel(nil, time.Now(), "", 80, 24, open)

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if called {
		t.Error("Update(Enter) with no rows should not call the opener")
	}
}

func TestView_EmptyWhenQuitting(t *testing.T) {
	m := NewModel(testPRs(), time.Now(), "", 80, 24, func(string) error { return nil })
	m.quitting = true

	if got := m.View(); got != "" {
		t.Errorf("View() while quitting = %q, want empty string", got)
	}
}

func TestView_IncludesPipelinesText(t *testing.T) {
	m := NewModel(testPRs(), time.Now(), "PIPELINES TEXT HERE", 80, 24, func(string) error { return nil })

	if !strings.Contains(m.View(), "PIPELINES TEXT HERE") {
		t.Error("View() should include the pipelines text")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/... -v`
Expected: FAIL — `NewModel`'s signature doesn't match, `m.table`/`m.prs`/`m.status` don't exist yet

- [ ] **Step 3: Write the implementation**

Replace the full contents of `internal/tui/tui.go`:

```go
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/davaba86/daily-pulse/internal/model"
	"github.com/davaba86/daily-pulse/internal/render"
)

const helpText = "↑/↓ navigate · enter open in browser · esc/q quit"

type Model struct {
	table         table.Model
	prs           []model.PullRequest // same order as table rows, for URL lookup on Enter
	pipelinesText string
	open          opener
	quitting      bool
	status        string
}

func NewModel(prs []model.PullRequest, now time.Time, pipelinesText string, width, height int, open opener) Model {
	sorted := render.SortPullRequests(prs)
	sourceW, repoW, numberW, titleW, reviewersW, updatedW := render.PRColumnWidths(sorted, now, width)

	columns := []table.Column{
		{Title: "SOURCE", Width: sourceW},
		{Title: "REPO", Width: repoW},
		{Title: "#/!", Width: numberW},
		{Title: "TITLE", Width: titleW},
		{Title: "REVIEWERS", Width: reviewersW},
		{Title: "UPDATED", Width: updatedW},
	}

	rows := make([]table.Row, len(sorted))
	for i, pr := range sorted {
		rows[i] = table.Row(render.PRRow(pr, now))
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	return Model{table: t, prs: sorted, pipelinesText: pipelinesText, open: open}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if len(m.prs) == 0 {
				return m, nil
			}
			idx := m.table.Cursor()
			if idx < 0 || idx >= len(m.prs) {
				return m, nil
			}
			if err := m.open(m.prs[idx].URL); err != nil {
				m.status = fmt.Sprintf("couldn't open browser: %v", err)
			} else {
				m.status = ""
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		pipelineLines := strings.Count(m.pipelinesText, "\n")
		tableHeight := msg.Height - pipelineLines - 2
		if tableHeight < 1 {
			tableHeight = 1
		}
		m.table.SetHeight(tableHeight)
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	status := m.status
	if status == "" {
		status = helpText
	}
	return m.table.View() + "\n\n" + m.pipelinesText + "\n" + status
}

// Run starts the full-screen interactive session, blocking until the
// user quits (Esc or q).
func Run(prs []model.PullRequest, pipelinesText string, now time.Time, width, height int) error {
	m := NewModel(prs, now, pipelinesText, width, height, openInBrowser)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
```

- [ ] **Step 4: Add the table dependency**

Run: `go get github.com/charmbracelet/bubbles/table`
Expected: no new module added (it's already part of the `bubbles` dependency from Round 1) — `go.sum` may still update minor entries

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/tui/... -v`
Expected: PASS (all tests)

- [ ] **Step 6: Commit**

```bash
git add internal/tui/
git commit -m "feat: replace TUI viewport with a selectable table and Enter-to-open"
```

---

### Task 4: Wire the new `tui.Run` signature into the CLI

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Add `terminalHeight()` and update both `tui.Run` call sites**

Add alongside the existing `terminalWidth()` in `cmd/root.go`:

```go
// terminalHeight returns the current terminal's row height, or 0 when
// stdout isn't a terminal — mirrors terminalWidth().
func terminalHeight() int {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return 0
	}
	_, height, err := term.GetSize(fd)
	if err != nil {
		return 0
	}
	return height
}
```

Replace the mock branch:

```go
			if useMock {
				prs := mockdata.PullRequests()
				pipelines := mockdata.PipelineRuns()
				if interactive {
					return tui.Run(prs, render.PipelinesText(pipelines, now), now, terminalWidth(), terminalHeight())
				}
				cmd.Print(render.FormatTable(prs, pipelines, now, terminalWidth()))
				return nil
			}
```

Replace the real-fetch branch's final block:

```go
			if interactive {
				return tui.Run(prs, render.PipelinesText(pipelines, now), now, terminalWidth(), terminalHeight())
			}
			cmd.Print(render.FormatTable(prs, pipelines, now, terminalWidth()))
			return nil
```

- [ ] **Step 2: Run the full suite**

Run: `go test ./... -v`
Expected: PASS across all packages (there's no automated test exercising the new call sites directly beyond compilation, same as Round 1 — `TestRootCommand_InteractiveFlagRegistered` already covers flag registration and still passes unchanged)

- [ ] **Step 3: Commit**

```bash
git add cmd/root.go
git commit -m "feat: wire selectable table + browser-open into --interactive"
```

---

### Task 5: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Full suite, vet, format**

Run: `go vet ./... && gofmt -l . && go test ./...`
Expected: `gofmt -l .` prints nothing; `go test ./...` reports `ok` for every package

- [ ] **Step 2: Confirm non-interactive mode is unaffected**

Run: `go build -o daily-pulse . && ./daily-pulse --mock`
Expected: identical static output to before this plan (Task 1 was a pure refactor; `FormatTable`'s behavior didn't change)

- [ ] **Step 3: Manual smoke test in a real terminal**

Run: `./daily-pulse --mock --interactive`
Expected: a full-screen session with a highlighted cursor row in the OPEN PRs/MRs table; Up/Down/`j`/`k` move it; `Enter` opens the selected row's URL in the default browser (mock data's URLs point at real GitHub/GitLab URLs for the mock repos, so this is safe to try); PIPELINES renders as static text below; `Esc`/`q` quits cleanly. This can't be exercised from an automated/non-tty environment — same limitation Round 1 had.

## Plan self-review notes

- **Spec coverage:** the table-based selection, PIPELINES staying static/non-selectable, the shared render-package helpers avoiding duplication, the injectable opener (darwin/linux/windows + fallback), the one-line status/help bar, and "no live column-width reflow" (widths are fixed from `NewModel`'s `width` param, never recomputed on `WindowSizeMsg`) are all covered.
- **Type consistency:** `render.PRColumnWidths`/`render.PRRow`/`render.SortPullRequests`/`render.PipelinesText` signatures introduced in Task 1 are used identically in Task 3's `tui.go`. `opener` (Task 2) is the exact type `NewModel` and `Run` take in Task 3. `tui.Run`'s new 5-argument signature (Task 3) matches both call sites added in Task 4.

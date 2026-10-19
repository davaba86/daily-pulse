# daily-pulse Interactive Session, Round 1 (Scrollable Viewport) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `daily-pulse --interactive`, a full-screen session (like `htop`) that shows the exact same output `render.FormatTable` already produces, scrollable with arrow/`j`/`k`/PgUp/PgDn, quitting with `Esc` or `q`. No row selection or actions yet.

**Architecture:** A new `internal/tui` package wraps `charmbracelet/bubbles/viewport` in a minimal `charmbracelet/bubbletea` model — content comes from the existing `render.FormatTable` string, no new rendering logic. `cmd/root.go` gains an `--interactive` flag that, when set, hands the already-rendered string to `tui.Run` instead of printing it.

**Tech Stack:** `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/bubbles/viewport` (both from the same Charm ecosystem as the already-approved "beautiful CLI" direction).

Spec: `docs/superpowers/specs/2026-10-18-daily-pulse-interactive-viewport-design.md`

---

## File Structure

```
internal/tui/tui.go       new: bubbletea Model (wraps viewport) + Run(content string) error
internal/tui/tui_test.go  new: Update() key-handling tests (quit, scroll), no real terminal needed
cmd/root.go                modify: add --interactive flag, branch to tui.Run on the rendered string
cmd/root_test.go           modify: assert the --interactive flag is registered and defaults to false
```

---

### Task 1: TUI model (viewport + quit)

**Files:**
- Create: `internal/tui/tui.go`
- Test: `internal/tui/tui_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func manyLines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = "line"
	}
	return strings.Join(lines, "\n")
}

func TestUpdate_EscQuits(t *testing.T) {
	m := NewModel(manyLines(5), 80, 24)

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
	m := NewModel(manyLines(5), 80, 24)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	model := updated.(Model)

	if !model.quitting {
		t.Error("Update('q') should set quitting = true")
	}
	if cmd == nil {
		t.Error("Update('q') should return tea.Quit, got nil cmd")
	}
}

func TestUpdate_ScrollsDown(t *testing.T) {
	// A viewport shorter than the content so there's something to scroll.
	m := NewModel(manyLines(50), 80, 10)
	before := m.viewport.YOffset

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := updated.(Model)

	if model.viewport.YOffset <= before {
		t.Errorf("Update(Down) should scroll the viewport forward; before=%d after=%d", before, model.viewport.YOffset)
	}
}

func TestView_EmptyWhenQuitting(t *testing.T) {
	m := NewModel(manyLines(5), 80, 24)
	m.quitting = true

	if got := m.View(); got != "" {
		t.Errorf("View() while quitting = %q, want empty string", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/... -v`
Expected: FAIL — package/`NewModel`/`Model` don't exist yet

- [ ] **Step 3: Write the implementation**

```go
package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	viewport viewport.Model
	quitting bool
}

func NewModel(content string, width, height int) Model {
	vp := viewport.New(width, height)
	vp.SetContent(content)
	return Model{viewport: vp}
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
		}
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	return m.viewport.View()
}

// Run starts the full-screen interactive session, blocking until the
// user quits (Esc or q).
func Run(content string) error {
	m := NewModel(content, 0, 0)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
```

- [ ] **Step 4: Add dependencies**

Run: `go get github.com/charmbracelet/bubbletea github.com/charmbracelet/bubbles`
Expected: `go.mod`/`go.sum` updated with both require entries

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/tui/... -v`
Expected: PASS (all four tests)

- [ ] **Step 6: Commit**

```bash
git add internal/tui/ go.mod go.sum
git commit -m "feat: add scrollable TUI viewport model with Esc/q to quit"
```

---

### Task 2: Wire `--interactive` into the CLI

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/root_test.go`

- [ ] **Step 1: Write the failing test**

Append to `cmd/root_test.go`:

```go
func TestRootCommand_InteractiveFlagRegistered(t *testing.T) {
	cmd := NewRootCommand()
	flag := cmd.Flags().Lookup("interactive")
	if flag == nil {
		t.Fatal("--interactive flag not registered")
	}
	if flag.DefValue != "false" {
		t.Errorf("--interactive default = %q, want %q", flag.DefValue, "false")
	}
}
```

(The interactive session itself takes over the terminal via Bubble Tea's
`Run()`, which needs a real tty — it isn't exercised by this automated
test. Manual smoke testing in a real terminal covers that, same as the
spec's testing section describes.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -v`
Expected: FAIL — `--interactive` flag not registered

- [ ] **Step 3: Write the implementation**

In `cmd/root.go`, add the import:

```go
	"github.com/davaba86/daily-pulse/internal/tui"
```

Add the flag variable and registration (alongside the existing `useMock`/`showAll`/`configPath` vars and `cmd.Flags()` calls):

```go
	var interactive bool
```

```go
	cmd.Flags().BoolVar(&interactive, "interactive", false, "enter a full-screen scrollable session instead of printing once")
```

Then change both places that currently do `cmd.Print(render.FormatTable(...))` to route through a shared `output` variable so `--interactive` can intercept it. Replace the mock branch:

```go
			if useMock {
				output := render.FormatTable(mockdata.PullRequests(), mockdata.PipelineRuns(), now, terminalWidth())
				if interactive {
					return tui.Run(output)
				}
				cmd.Print(output)
				return nil
			}
```

And the real-fetch branch's final block:

```go
			output := render.FormatTable(prs, pipelines, now, terminalWidth())
			if interactive {
				return tui.Run(output)
			}
			cmd.Print(output)
			return nil
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./... -v`
Expected: PASS across all packages

- [ ] **Step 5: Manual smoke test in a real terminal**

Run: `go build -o daily-pulse . && ./daily-pulse --mock --interactive`
Expected: full-screen session opens showing the same table content as
`./daily-pulse --mock`; Up/Down/`j`/`k`/PgUp/PgDn scroll; `Esc` or `q`
returns to the shell cleanly (no leftover alt-screen artifacts)

- [ ] **Step 6: Commit**

```bash
git add cmd/
git commit -m "feat: add --interactive flag for full-screen scrollable session"
```

---

### Task 3: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Full suite, vet, format**

Run: `go vet ./... && gofmt -l . && go test ./...`
Expected: `gofmt -l .` prints nothing; `go test ./...` reports `ok` for every package

- [ ] **Step 2: Manual smoke test against real data**

Run: `./daily-pulse --all --interactive` (using the real `~/.config/daily-pulse/config.yaml` from earlier testing)
Expected: same real PR/MR + pipeline data as `./daily-pulse --all`, now in a scrollable full-screen session; `Esc`/`q` exits cleanly

- [ ] **Step 3: Confirm non-interactive mode is unaffected**

Run: `./daily-pulse --mock` (no `--interactive`)
Expected: identical output to before this plan — prints once, exits, no alt-screen

## Plan self-review notes

- **Spec coverage:** the `--interactive` flag, viewport-based scrolling, Esc/q quit, reuse of `render.FormatTable`'s existing output, and "non-interactive mode stays byte-for-byte unchanged" are all covered (Tasks 1-2, verified in Task 3). Row selection/actions, filtering, and sorting are explicitly out of scope per the spec and not touched here.
- **Type consistency:** `Model`'s fields (`viewport`, `quitting`) and `NewModel`/`Run` signatures are defined once in Task 1 and used identically in Task 2's `cmd/root.go` wiring.

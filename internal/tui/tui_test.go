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

func TestClampTableHeight(t *testing.T) {
	cases := []struct {
		available, rowCount, want int
	}{
		{available: 30, rowCount: 3, want: 3}, // more space than rows: don't pad with blank rows
		{available: 2, rowCount: 10, want: 2}, // less space than rows: cap to available, let it scroll
		{available: 5, rowCount: 0, want: 1},  // no rows: still show at least 1 row of space
		{available: 0, rowCount: 5, want: 1},  // no space: never go below 1
		{available: -3, rowCount: 5, want: 1}, // negative available (tiny terminal): clamp to 1
	}
	for _, c := range cases {
		if got := clampTableHeight(c.available, c.rowCount); got != c.want {
			t.Errorf("clampTableHeight(%d, %d) = %d, want %d", c.available, c.rowCount, got, c.want)
		}
	}
}

func TestUpdate_WindowSizeClampsTableHeight(t *testing.T) {
	m := NewModel(testPRs(), time.Now(), "", 80, 24, func(string) error { return nil })

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 50})
	model := updated.(Model)

	if got := model.table.Height(); got != len(testPRs()) {
		t.Errorf("table.Height() after large WindowSizeMsg = %d, want %d (row count, not full available space)", got, len(testPRs()))
	}
}

func TestView_IncludesPipelinesText(t *testing.T) {
	m := NewModel(testPRs(), time.Now(), "PIPELINES TEXT HERE", 80, 24, func(string) error { return nil })

	if !strings.Contains(m.View(), "PIPELINES TEXT HERE") {
		t.Error("View() should include the pipelines text")
	}
}

func TestView_DividersBetweenSections(t *testing.T) {
	m := NewModel(testPRs(), time.Now(), "PIPELINES TEXT\n\n", 20, 24, func(string) error { return nil })

	divider := strings.Repeat("─", 20)
	if got := strings.Count(m.View(), divider); got != 2 {
		t.Errorf("View() divider count = %d, want 2 (before pipelines, before status)", got)
	}
}

func TestView_NoDividerBeforeEmptyPipelines(t *testing.T) {
	m := NewModel(testPRs(), time.Now(), "", 20, 24, func(string) error { return nil })

	divider := strings.Repeat("─", 20)
	if got := strings.Count(m.View(), divider); got != 1 {
		t.Errorf("View() divider count = %d, want 1 (only before status, since pipelines text is empty)", got)
	}
}

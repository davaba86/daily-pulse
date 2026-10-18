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

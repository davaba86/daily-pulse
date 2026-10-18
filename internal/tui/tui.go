package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/davaba86/daily-pulse/internal/model"
	"github.com/davaba86/daily-pulse/internal/render"
)

const helpText = "↑/↓ navigate · enter open in browser · esc/q quit"

var boxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)

// clampTableHeight returns how many data rows the table should show: never
// more than rowCount (bubbles/table otherwise pads with blank rows to fill
// whatever height it's given), never less than 1.
func clampTableHeight(available, rowCount int) int {
	h := available
	if rowCount < h {
		h = rowCount
	}
	if h < 1 {
		h = 1
	}
	return h
}

// tableHeaderHeight is how many lines bubbles/table reserves for its
// header row — table.Model's SetHeight/WithHeight both subtract this
// internally, so callers passing a desired *data*-row count (as
// clampTableHeight returns) need to add it back.
const tableHeaderHeight = 1

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
		table.WithHeight(clampTableHeight(height, len(rows))+tableHeaderHeight),
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
		// Fixed overhead outside the table's own data rows: our "OPEN
		// PRs/MRs" title line, the table's own header row, its box's
		// top+bottom border, the blank line after the box, and the
		// status line — 6 lines total, plus the PIPELINES box (its own
		// content height + 2 border lines + 1 trailing blank) when
		// present. Approximate — a tall PR list may end up a line or
		// two off, which just means the last row scrolls slightly.
		overhead := 6
		if m.pipelinesText != "" {
			overhead += strings.Count(m.pipelinesText, "\n") + 3
		}
		available := msg.Height - overhead
		m.table.SetHeight(clampTableHeight(available, len(m.prs)) + tableHeaderHeight)
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

	var b strings.Builder
	b.WriteString(boxStyle.Render("OPEN PRs/MRs\n" + m.table.View()))
	b.WriteString("\n")
	if m.pipelinesText != "" {
		b.WriteString(boxStyle.Render(strings.TrimRight(m.pipelinesText, "\n")))
		b.WriteString("\n")
	}
	b.WriteString(status)
	return b.String()
}

// Run starts the full-screen interactive session, blocking until the
// user quits (Esc or q).
func Run(prs []model.PullRequest, pipelinesText string, now time.Time, width, height int) error {
	m := NewModel(prs, now, pipelinesText, width, height, openInBrowser)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

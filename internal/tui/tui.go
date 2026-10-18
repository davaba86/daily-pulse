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

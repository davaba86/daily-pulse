package render

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/davaba86/daily-pulse/internal/model"
	"github.com/fatih/color"
)

// columnGap matches the tabwriter padding passed to tabwriter.NewWriter
// below, so natural-width calculations here agree with what tabwriter
// actually renders.
const columnGap = 2

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

func sourceLabel(source string) string {
	switch source {
	case "github":
		return "GH"
	case "gitlab":
		return "GL"
	default:
		return source
	}
}

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

// padCell right-pads s with spaces to width, measured in runes. It's used
// instead of tabwriter for the STATUS column because tabwriter counts
// ANSI color escape bytes as visible width, which would misalign rows
// where only some statuses (pass/fail/running, not none) get colorized.
func padCell(s string, width int) string {
	return s + strings.Repeat(" ", width-utf8.RuneCountInString(s))
}

// padColorized pads a colorized string to width, measured by the rune
// count of its plain (uncolorized) form — padCell can't be used directly
// since the ANSI escape runes in s would otherwise count toward width.
func padColorized(colorized, plain string, width int) string {
	return colorized + strings.Repeat(" ", width-utf8.RuneCountInString(plain))
}

func pipelineColumnWidths(pipelines []model.PipelineRun) (repo, status, branch, duration int) {
	repo = utf8.RuneCountInString("REPO")
	status = utf8.RuneCountInString("STATUS")
	branch = utf8.RuneCountInString("BRANCH")
	duration = utf8.RuneCountInString("DURATION")

	for _, p := range pipelines {
		d := p.Duration.String()
		if p.Status == model.PipelineNone {
			d = "—"
		}
		repo = max(repo, utf8.RuneCountInString(p.Repo))
		status = max(status, utf8.RuneCountInString(string(p.Status)))
		branch = max(branch, utf8.RuneCountInString(p.Branch))
		duration = max(duration, utf8.RuneCountInString(d))
	}
	return
}

func writePipelineSection(buf *bytes.Buffer, pipelines []model.PipelineRun, now time.Time) {
	if len(pipelines) == 0 {
		return
	}
	repoW, statusW, branchW, durationW := pipelineColumnWidths(pipelines)

	buf.WriteString("PIPELINES  (main/default branch)\n")
	fmt.Fprintf(buf, "%s  %s  %s  %s  %s\n",
		padCell("REPO", repoW), padCell("STATUS", statusW), padCell("BRANCH", branchW),
		padCell("DURATION", durationW), "UPDATED")

	for _, p := range pipelines {
		duration, updated := p.Duration.String(), relativeTime(p.UpdatedAt, now)
		if p.Status == model.PipelineNone {
			duration, updated = "—", "—"
		}
		statusCell := padColorized(Colorize(string(p.Status)), string(p.Status), statusW)
		fmt.Fprintf(buf, "%s  %s  %s  %s  %s\n",
			padCell(p.Repo, repoW), statusCell, padCell(p.Branch, branchW), padCell(duration, durationW), updated)
	}
	buf.WriteString("\n")
}

// PipelinesText renders just the PIPELINES section, for the interactive
// session to display alongside its selectable PR/MR table.
func PipelinesText(pipelines []model.PipelineRun, now time.Time) string {
	var buf bytes.Buffer
	writePipelineSection(&buf, pipelines, now)
	return buf.String()
}

func FormatTable(prs []model.PullRequest, pipelines []model.PipelineRun, now time.Time, width int) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "daily-pulse · %s\n\n", now.Format("2006-01-02 15:04"))

	writePRSection(&buf, prs, now, width)
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

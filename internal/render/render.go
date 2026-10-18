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

// titleStretch returns how many extra spaces the TITLE header needs so the
// table's total width reaches the terminal width, or 0 if the table is
// already at least that wide naturally.
func titleStretch(prs []model.PullRequest, now time.Time, width int) int {
	maxSource := utf8.RuneCountInString("SOURCE")
	maxRepo := utf8.RuneCountInString("REPO")
	maxNumber := utf8.RuneCountInString("#/!")
	maxTitle := utf8.RuneCountInString("TITLE")
	maxReviewers := utf8.RuneCountInString("REVIEWERS")
	maxUpdated := utf8.RuneCountInString("UPDATED")

	for _, pr := range prs {
		maxSource = max(maxSource, utf8.RuneCountInString(sourceLabel(pr.Source)))
		maxRepo = max(maxRepo, utf8.RuneCountInString(pr.Repo))
		maxNumber = max(maxNumber, utf8.RuneCountInString(strconv.Itoa(pr.Number)))
		maxTitle = max(maxTitle, utf8.RuneCountInString(pr.Title))
		maxReviewers = max(maxReviewers, utf8.RuneCountInString(reviewersCell(pr.Reviewers)))
		maxUpdated = max(maxUpdated, utf8.RuneCountInString(relativeTime(pr.UpdatedAt, now)))
	}

	natural := maxSource + maxRepo + maxNumber + maxTitle + maxReviewers + maxUpdated + 5*columnGap
	if width <= natural {
		return 0
	}
	return width - natural
}

func writePRSection(buf *bytes.Buffer, prs []model.PullRequest, now time.Time, width int) {
	if len(prs) == 0 {
		return
	}
	sorted := make([]model.PullRequest, len(prs))
	copy(sorted, prs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].UpdatedAt.After(sorted[j].UpdatedAt) })

	titleHeader := "TITLE" + strings.Repeat(" ", titleStretch(sorted, now, width))

	buf.WriteString("OPEN PRs/MRs\n")
	tw := tabwriter.NewWriter(buf, 0, 0, columnGap, ' ', 0)
	fmt.Fprintf(tw, "SOURCE\tREPO\t#/!\t%s\tREVIEWERS\tUPDATED\n", titleHeader)
	for _, pr := range sorted {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%s\t%s\n",
			sourceLabel(pr.Source), pr.Repo, pr.Number, pr.Title, reviewersCell(pr.Reviewers), relativeTime(pr.UpdatedAt, now))
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
		duration, updated := p.Duration.String(), relativeTime(p.UpdatedAt, now)
		if p.Status == model.PipelineNone {
			duration, updated = "—", "—"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", p.Repo, string(p.Status), p.Branch, duration, updated)
	}
	tw.Flush()
	buf.WriteString("\n")
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

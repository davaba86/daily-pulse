package render

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
	"github.com/fatih/color"
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

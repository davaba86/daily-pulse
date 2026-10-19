package render

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
	"github.com/fatih/color"
)

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

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

	out := FormatTable(prs, pipelines, nil, nil, now, 0)

	for _, want := range []string{
		"OPEN PRs/MRs",
		"SOURCE", "GH", "482", "fix-auth token refresh", "jsmith", "12m ago",
		"GL", "91", "add SSO login flow", "agupta", "5m ago",
		"PIPELINES", "davaba86/mc-platform-cloud", "pass", "main", "3m12s",
		"2 open PRs/MRs",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatTable() missing %q in output:\n%s", want, out)
		}
	}

	// gitlab PR updated 5m ago is more recent than github PR updated 12m
	// ago, so it should render first.
	glIdx := strings.Index(out, "GL")
	ghIdx := strings.Index(out, "GH")
	if glIdx == -1 || ghIdx == -1 || glIdx > ghIdx {
		t.Errorf("FormatTable() rows not sorted by most-recently-updated first, got:\n%s", out)
	}
}

func TestFormatTable_PipelineNone(t *testing.T) {
	pipelines := []model.PipelineRun{
		{Repo: "davaba86/daily-pulse", Status: model.PipelineNone, Branch: "main"},
	}

	out := FormatTable(nil, pipelines, nil, nil, fixedNow(), 0)

	var fields []string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "davaba86/daily-pulse") {
			fields = strings.Fields(line)
		}
	}
	if len(fields) != 5 {
		t.Fatalf("pipeline row fields = %v, want 5 fields (repo, status, branch, duration, updated)", fields)
	}
	if fields[3] != "—" || fields[4] != "—" {
		t.Errorf("duration/updated = %q/%q, want —/— for a repo with no pipeline runs", fields[3], fields[4])
	}
}

func TestFormatTable_NoPullRequests(t *testing.T) {
	out := FormatTable(nil, nil, nil, nil, fixedNow(), 0)
	if !strings.Contains(out, "0 open PRs/MRs") {
		t.Errorf("FormatTable() with no PRs missing summary line, got:\n%s", out)
	}
}

func TestFormatTable_StretchTitleColumn(t *testing.T) {
	now := fixedNow()
	prs := []model.PullRequest{
		{Source: "github", Repo: "davaba86/x", Number: 1, Title: "short", Reviewers: []string{"jsmith"}, UpdatedAt: now},
	}

	natural := FormatTable(prs, nil, nil, nil, now, 0)
	stretched := FormatTable(prs, nil, nil, nil, now, 120)

	naturalIdx := strings.Index(natural, "jsmith")
	stretchedIdx := strings.Index(stretched, "jsmith")
	if naturalIdx == -1 || stretchedIdx == -1 || stretchedIdx <= naturalIdx {
		t.Errorf("stretched REVIEWERS column should start further right than natural; natural idx=%d stretched idx=%d\nnatural:\n%s\nstretched:\n%s",
			naturalIdx, stretchedIdx, natural, stretched)
	}
}

func TestFormatTable_NoStretchWhenWidthTooSmall(t *testing.T) {
	now := fixedNow()
	prs := []model.PullRequest{
		{Source: "github", Repo: "davaba86/x", Number: 1, Title: "a long enough title to exceed a tiny width", Reviewers: []string{"jsmith"}, UpdatedAt: now},
	}

	natural := FormatTable(prs, nil, nil, nil, now, 0)
	tooSmall := FormatTable(prs, nil, nil, nil, now, 10)

	if natural != tooSmall {
		t.Errorf("width smaller than natural table width should not change output\nnatural:\n%s\ntooSmall:\n%s", natural, tooSmall)
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

func TestFormatTable_PipelineStatusColorizedAndAligned(t *testing.T) {
	color.NoColor = false
	defer func() { color.NoColor = true }()

	now := fixedNow()
	pipelines := []model.PipelineRun{
		{Repo: "davaba86/x", Status: model.PipelinePass, Branch: "main", Duration: 5 * time.Second, UpdatedAt: now},
		{Repo: "davaba86/y", Status: model.PipelineNone, Branch: "main"},
	}

	out := FormatTable(nil, pipelines, nil, nil, now, 0)

	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("FormatTable() pipeline section missing ANSI color codes, got:\n%s", out)
	}

	var passLine, noneLine string
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.Contains(line, "davaba86/x"):
			passLine = line
		case strings.Contains(line, "davaba86/y"):
			noneLine = line
		}
	}
	if passLine == "" || noneLine == "" {
		t.Fatalf("missing expected pipeline rows in output:\n%s", out)
	}

	passIdx := strings.Index(stripANSI(passLine), "main")
	noneIdx := strings.Index(stripANSI(noneLine), "main")
	if passIdx != noneIdx {
		t.Errorf("colorized (pass) and uncolored (none) pipeline rows misaligned: BRANCH starts at %d vs %d\ncolorized: %q\nuncolored: %q",
			passIdx, noneIdx, passLine, noneLine)
	}
}

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

func TestFormatTable_JiraSections(t *testing.T) {
	now := fixedNow()
	open := []model.JiraTicket{
		{Key: "ENG-482", Summary: "Fix auth token refresh", Status: "In Progress", UpdatedAt: now.Add(-12 * time.Minute)},
	}
	completed := []model.JiraTicket{
		{Key: "ENG-470", Summary: "Bump terraform providers", UpdatedAt: now.Add(-48 * time.Hour)},
	}

	out := FormatTable(nil, nil, open, completed, now, 0)

	for _, want := range []string{
		"JIRA · open", "ENG-482", "Fix auth token refresh", "In Progress", "12m ago",
		"JIRA · completed recently", "ENG-470", "Bump terraform providers", "2d ago",
		"1 open tickets",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatTable() missing %q in output:\n%s", want, out)
		}
	}
}

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

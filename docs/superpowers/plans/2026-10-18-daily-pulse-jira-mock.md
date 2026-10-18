# daily-pulse Jira Section (Mock Data) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Jira section (open tickets + recently-completed tickets) to the `daily-pulse` combined overview, backed entirely by mock data — no real Jira API or auth in this plan.

**Architecture:** Extend the existing `model`/`mockdata`/`render` packages from Phase 1 the same way GitHub/GitLab were built: a plain `model.JiraTicket` type, two new `internal/mockdata` functions returning pre-categorized fixture tickets, and two new sections in `render.FormatTable` (which gains two new parameters). `cmd/root.go` wires the mock Jira data into the `--mock` path; the real-fetch path passes empty slices since Jira isn't wired to a live API yet.

**Tech Stack:** Go, same packages as Phase 1 (`internal/model`, `internal/mockdata`, `internal/render`, `cmd`).

Spec: `docs/superpowers/specs/2026-10-18-daily-pulse-jira-mock-design.md`

---

## File Structure

```
internal/model/model.go           modify: add JiraTicket type
internal/mockdata/mockdata.go     modify: add JiraOpenTickets(), JiraCompletedTickets()
internal/mockdata/mockdata_test.go modify: add tests for the two new functions
internal/render/render.go         modify: add Jira sections, change FormatTable signature
internal/render/render_test.go    modify: update existing FormatTable calls, add Jira section test
cmd/root.go                       modify: wire mock Jira data into both FormatTable call sites
cmd/root_test.go                  modify: assert JIRA appears in --mock output
```

---

### Task 1: Jira domain type

**Files:**
- Modify: `internal/model/model.go`

- [ ] **Step 1: Add the type**

No test needed — plain type declaration, same treatment as `PullRequest`/`PipelineRun`.

Append to `internal/model/model.go`:

```go
type JiraTicket struct {
	Key       string
	Summary   string
	Status    string // e.g. "In Progress", "To Do" — empty for completed tickets
	UpdatedAt time.Time
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/model/`
Expected: no output (success)

- [ ] **Step 3: Commit**

```bash
git add internal/model/model.go
git commit -m "feat: add JiraTicket domain type"
```

---

### Task 2: Mock Jira data

**Files:**
- Modify: `internal/mockdata/mockdata.go`
- Modify: `internal/mockdata/mockdata_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/mockdata/mockdata_test.go`:

```go
func TestJiraOpenTickets(t *testing.T) {
	tickets := JiraOpenTickets()
	if len(tickets) != 2 {
		t.Fatalf("len(JiraOpenTickets()) = %d, want 2", len(tickets))
	}
}

func TestJiraCompletedTickets(t *testing.T) {
	tickets := JiraCompletedTickets()
	if len(tickets) != 1 {
		t.Fatalf("len(JiraCompletedTickets()) = %d, want 1", len(tickets))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mockdata/... -v`
Expected: FAIL — `JiraOpenTickets`/`JiraCompletedTickets` undefined

- [ ] **Step 3: Write the implementation**

Append to `internal/mockdata/mockdata.go`:

```go
func JiraOpenTickets() []model.JiraTicket {
	now := time.Now()
	return []model.JiraTicket{
		{
			Key:       "ENG-482",
			Summary:   "Fix auth token refresh",
			Status:    "In Progress",
			UpdatedAt: now.Add(-12 * time.Minute),
		},
		{
			Key:       "ENG-501",
			Summary:   "Add SSO login flow",
			Status:    "To Do",
			UpdatedAt: now.Add(-48 * time.Hour),
		},
	}
}

func JiraCompletedTickets() []model.JiraTicket {
	now := time.Now()
	return []model.JiraTicket{
		{
			Key:       "ENG-470",
			Summary:   "Bump terraform providers",
			UpdatedAt: now.Add(-48 * time.Hour),
		},
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mockdata/... -v`
Expected: PASS (all four tests, including the two from Task 4 of the Phase 1 plan)

- [ ] **Step 5: Commit**

```bash
git add internal/mockdata/
git commit -m "feat: add mock Jira open and completed ticket data"
```

---

### Task 3: Jira sections in render

**Files:**
- Modify: `internal/render/render.go`
- Modify: `internal/render/render_test.go`

- [ ] **Step 1: Update existing tests for the new `FormatTable` signature and add the Jira test**

`FormatTable` gains two new parameters (`openTickets`, `completedTickets`) before `now`. Update every existing call site in `internal/render/render_test.go` to pass `nil, nil` for them, and add a new test for the Jira sections.

In `TestFormatTable_Sections`, change:

```go
	out := FormatTable(prs, pipelines, now)
```

to:

```go
	out := FormatTable(prs, pipelines, nil, nil, now)
```

In `TestFormatTable_PipelineNone`, change:

```go
	out := FormatTable(nil, pipelines, fixedNow())
```

to:

```go
	out := FormatTable(nil, pipelines, nil, nil, fixedNow())
```

In `TestFormatTable_NoPullRequests`, change:

```go
	out := FormatTable(nil, nil, fixedNow())
```

to:

```go
	out := FormatTable(nil, nil, nil, nil, fixedNow())
```

Then append a new test:

```go
func TestFormatTable_JiraSections(t *testing.T) {
	now := fixedNow()
	open := []model.JiraTicket{
		{Key: "ENG-482", Summary: "Fix auth token refresh", Status: "In Progress", UpdatedAt: now.Add(-12 * time.Minute)},
	}
	completed := []model.JiraTicket{
		{Key: "ENG-470", Summary: "Bump terraform providers", UpdatedAt: now.Add(-48 * time.Hour)},
	}

	out := FormatTable(nil, nil, open, completed, now)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/render/... -v`
Expected: FAIL — too many arguments to `FormatTable` (compile error)

- [ ] **Step 3: Write the implementation**

Replace the `writePipelineSection` function's closing and the `FormatTable` function in `internal/render/render.go` with the following (adds a new `writeJiraSection` helper and updates `FormatTable`'s signature and body):

```go
func writeJiraSection(buf *bytes.Buffer, title string, tickets []model.JiraTicket, showStatus bool, now time.Time) {
	if len(tickets) == 0 {
		return
	}
	fmt.Fprintf(buf, "%s\n", title)
	tw := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	if showStatus {
		fmt.Fprintf(tw, "KEY\tSUMMARY\tSTATUS\tUPDATED\n")
		for _, tk := range tickets {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", tk.Key, tk.Summary, tk.Status, relativeTime(tk.UpdatedAt, now))
		}
	} else {
		fmt.Fprintf(tw, "KEY\tSUMMARY\tUPDATED\n")
		for _, tk := range tickets {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", tk.Key, tk.Summary, relativeTime(tk.UpdatedAt, now))
		}
	}
	tw.Flush()
	buf.WriteString("\n")
}

func FormatTable(prs []model.PullRequest, pipelines []model.PipelineRun, openTickets, completedTickets []model.JiraTicket, now time.Time) string {
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
	writeJiraSection(&buf, "JIRA · open (assigned or created by me)", openTickets, true, now)
	writeJiraSection(&buf, "JIRA · completed recently (7d)", completedTickets, false, now)

	failing := 0
	for _, p := range pipelines {
		if p.Status == model.PipelineFail {
			failing++
		}
	}
	fmt.Fprintf(&buf, "%d open PRs/MRs · %d open tickets · %d pipeline failing\n", len(prs), len(openTickets), failing)

	return buf.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/render/... -v`
Expected: PASS (all tests, including the new `TestFormatTable_JiraSections`)

- [ ] **Step 5: Commit**

```bash
git add internal/render/
git commit -m "feat: add Jira open/completed sections to table rendering"
```

---

### Task 4: Wire mock Jira data into the CLI

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/root_test.go`

- [ ] **Step 1: Write the failing test**

In `cmd/root_test.go`, update `TestRootCommand_Mock`'s assertion list:

```go
	for _, want := range []string{"GITHUB", "GITLAB", "PIPELINES", "JIRA", "open PRs/MRs"} {
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -v`
Expected: FAIL — output missing "JIRA" (compiles fine, assertion fails) — or a compile error if `FormatTable` call sites in `root.go` weren't updated yet. Either way, red.

Actually `cmd/root.go` currently calls `render.FormatTable` with the old signature, so this step will first fail to **compile** once Task 3 changed `FormatTable`'s signature. That compile failure is expected and is the "fails" signal for this step.

- [ ] **Step 3: Write the implementation**

In `cmd/root.go`, update the import block to add `"github.com/davaba86/daily-pulse/internal/model"` is already imported; no new import needed since `model` is already there.

Update the mock branch of `RunE`:

```go
			if useMock {
				cmd.Print(render.FormatTable(
					mockdata.PullRequests(), mockdata.PipelineRuns(),
					mockdata.JiraOpenTickets(), mockdata.JiraCompletedTickets(),
					now,
				))
				return nil
			}
```

Update the real-fetch branch's final render call:

```go
			cmd.Print(render.FormatTable(prs, pipelines, nil, nil, now))
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./... -v`
Expected: PASS across all packages

- [ ] **Step 5: Build and manually check output**

Run: `go build -o daily-pulse . && ./daily-pulse --mock`
Expected: output includes GITHUB, GITLAB, PIPELINES, then `JIRA · open (assigned or created by me)` and `JIRA · completed recently (7d)` sections matching the spec's mockup, and the summary line includes `· 1 open tickets ·` (two mock open tickets, so it will actually read `2 open tickets` — verify the printed count matches `len(mockdata.JiraOpenTickets())`, which is 2)

- [ ] **Step 6: Commit**

```bash
git add cmd/
git commit -m "feat: wire mock Jira data into CLI --mock output"
```

---

### Task 5: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Full suite, vet, format**

Run: `go vet ./... && gofmt -l . && go test ./...`
Expected: `gofmt -l .` prints nothing; `go test ./...` reports `ok` for every package

- [ ] **Step 2: Manual smoke test against mock data**

Run: `./daily-pulse --mock`
Expected: full combined output — GITHUB, GITLAB, PIPELINES, JIRA open, JIRA completed, summary line — matching the approved mockup

- [ ] **Step 3: Manual smoke test against real GitHub/GitLab (Jira stays empty)**

Run: `./daily-pulse --all` (using the real `~/.config/daily-pulse/config.yaml` from Phase 1 testing)
Expected: GITHUB/GITLAB/PIPELINES sections populated as before; no JIRA sections print (empty slices passed on the real-fetch path); summary line reads `· 0 open tickets ·`

## Plan self-review notes

- **Spec coverage:** both Jira sections, their exact columns, the combined-view CLI structure decision, and the summary-footer change are all covered (Tasks 1–4). Real Jira API/auth and comment-diffing alerts are explicitly out of scope per the spec and not touched here.
- **Type consistency:** `model.JiraTicket` field names (`Key`, `Summary`, `Status`, `UpdatedAt`) are defined once in Task 1 and used identically in Tasks 2–4. `FormatTable`'s new signature (`prs, pipelines, openTickets, completedTickets, now`) is introduced in Task 3 and used with the same parameter order in Task 4.

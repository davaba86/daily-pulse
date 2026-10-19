# daily-pulse — Interactive Session, Round 2 (Row Selection + Open in Browser) — Design Spec

Date: 2026-10-18
Status: Approved

## Purpose

Round 2 of the 4-increment interactive session (see
`2026-10-18-daily-pulse-interactive-viewport-design.md`): add a visible
cursor row to the OPEN PRs/MRs section and let `Enter` open the selected
row's URL in the default browser.

## Non-goals (this spec)

- Selecting/opening pipeline rows. `model.PipelineRun` has no `URL` field
  (unlike `model.PullRequest`, which already captures one unused since
  Phase 1). Adding pipeline URLs is a model change for a future round if
  wanted; PIPELINES stays static text under the table here.
- Filtering and sort-order cycling (Rounds 3 and 4).
- Live column-width reflow on terminal resize mid-session. Column widths
  are computed once at launch from the same `terminalWidth()` used by the
  static path today, matching how Round 1 already handles sizing.

## Design

### From viewport to a real table

Round 1 fed one opaque pre-rendered string into a `viewport`. A visible
cursor needs structured rows, so the PR/MR section becomes a
`charmbracelet/bubbles/table.Model` instead: each row knows its index,
`table.Model` already renders a highlighted cursor row by default, and
its default keymap already handles Up/Down/PgUp/PgDn navigation (same
keys Round 1 wired by hand for the viewport). PIPELINES remains static
text rendered beneath the table, unchanged from today — it's not
selectable in this round.

### Shared column-width logic

`render.go` already computes natural per-column widths and a TITLE
stretch amount for the static path (`titleStretch`). Round 2 extracts the
shared natural-width loop into one function both paths call:

```go
func prNaturalWidths(prs []model.PullRequest, now time.Time) (source, repo, number, title, reviewers, updated int)
```

`titleStretch` (static path) and a new exported `render.PRColumnWidths`
(interactive path, used to build `table.Column` definitions) both call
this instead of duplicating the loop.

### Data flow for the interactive path

`cmd/root.go`'s interactive branch now passes structured data to
`tui.Run` instead of one string:

```go
func Run(prs []model.PullRequest, pipelinesText string, now time.Time, width int) error
```

`tui.Run` sorts PRs the same way the static path does (reusing
`render`'s existing sort, exported as `render.SortPullRequests`) and
builds the table rows from the same per-row formatting the static path
uses (exported as `render.PRRow`), so the two paths can never visually
drift apart.

### Opening the browser

A small `internal/tui/browser.go` picks the OS-appropriate opener
(`open` on macOS, `xdg-open` on Linux, `rundll32 url.dll,FileProtocolHandler`
on Windows) via `runtime.GOOS`, run through an injectable function type
(same dependency-injection pattern `internal/githost.Runner` already
uses) so `Enter`'s behavior is unit-testable without actually launching a
browser:

```go
type opener func(url string) error
```

`Enter` on a PR/MR row calls the opener with that row's `model.PullRequest.URL`.
On error (e.g. no opener binary found), the session shows a one-line
status message and keeps running — never exits or crashes on a failed
open.

### Layout

Total terminal height splits between the table (gets the remaining
height after subtracting the pipelines text's line count and a status
line) and the static PIPELINES text pinned below it. A one-line help/
status bar at the bottom shows keybindings normally, or the last open
error when one just happened.

### Keybindings (this round)

- Up/Down, `j`/`k`, PgUp/PgDn: move the cursor (via `table.Model`'s
  default keymap, not reimplemented).
- `Enter`: open the selected row's URL in the browser.
- `Esc`/`q`: quit, same as Round 1.

## Testing

Same approach as Round 1: `Update()` is tested directly with synthetic
`tea.KeyMsg` values and an injected fake `opener`, asserting which URL it
was called with on `Enter` and that a failed open sets a visible error
message without quitting. No real terminal or real browser launch is
exercised by automated tests; a real terminal check is manual, same as
every other live-data check in this project.

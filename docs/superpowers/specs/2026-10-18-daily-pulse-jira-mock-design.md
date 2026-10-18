# daily-pulse — Jira Section (Mock Data) — Design Spec

Date: 2026-10-18
Status: Approved

## Purpose

Add a Jira section to `daily-pulse`'s combined overview, backed by mock data
only. This is a slice of Phase 2 from the main design spec
(`2026-10-18-daily-pulse-design.md`): render the Jira UI and data shape
first, the same mock-first order Phase 1 used for GitHub/GitLab, before any
real Jira REST/auth wiring.

## Non-goals (this spec)

- Real Jira REST API calls or authentication. A later increment wires
  `internal/jira` against live data using the same `Runner`-style
  dependency injection pattern as `internal/githost`.
- Comment-diffing / "new comment on my ticket" alerts. That's Phase 4
  territory in the main spec, same as GitHub/GitLab alerts.
- A Jira config schema (base URL, project keys, JQL). Deferred to the real
  Jira wiring increment, once there's a live API shape to configure against.

## CLI structure decision

GitHub/GitLab and Jira render in **one combined view**: `daily-pulse`
prints GITHUB, GITLAB, PIPELINES, then the new JIRA sections, in one
scrollable output — consistent with the tool's purpose (one command
instead of tab-hopping across web UIs). No separate `jira` subcommand.

## Jira sections

Two sections, both always shown when Jira mock data is present:

1. **`JIRA · open (assigned or created by me)`** — tickets assigned to the
   author or created by the author that aren't done, deduplicated (a
   ticket both assigned to and created by the author appears once).
   Columns: `KEY`, `SUMMARY`, `STATUS`, `UPDATED`.
2. **`JIRA · completed recently (7d)`** — tickets the author completed in
   the last 7 days. Columns: `KEY`, `SUMMARY`, `UPDATED` (no `STATUS`
   column — it's redundant once a ticket is done).

The summary footer gains an open-ticket count: `N open PRs/MRs · M open
tickets · K pipelines failing`. Recently-completed tickets aren't counted
in the footer — they're informational, not open backlog.

## Data shape

```go
type JiraTicket struct {
    Key       string // e.g. "ENG-482"
    Summary   string
    Status    string // e.g. "In Progress", "To Do" — empty for completed tickets
    UpdatedAt time.Time
}
```

`internal/mockdata` gains `JiraOpenTickets() []model.JiraTicket` and
`JiraCompletedTickets() []model.JiraTicket`, returning separate
pre-categorized slices — mirroring how mock PR/pipeline data is already
split by source. The real Jira integration will categorize via JQL
queries instead of a status-string check, so no filtering logic needs to
live in `render` or `model`.

## Output mockup

```
JIRA · open (assigned or created by me)
KEY      SUMMARY                  STATUS       UPDATED
ENG-482  Fix auth token refresh   In Progress  12m ago
ENG-501  Add SSO login flow       To Do        2d ago

JIRA · completed recently (7d)
KEY      SUMMARY                   UPDATED
ENG-470  Bump terraform providers  2d ago

2 open PRs/MRs · 2 open tickets · 0 pipelines failing
```

## Testing

Same approach as Phase 1: table-driven Go unit tests on `render` (fixture
tickets in, expected substrings/fields out) and on `mockdata` (fixed
counts). No network/auth-dependent code is introduced in this slice.

# daily-pulse — Design Spec

Date: 2026-10-18
Status: Approved (pending final author read-through)

## Purpose

A CLI tool that gives a daily overview of the author's work across GitHub,
GitLab, and Jira: open PRs/MRs (by repo, by author, filtered to "mine"),
CI/CD pipeline status per repo, Jira tickets assigned to/created by the
author, and — over time — a local history used to benchmark personal
throughput and responsiveness. Built to replace tab-hopping across
GitHub/GitLab/Jira web UIs with one terminal command.

Public, open-source project. MIT licensed. No company branding.

## Non-goals (this spec)

- AI-assisted Jira ticket creation / automation. Noted as a future idea,
  not designed or built here.
- Interactive TUI. Static table output only (see Output layout below); an
  `--interactive` mode may be added later but is not part of this spec.
- Multi-user / team dashboards. This is a single-user, local-first tool.

## Phases

1. **PR/MR + pipeline dashboard (MVP)** — open PRs/MRs across configured
   GitHub repos and GitLab projects, filterable to the author vs.
   everyone, rendered as static aligned tables; plus a pipeline status
   section showing the latest GitHub Actions / GitLab CI run on each
   tracked repo's default branch (pass/fail/running/none, branch,
   duration, last-updated). Same data source (`gh`/`glab` CLI) as the
   PR/MR fetch, so it ships in the same phase.

   Built in two steps: first wire `internal/render` against hardcoded
   mock fixture data (matching the Output layout mockup below) to lock
   down table formatting/alignment without needing live auth; then swap
   the mock data source for real `internal/githost` calls once rendering
   looks right. Later phases (Jira, stats) follow the same mock-first
   pattern before wiring live calls.
2. **Jira integration** — tickets assigned to/created by the author, with
   status, same table treatment.
3. **Benchmarking** — `daily-pulse stats` subcommand querying local history
   for: PR/MR throughput (opened/merged counts, time-to-merge, review
   turnaround), Jira ticket velocity (assigned/completed counts, avg
   time-to-close), review responsiveness (time from a comment landing to
   the author replying), and review load given to others (PRs/MRs the
   author reviewed for teammates, count and turnaround).
4. **Inline alerts** — diffing each run's snapshot against the previous one
   and flagging rows with new comments since last run (e.g. `● NEW`)
   directly in the normal table output. No separate daemon/notification
   mechanism in this phase.

## Architecture

```
cmd/              cobra CLI entrypoints: root (overview), stats
internal/config   YAML load/validate (~/.config/daily-pulse/config.yaml)
internal/githost  shells out to `gh`/`glab` CLI, parses their JSON output
internal/jira     REST client; auth via JIRA_URL/JIRA_EMAIL/JIRA_API_TOKEN env vars
internal/store    SQLite wrapper — snapshot writes, diffing, query helpers
internal/render   table rendering (aligned, colored columns) + alert markers
internal/stats    aggregation queries over store data (Phase 3)
```

### Output layout

Grouped sections per source, static aligned/colored tables, a one-line
summary footer. Rows with comments new since the last run get an inline
`● NEW` marker (Phase 4). Example:

```
daily-pulse · 2026-10-18 08:02

GITHUB · open PRs (mine)
REPO                 #    TITLE                      REVIEWERS       UPDATED
mc-platform-cloud    482  fix-auth token refresh      jsmith          12m ago
mc-platform-onprem   478  bump terraform providers    —               2d ago   ● NEW

GITLAB · open MRs (mine)
REPO                 !    TITLE                      REVIEWERS       UPDATED
mc-product-lifttraq  91   add SSO login flow         agupta          5m ago

PIPELINES  (main/default branch)
REPO                   STATUS    BRANCH   DURATION   UPDATED
mc-platform-cloud      ✓ pass    main     3m12s      12m ago
mc-platform-onprem     ✗ fail    main     1m48s      34m ago

2 open PRs/MRs · 1 new comment · 1 pipeline failing
```

### Data flow

1. Load YAML config (tracked GitHub orgs/repos, GitLab groups/projects,
   Jira base URL, display preferences).
2. Fetch live data:
   - GitHub/GitLab: shell out to `gh api` / `gh pr list --json ...` and
     `glab api` / `glab mr list` (JSON output), reusing whatever
     authentication the user already has set up for those CLIs. No
     separate token management for git hosts.
   - Jira (Phase 2): REST calls authenticated via env vars.
3. Write a snapshot of the fetched data to SQLite
   (`~/.local/share/daily-pulse/pulse.db`).
4. Diff the new snapshot against the previous one to compute alert markers
   (new comments since last run).
5. Render tables to stdout.

### Configuration

`~/.config/daily-pulse/config.yaml` — explicit list of GitHub
orgs/repos and GitLab groups/projects to track (no auto-discovery), Jira
base URL, display preferences. Config holds no secrets; all tokens come
from `gh`/`glab` CLI auth (git hosts) or environment variables (Jira).

### Error handling

- Missing or unauthenticated `gh`/`glab` CLI: clear, actionable error
  naming which CLI and how to fix (`gh auth login` / `glab auth login`).
- One host reachable, another not: show results for what succeeded, print
  a warning for what failed, do not abort the whole run.
- Jira env vars unset: skip the Jira section with a note — not an error,
  since Jira (Phase 2) is optional until configured.

## Testing

Table-driven Go unit tests for pure logic with no network dependency:

- Config parsing/validation (good and malformed YAML fixtures).
- Snapshot diffing / new-comment detection (fixture snapshot pairs).
- Stats aggregation (fixture history data → expected metrics).
- JSON-parsing layer for `gh`/`glab`/Jira responses, tested against fixture
  JSON captured from real responses — kept separate from the actual
  network/exec calls so the parsing logic is testable without live auth.

Actual `gh`/`glab` exec calls and live Jira REST calls are smoke-tested
manually; they are not meaningfully mockable without reimplementing CLI
auth, so unit coverage stops at the parsing boundary.

## Distribution

Public GitHub repo, MIT license. Binary distributed via the author's
existing Homebrew tap. Standard OSS repo structure (README, LICENSE,
CONTRIBUTING) — no internal company compliance scaffolding, since this
repo is public and the scaffolding is meaningless to external
contributors.

## Future ideas (explicitly out of scope here)

- AI-assisted creation of Jira tickets, to reduce ticket-writing overhead
  as part of the author's broader productivity workflow. Will get its own
  design pass if pursued.

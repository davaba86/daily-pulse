# daily-pulse — Interactive Session, Round 1 (Scrollable Viewport) — Design Spec

Date: 2026-10-18
Status: Approved

## Purpose

Add the `--interactive` mode flagged as a future possibility in the main
design spec's non-goals: `daily-pulse --interactive` enters a full-screen
session (like `htop`) instead of printing once and exiting. This is Round
1 of 4 planned increments:

1. **This spec:** scrollable viewport, `Esc`/`q` to quit. No row
   selection or actions yet — same data, same columns, just interactive
   instead of static.
2. Open the selected row's URL in the browser (needs row selection).
3. Type to filter visible rows.
4. Key to cycle sort order.

Rounds 2-4 are out of scope here and get their own spec/plan when built.

## Non-goals (this spec)

- Row selection/highlighting. There's no "current row" concept yet —
  that's introduced in Round 2 once something (opening a URL) needs it.
- Any new data or columns. The interactive session renders exactly what
  `render.FormatTable` already produces for the static mode.
- Filtering, sorting, or any other keybinding beyond scrolling and quit.
- Auto-refresh / live polling while the session is open. Entering the
  session fetches once (same as today), the same way `htop` differs from
  a true "session" only in that daily-pulse's data is a point-in-time
  snapshot, not live process stats. Refresh-while-open may be considered
  in a future round if wanted, but isn't scoped now.

## Design

- New `--interactive` boolean flag on the root command, alongside
  `--mock`/`--all`/`--config`. Works with both real and mock data, same
  as every other flag.
- Implementation: `charmbracelet/bubbletea` for the event loop,
  `charmbracelet/bubbles/viewport` for scrolling. The viewport's content
  is set directly from `render.FormatTable(...)`'s existing string
  output — no new rendering logic, no duplicated column/formatting code.
- Keybindings: Up/Down, `j`/`k`, PgUp/PgDn, Home/End scroll the viewport.
  `Esc` or `q` quits and returns to the shell, same as `htop`.
- When `--interactive` isn't passed, behavior is byte-for-byte unchanged
  from today: print once, exit. The static path and the interactive path
  share the same fetch/render code; only what happens with the resulting
  string differs (printed directly, vs. loaded into a viewport).

## Testing

Bubble Tea models separate "what keys do" (the `Update` function) from
actual terminal rendering, so the keybinding logic is unit-testable
without a real terminal: construct the model directly, call `Update`
with a `tea.KeyMsg{Type: tea.KeyEsc}` (or `j`/`k`/etc.), and assert on
the resulting model state (e.g. `quitting` becomes true, or the
viewport's `YOffset` changes). No fixture/golden-file testing of actual
rendered frames — that's what manual smoke testing in a real terminal
is for, same as Phase 1's live-data checks.

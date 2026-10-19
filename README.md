# daily-pulse

A CLI daily overview of your open GitHub PRs, GitLab MRs, and repo pipeline
status — one command instead of tab-hopping between web UIs.

## Install

```bash
go build -o daily-pulse .
```

## Prerequisites

- [`gh`](https://cli.github.com/) authenticated: `gh auth login`
- [`glab`](https://gitlab.com/gitlab-org/cli) authenticated: `glab auth login`

daily-pulse shells out to both CLIs and reuses whatever auth they already
have — it never stores a GitHub/GitLab token itself.

## Configure

Create `~/.config/daily-pulse/config.yaml`:

```yaml
github:
  username: davaba86
  repos:
    - owner: davaba86
      name: daily-pulse
gitlab:
  username: david.abarca
  projects:
    - path: mecha-consulting/mc-platform-cloud
```

## Usage

```bash
daily-pulse            # your open PRs/MRs + pipeline status for configured repos
daily-pulse --all      # everyone's open PRs/MRs, not just yours
daily-pulse --mock      # render against built-in sample data, no live calls
```

Each run also writes a snapshot to `~/.local/share/daily-pulse/pulse.db`,
used by future benchmarking features.

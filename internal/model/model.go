package model

import "time"

type PullRequest struct {
	Source    string // "github" or "gitlab"
	Repo      string // "owner/name" for GitHub, "group/project" path for GitLab
	Number    int
	Title     string
	URL       string
	Author    string
	Reviewers []string
	UpdatedAt time.Time
}

type PipelineStatus string

const (
	PipelinePass    PipelineStatus = "pass"
	PipelineFail    PipelineStatus = "fail"
	PipelineRunning PipelineStatus = "running"
	PipelineNone    PipelineStatus = "none"
)

type PipelineRun struct {
	Repo      string
	Status    PipelineStatus
	Branch    string
	Duration  time.Duration
	UpdatedAt time.Time
}

type JiraTicket struct {
	Key       string
	Summary   string
	Status    string // e.g. "In Progress", "To Do" — empty for completed tickets
	UpdatedAt time.Time
}

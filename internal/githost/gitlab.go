package githost

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

func encodeProjectPath(path string) string {
	return url.QueryEscape(path)
}

type gitlabUser struct {
	Username string `json:"username"`
}

type gitlabMergeRequest struct {
	IID       int          `json:"iid"`
	Title     string       `json:"title"`
	WebURL    string       `json:"web_url"`
	UpdatedAt time.Time    `json:"updated_at"`
	Author    gitlabUser   `json:"author"`
	Reviewers []gitlabUser `json:"reviewers"`
}

func FetchGitLabMergeRequests(run Runner, projectPath string) ([]model.PullRequest, error) {
	path := fmt.Sprintf("projects/%s/merge_requests?state=opened", encodeProjectPath(projectPath))
	out, err := run("glab", "api", path)
	if err != nil {
		return nil, fmt.Errorf("glab api %s: %w", path, err)
	}

	var mrs []gitlabMergeRequest
	if err := json.Unmarshal(out, &mrs); err != nil {
		return nil, fmt.Errorf("parse %s response: %w", path, err)
	}

	prs := make([]model.PullRequest, 0, len(mrs))
	for _, m := range mrs {
		reviewers := make([]string, 0, len(m.Reviewers))
		for _, r := range m.Reviewers {
			reviewers = append(reviewers, r.Username)
		}
		prs = append(prs, model.PullRequest{
			Source:    "gitlab",
			Repo:      projectPath,
			Number:    m.IID,
			Title:     m.Title,
			URL:       m.WebURL,
			Author:    m.Author.Username,
			Reviewers: reviewers,
			UpdatedAt: m.UpdatedAt,
		})
	}
	return prs, nil
}

type gitlabProject struct {
	DefaultBranch string `json:"default_branch"`
}

type gitlabPipelineSummary struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Ref    string `json:"ref"`
}

type gitlabPipelineDetail struct {
	ID        int       `json:"id"`
	Status    string    `json:"status"`
	Ref       string    `json:"ref"`
	Duration  int       `json:"duration"`
	UpdatedAt time.Time `json:"updated_at"`
}

// gitlabPipelineStatus maps GitLab pipeline statuses onto the four
// buckets daily-pulse renders. Only "success" is a pass and only
// "running"/"pending" count as in-progress; every other terminal status
// (failed, canceled, skipped) renders as "fail" so a non-green pipeline
// is never silently hidden.
func gitlabPipelineStatus(status string) model.PipelineStatus {
	switch status {
	case "success":
		return model.PipelinePass
	case "running", "pending":
		return model.PipelineRunning
	default:
		return model.PipelineFail
	}
}

func FetchGitLabPipelineStatus(run Runner, projectPath string) (model.PipelineRun, error) {
	encoded := encodeProjectPath(projectPath)

	projectOut, err := run("glab", "api", fmt.Sprintf("projects/%s", encoded))
	if err != nil {
		return model.PipelineRun{}, fmt.Errorf("glab api projects/%s: %w", encoded, err)
	}
	var project gitlabProject
	if err := json.Unmarshal(projectOut, &project); err != nil {
		return model.PipelineRun{}, fmt.Errorf("parse project response: %w", err)
	}

	listPath := fmt.Sprintf("projects/%s/pipelines?ref=%s&per_page=1", encoded, project.DefaultBranch)
	listOut, err := run("glab", "api", listPath)
	if err != nil {
		return model.PipelineRun{}, fmt.Errorf("glab api %s: %w", listPath, err)
	}
	var summaries []gitlabPipelineSummary
	if err := json.Unmarshal(listOut, &summaries); err != nil {
		return model.PipelineRun{}, fmt.Errorf("parse %s response: %w", listPath, err)
	}

	if len(summaries) == 0 {
		return model.PipelineRun{Repo: projectPath, Status: model.PipelineNone, Branch: project.DefaultBranch}, nil
	}

	detailPath := fmt.Sprintf("projects/%s/pipelines/%d", encoded, summaries[0].ID)
	detailOut, err := run("glab", "api", detailPath)
	if err != nil {
		return model.PipelineRun{}, fmt.Errorf("glab api %s: %w", detailPath, err)
	}
	var detail gitlabPipelineDetail
	if err := json.Unmarshal(detailOut, &detail); err != nil {
		return model.PipelineRun{}, fmt.Errorf("parse %s response: %w", detailPath, err)
	}

	return model.PipelineRun{
		Repo:      projectPath,
		Status:    gitlabPipelineStatus(detail.Status),
		Branch:    detail.Ref,
		Duration:  time.Duration(detail.Duration) * time.Second,
		UpdatedAt: detail.UpdatedAt,
	}, nil
}

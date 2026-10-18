package githost

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

type githubUser struct {
	Login string `json:"login"`
}

type githubPull struct {
	Number             int          `json:"number"`
	Title              string       `json:"title"`
	HTMLURL            string       `json:"html_url"`
	UpdatedAt          time.Time    `json:"updated_at"`
	User               githubUser   `json:"user"`
	RequestedReviewers []githubUser `json:"requested_reviewers"`
}

func FetchGitHubPullRequests(run Runner, owner, name string) ([]model.PullRequest, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls?state=open", owner, name)
	out, err := run("gh", "api", path)
	if err != nil {
		return nil, fmt.Errorf("gh api %s: %w", path, err)
	}

	var pulls []githubPull
	if err := json.Unmarshal(out, &pulls); err != nil {
		return nil, fmt.Errorf("parse %s response: %w", path, err)
	}

	prs := make([]model.PullRequest, 0, len(pulls))
	for _, p := range pulls {
		reviewers := make([]string, 0, len(p.RequestedReviewers))
		for _, r := range p.RequestedReviewers {
			reviewers = append(reviewers, r.Login)
		}
		prs = append(prs, model.PullRequest{
			Source:    "github",
			Repo:      fmt.Sprintf("%s/%s", owner, name),
			Number:    p.Number,
			Title:     p.Title,
			URL:       p.HTMLURL,
			Author:    p.User.Login,
			Reviewers: reviewers,
			UpdatedAt: p.UpdatedAt,
		})
	}
	return prs, nil
}

type githubRepo struct {
	DefaultBranch string `json:"default_branch"`
}

type githubRun struct {
	Status       string    `json:"status"`
	Conclusion   string    `json:"conclusion"`
	HeadBranch   string    `json:"head_branch"`
	RunStartedAt time.Time `json:"run_started_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type githubRunsResponse struct {
	WorkflowRuns []githubRun `json:"workflow_runs"`
}

func FetchGitHubPipelineStatus(run Runner, owner, name string) (model.PipelineRun, error) {
	repoPath := fmt.Sprintf("repos/%s/%s", owner, name)
	repoOut, err := run("gh", "api", repoPath)
	if err != nil {
		return model.PipelineRun{}, fmt.Errorf("gh api %s: %w", repoPath, err)
	}
	var repo githubRepo
	if err := json.Unmarshal(repoOut, &repo); err != nil {
		return model.PipelineRun{}, fmt.Errorf("parse %s response: %w", repoPath, err)
	}

	runsPath := fmt.Sprintf("repos/%s/%s/actions/runs?branch=%s&per_page=1", owner, name, repo.DefaultBranch)
	runsOut, err := run("gh", "api", runsPath)
	if err != nil {
		return model.PipelineRun{}, fmt.Errorf("gh api %s: %w", runsPath, err)
	}
	var resp githubRunsResponse
	if err := json.Unmarshal(runsOut, &resp); err != nil {
		return model.PipelineRun{}, fmt.Errorf("parse %s response: %w", runsPath, err)
	}

	repoFullName := fmt.Sprintf("%s/%s", owner, name)
	if len(resp.WorkflowRuns) == 0 {
		return model.PipelineRun{Repo: repoFullName, Status: model.PipelineNone, Branch: repo.DefaultBranch}, nil
	}

	r := resp.WorkflowRuns[0]
	status := model.PipelineRunning
	if r.Status == "completed" {
		if r.Conclusion == "success" {
			status = model.PipelinePass
		} else {
			status = model.PipelineFail
		}
	}

	return model.PipelineRun{
		Repo:      repoFullName,
		Status:    status,
		Branch:    repo.DefaultBranch,
		Duration:  r.UpdatedAt.Sub(r.RunStartedAt),
		UpdatedAt: r.UpdatedAt,
	}, nil
}

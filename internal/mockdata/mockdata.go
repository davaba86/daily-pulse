package mockdata

import (
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

func PullRequests() []model.PullRequest {
	now := time.Now()
	return []model.PullRequest{
		{
			Source:    "github",
			Repo:      "davaba86/mc-platform-cloud",
			Number:    482,
			Title:     "fix-auth token refresh",
			URL:       "https://github.com/davaba86/mc-platform-cloud/pull/482",
			Author:    "davaba86",
			Reviewers: []string{"jsmith"},
			UpdatedAt: now.Add(-12 * time.Minute),
		},
		{
			Source:    "github",
			Repo:      "davaba86/mc-platform-onprem",
			Number:    478,
			Title:     "bump terraform providers",
			URL:       "https://github.com/davaba86/mc-platform-onprem/pull/478",
			Author:    "davaba86",
			Reviewers: nil,
			UpdatedAt: now.Add(-48 * time.Hour),
		},
		{
			Source:    "gitlab",
			Repo:      "mecha-consulting/mc-product-lifttraq",
			Number:    91,
			Title:     "add SSO login flow",
			URL:       "https://gitlab.com/mecha-consulting/mc-product-lifttraq/-/merge_requests/91",
			Author:    "david.abarca",
			Reviewers: []string{"agupta"},
			UpdatedAt: now.Add(-5 * time.Minute),
		},
	}
}

func PipelineRuns() []model.PipelineRun {
	now := time.Now()
	return []model.PipelineRun{
		{
			Repo:      "davaba86/mc-platform-cloud",
			Status:    model.PipelinePass,
			Branch:    "main",
			Duration:  3*time.Minute + 12*time.Second,
			UpdatedAt: now.Add(-12 * time.Minute),
		},
		{
			Repo:      "davaba86/mc-platform-onprem",
			Status:    model.PipelineFail,
			Branch:    "main",
			Duration:  1*time.Minute + 48*time.Second,
			UpdatedAt: now.Add(-34 * time.Minute),
		},
	}
}

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

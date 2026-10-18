package githost

import (
	"os"
	"testing"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

func fakeRunner(t *testing.T, byArgsPrefix map[string]string) Runner {
	return func(name string, args ...string) ([]byte, error) {
		for prefix, fixture := range byArgsPrefix {
			if len(args) > 0 && args[len(args)-1] == prefix {
				return os.ReadFile(fixture)
			}
		}
		t.Fatalf("fakeRunner: no fixture for %s %v", name, args)
		return nil, nil
	}
}

func TestFetchPullRequests_GitHub(t *testing.T) {
	runner := fakeRunner(t, map[string]string{
		"repos/davaba86/mc-platform-cloud/pulls?state=open": "testdata/github_pulls.json",
	})

	prs, err := FetchGitHubPullRequests(runner, "davaba86", "mc-platform-cloud")
	if err != nil {
		t.Fatalf("FetchGitHubPullRequests() error = %v", err)
	}
	if len(prs) != 1 {
		t.Fatalf("len(prs) = %d, want 1", len(prs))
	}
	pr := prs[0]
	if pr.Source != "github" || pr.Repo != "davaba86/mc-platform-cloud" || pr.Number != 482 ||
		pr.Title != "fix-auth token refresh" || pr.Author != "davaba86" ||
		len(pr.Reviewers) != 1 || pr.Reviewers[0] != "jsmith" {
		t.Errorf("FetchGitHubPullRequests() = %+v, mismatch", pr)
	}
	wantUpdated := time.Date(2026, 10, 18, 7, 50, 0, 0, time.UTC)
	if !pr.UpdatedAt.Equal(wantUpdated) {
		t.Errorf("UpdatedAt = %v, want %v", pr.UpdatedAt, wantUpdated)
	}
}

func TestFetchPipelineStatus_GitHub(t *testing.T) {
	runner := fakeRunner(t, map[string]string{
		"repos/davaba86/mc-platform-cloud":                                      "testdata/github_repo.json",
		"repos/davaba86/mc-platform-cloud/actions/runs?branch=main&per_page=1": "testdata/github_runs.json",
	})

	run, err := FetchGitHubPipelineStatus(runner, "davaba86", "mc-platform-cloud")
	if err != nil {
		t.Fatalf("FetchGitHubPipelineStatus() error = %v", err)
	}
	if run.Repo != "davaba86/mc-platform-cloud" || run.Status != model.PipelinePass || run.Branch != "main" {
		t.Errorf("FetchGitHubPipelineStatus() = %+v, mismatch", run)
	}
	if run.Duration != 3*time.Minute+12*time.Second {
		t.Errorf("Duration = %v, want 3m12s", run.Duration)
	}
}

func TestFetchPipelineStatus_GitHub_NoRuns(t *testing.T) {
	runner := fakeRunner(t, map[string]string{
		"repos/davaba86/mc-platform-cloud":                                      "testdata/github_repo.json",
		"repos/davaba86/mc-platform-cloud/actions/runs?branch=main&per_page=1": "testdata/github_runs_empty.json",
	})

	run, err := FetchGitHubPipelineStatus(runner, "davaba86", "mc-platform-cloud")
	if err != nil {
		t.Fatalf("FetchGitHubPipelineStatus() error = %v", err)
	}
	if run.Status != model.PipelineNone {
		t.Errorf("Status = %v, want %v", run.Status, model.PipelineNone)
	}
}

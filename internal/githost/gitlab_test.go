package githost

import (
	"testing"
	"time"

	"github.com/davaba86/daily-pulse/internal/model"
)

func TestFetchMergeRequests_GitLab(t *testing.T) {
	runner := fakeRunner(t, map[string]string{
		"projects/mecha-consulting%2Fmc-product-lifttraq/merge_requests?state=opened": "testdata/gitlab_mrs.json",
	})

	mrs, err := FetchGitLabMergeRequests(runner, "mecha-consulting/mc-product-lifttraq")
	if err != nil {
		t.Fatalf("FetchGitLabMergeRequests() error = %v", err)
	}
	if len(mrs) != 1 {
		t.Fatalf("len(mrs) = %d, want 1", len(mrs))
	}
	mr := mrs[0]
	if mr.Source != "gitlab" || mr.Repo != "mecha-consulting/mc-product-lifttraq" || mr.Number != 91 ||
		mr.Title != "add SSO login flow" || mr.Author != "david.abarca" ||
		len(mr.Reviewers) != 1 || mr.Reviewers[0] != "agupta" {
		t.Errorf("FetchGitLabMergeRequests() = %+v, mismatch", mr)
	}
}

func TestFetchPipelineStatus_GitLab(t *testing.T) {
	runner := fakeRunner(t, map[string]string{
		"projects/mecha-consulting%2Fmc-product-lifttraq":                               "testdata/gitlab_project.json",
		"projects/mecha-consulting%2Fmc-product-lifttraq/pipelines?ref=main&per_page=1": "testdata/gitlab_pipelines.json",
		"projects/mecha-consulting%2Fmc-product-lifttraq/pipelines/5501":                "testdata/gitlab_pipeline_detail.json",
	})

	run, err := FetchGitLabPipelineStatus(runner, "mecha-consulting/mc-product-lifttraq")
	if err != nil {
		t.Fatalf("FetchGitLabPipelineStatus() error = %v", err)
	}
	if run.Repo != "mecha-consulting/mc-product-lifttraq" || run.Status != model.PipelinePass || run.Branch != "main" {
		t.Errorf("FetchGitLabPipelineStatus() = %+v, mismatch", run)
	}
	if run.Duration != 108*time.Second {
		t.Errorf("Duration = %v, want 108s", run.Duration)
	}
}

func TestFetchPipelineStatus_GitLab_NoRuns(t *testing.T) {
	runner := fakeRunner(t, map[string]string{
		"projects/mecha-consulting%2Fmc-product-lifttraq":                               "testdata/gitlab_project.json",
		"projects/mecha-consulting%2Fmc-product-lifttraq/pipelines?ref=main&per_page=1": "testdata/gitlab_pipelines_empty.json",
	})

	run, err := FetchGitLabPipelineStatus(runner, "mecha-consulting/mc-product-lifttraq")
	if err != nil {
		t.Fatalf("FetchGitLabPipelineStatus() error = %v", err)
	}
	if run.Status != model.PipelineNone {
		t.Errorf("Status = %v, want %v", run.Status, model.PipelineNone)
	}
}

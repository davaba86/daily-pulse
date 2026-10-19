package mockdata

import "testing"

func TestPullRequests(t *testing.T) {
	prs := PullRequests()
	if len(prs) != 3 {
		t.Fatalf("len(PullRequests()) = %d, want 3", len(prs))
	}
}

func TestPipelineRuns(t *testing.T) {
	runs := PipelineRuns()
	if len(runs) != 2 {
		t.Fatalf("len(PipelineRuns()) = %d, want 2", len(runs))
	}
}

func TestJiraOpenTickets(t *testing.T) {
	tickets := JiraOpenTickets()
	if len(tickets) != 2 {
		t.Fatalf("len(JiraOpenTickets()) = %d, want 2", len(tickets))
	}
}

func TestJiraCompletedTickets(t *testing.T) {
	tickets := JiraCompletedTickets()
	if len(tickets) != 1 {
		t.Fatalf("len(JiraCompletedTickets()) = %d, want 1", len(tickets))
	}
}

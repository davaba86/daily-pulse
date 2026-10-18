package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand_Mock(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--mock"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"GITHUB", "GITLAB", "PIPELINES", "JIRA", "open PRs/MRs"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q, got:\n%s", want, got)
		}
	}
}

func TestRootCommand_MissingConfig(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", "/nonexistent/config.yaml"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want error for missing config")
	}
}

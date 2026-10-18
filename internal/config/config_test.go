package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	contents := `
github:
  username: davaba86
  repos:
    - owner: davaba86
      name: daily-pulse
gitlab:
  username: david.abarca
  projects:
    - path: mecha-consulting/mc-platform-cloud
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.GitHub.Username != "davaba86" {
		t.Errorf("GitHub.Username = %q, want %q", cfg.GitHub.Username, "davaba86")
	}
	if len(cfg.GitHub.Repos) != 1 || cfg.GitHub.Repos[0].Owner != "davaba86" || cfg.GitHub.Repos[0].Name != "daily-pulse" {
		t.Errorf("GitHub.Repos = %+v, want one repo davaba86/daily-pulse", cfg.GitHub.Repos)
	}
	if cfg.GitLab.Username != "david.abarca" {
		t.Errorf("GitLab.Username = %q, want %q", cfg.GitLab.Username, "david.abarca")
	}
	if len(cfg.GitLab.Projects) != 1 || cfg.GitLab.Projects[0].Path != "mecha-consulting/mc-platform-cloud" {
		t.Errorf("GitLab.Projects = %+v, want one project mecha-consulting/mc-platform-cloud", cfg.GitLab.Projects)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Load() error = nil, want error for missing file")
	}
}

func TestLoad_NoReposConfigured(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("github:\n  username: davaba86\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want error when no repos/projects are configured")
	}
}

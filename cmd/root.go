package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/davaba86/daily-pulse/internal/config"
	"github.com/davaba86/daily-pulse/internal/githost"
	"github.com/davaba86/daily-pulse/internal/mockdata"
	"github.com/davaba86/daily-pulse/internal/model"
	"github.com/davaba86/daily-pulse/internal/render"
	"github.com/davaba86/daily-pulse/internal/store"
)

func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "daily-pulse", "config.yaml")
}

func defaultStorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "daily-pulse", "pulse.db")
}

func fetchAll(cfg *config.Config, showAll bool) ([]model.PullRequest, []model.PipelineRun, error) {
	var prs []model.PullRequest
	var pipelines []model.PipelineRun

	for _, repo := range cfg.GitHub.Repos {
		repoPRs, err := githost.FetchGitHubPullRequests(githost.ExecRunner, repo.Owner, repo.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch github PRs for %s/%s: %w", repo.Owner, repo.Name, err)
		}
		for _, pr := range repoPRs {
			if showAll || pr.Author == cfg.GitHub.Username {
				prs = append(prs, pr)
			}
		}

		run, err := githost.FetchGitHubPipelineStatus(githost.ExecRunner, repo.Owner, repo.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch github pipeline for %s/%s: %w", repo.Owner, repo.Name, err)
		}
		pipelines = append(pipelines, run)
	}

	for _, project := range cfg.GitLab.Projects {
		projectMRs, err := githost.FetchGitLabMergeRequests(githost.ExecRunner, project.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch gitlab MRs for %s: %w", project.Path, err)
		}
		for _, mr := range projectMRs {
			if showAll || mr.Author == cfg.GitLab.Username {
				prs = append(prs, mr)
			}
		}

		run, err := githost.FetchGitLabPipelineStatus(githost.ExecRunner, project.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch gitlab pipeline for %s: %w", project.Path, err)
		}
		pipelines = append(pipelines, run)
	}

	return prs, pipelines, nil
}

func NewRootCommand() *cobra.Command {
	var useMock bool
	var showAll bool
	var configPath string

	cmd := &cobra.Command{
		Use:   "daily-pulse",
		Short: "Daily overview of your open PRs, MRs, and pipeline status",
		RunE: func(cmd *cobra.Command, args []string) error {
			now := time.Now()

			if useMock {
				cmd.Print(render.FormatTable(mockdata.PullRequests(), mockdata.PipelineRuns(), now))
				return nil
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			prs, pipelines, err := fetchAll(cfg, showAll)
			if err != nil {
				return err
			}

			dbPath := defaultStorePath()
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return fmt.Errorf("create store dir: %w", err)
			}
			db, err := store.Open(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()
			if _, err := store.SaveSnapshot(db, prs, pipelines, now); err != nil {
				return err
			}

			cmd.Print(render.FormatTable(prs, pipelines, now))
			return nil
		},
	}

	cmd.Flags().BoolVar(&useMock, "mock", false, "use mock data instead of live GitHub/GitLab calls")
	cmd.Flags().BoolVar(&showAll, "all", false, "show PRs/MRs from everyone, not just you")
	cmd.Flags().StringVar(&configPath, "config", defaultConfigPath(), "path to config.yaml")

	return cmd
}

func Execute() {
	cmd := NewRootCommand()
	cobra.CheckErr(cmd.Execute())
}

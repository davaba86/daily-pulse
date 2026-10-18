package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/davaba86/daily-pulse/internal/mockdata"
	"github.com/davaba86/daily-pulse/internal/render"
)

func NewRootCommand() *cobra.Command {
	var useMock bool
	var showAll bool

	cmd := &cobra.Command{
		Use:   "daily-pulse",
		Short: "Daily overview of your open PRs, MRs, and pipeline status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if useMock {
				prs := mockdata.PullRequests()
				pipelines := mockdata.PipelineRuns()
				_ = showAll // author filtering lands in Task 11 once real fetches exist
				cmd.Print(render.FormatTable(prs, pipelines, time.Now()))
				return nil
			}
			return cmd.Help()
		},
	}

	cmd.Flags().BoolVar(&useMock, "mock", false, "use mock data instead of live GitHub/GitLab calls")
	cmd.Flags().BoolVar(&showAll, "all", false, "show PRs/MRs from everyone, not just you")

	return cmd
}

func Execute() {
	cmd := NewRootCommand()
	cobra.CheckErr(cmd.Execute())
}

package main

import (
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/flak-analyzer/pkg/artifacts/loader"
)

var rootCmd = &cobra.Command{
	Use:   "flake-analyzer",
	Short: "Flake analyzer",
	Long: "The flake analyzer downloads JUNIT test report from GITHUB as artifacts and generate report to upload as" +
		" artifacts. It also creates comments for the PR initiating the tests to list out failures",
	RunE: func(cmd *cobra.Command, args []string) error {
		owner := cmd.Flag("owner").Value.String()
		repo := cmd.Flag("repo").Value.String()
		token := cmd.Flag("token").Value.String()

		fromDays := cmd.Flag("from").Value.String()
		fdays, err := strconv.Atoi(fromDays)
		if err != nil {
			return err
		}
		toDays := cmd.Flag("to").Value.String()
		tdays, err := strconv.Atoi(toDays)
		if err != nil {
			return err
		}

		nameFilter := cmd.Flag("test-suite-filter").Value.String()
		commitFilter := cmd.Flag("commit").Value.String()
		reportDir := cmd.Flag("report-dir").Value.String()
		ArtifactDir := cmd.Flag("download-dir").Value.String()

		PRnum := cmd.Flag("pull-request").Value.String()
		waitForQuotaReset := cmd.Flag("wait-for-quota-reset").Value.String()
		waitForReset, err := strconv.ParseBool(waitForQuotaReset)
		if err != nil {
			return err
		}

		report := loader.NewFlakeReport()

		if err := report.LoadReport(loader.RepositoryInfo(owner, repo), loader.WithToken(token),
			loader.FilterFromDaysAgo(fdays), loader.FilterToDaysAgo(tdays),
			loader.FilterTestSuite(nameFilter), loader.FilterCommit(commitFilter),
			loader.WithTempDownloadDir(ArtifactDir), loader.WaitWaitForQuotaReset(waitForReset),
			loader.FilterPR(PRnum)); err != nil {
			return err
		}

		if _, err := report.GenerateReport(reportDir); err != nil {
			return err
		}

		if PRnum != "" {
			_, err := report.PostReportAsPullRequestComment()
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func main() {
	rootCmd.Flags().StringP("owner", "n", "", "The owner of the repository to analyze the flakes.")
	if err := rootCmd.MarkFlagRequired("owner"); err != nil {
		log.Fatalf("Failed to mark `owner` flag for `flake-analyzer` subcommand as required")
	}

	rootCmd.Flags().StringP("repo", "r", "", "The name of the repository for analyze the flakes.")
	if err := rootCmd.MarkFlagRequired("repo"); err != nil {
		log.Fatalf("Failed to mark `repo` flag for `flake-analyzer` subcommand as required")
	}

	rootCmd.Flags().StringP("token", "t", "", "The personal access token for the repository to interact with the stored artifacts")
	if err := rootCmd.MarkFlagRequired("token"); err != nil {
		log.Fatalf("Failed to mark `token` flag for `flake-analyzer` subcommand as required")
	}

	rootCmd.Flags().Uint("from", 90, "Include test results created as artifacts from a number of days ago")
	rootCmd.Flags().Uint("to", 0, "Include test results created as artifacts until a number of days ago")

	rootCmd.Flags().StringP("test-suite-filter", "f", "",
		"Filter test by the test suite name or the common names between the artifacts.")
	rootCmd.Flags().StringP("commit", "c", "",
		"Filter test by the commit SHA or the common names between the artifacts")

	rootCmd.Flags().StringP("report-dir", "o", "./report", "The directory to save the generated report.")
	rootCmd.Flags().StringP("download-dir", "d", "", "The directory to save the downloaded artifacts.")

	rootCmd.Flags().StringP("pull-request", "p", "", "Generate a report for a Pull Request and post as comment.")
	rootCmd.Flags().BoolP("wait-for-quota-reset", "w", false, "Wait for GitHub to reset token limit if quota runs out.")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

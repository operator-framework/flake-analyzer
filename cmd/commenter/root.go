package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/flak-analyzer/pkg/artifacts/commenter"
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

		local_owner := cmd.Flag("local-owner").Value.String()
		local_repo := cmd.Flag("local-repo").Value.String()
		local_token := cmd.Flag("local-token").Value.String()
		if local_token == "" {
			local_token = token
		}

		testNameFilter := cmd.Flag("test-suite-filter").Value.String()
		artifactName := cmd.Flag("artifact-name").Value.String()
		progressFile := cmd.Flag("progress-file-dir").Value.String()

		cf, err := commenter.NewCommenter(local_owner, local_repo, local_token, artifactName, progressFile)
		if err != nil {
			return err
		}
		err = cf.AddRepo(owner, repo, token, testNameFilter)
		if err != nil {
			return err
		}
		comments, err := cf.GenerateComments()
		if err != nil {
			return err
		}
		for _, c := range comments {
			os.Stdout.Write([]byte(*c))
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

	rootCmd.Flags().StringP("local-owner", "m", "operator-framework",
		"The owner of the repository hosting the analyzer.")
	if err := rootCmd.MarkFlagRequired("owner"); err != nil {
		log.Fatalf("Failed to mark `local-owner` flag for `flake-analyzer` subcommand as required")
	}

	rootCmd.Flags().StringP("local-repo", "l", "flake-analyzer", "The name of the repository hosting the analyzer.")
	if err := rootCmd.MarkFlagRequired("repo"); err != nil {
		log.Fatalf("Failed to mark `local-repo` flag for `flake-analyzer` subcommand as required")
	}

	rootCmd.Flags().StringP("local-token", "a", "",
		"The personal access token for the repository hosting the analyzer (default to `token` value).")

	rootCmd.Flags().StringP("test-suite-filter", "f", "",
		"Filter test by the test suite name or the common names between the artifacts.")

	rootCmd.Flags().StringP("progress-file-dir", "p", "", "The directory to save the generated report.")
	rootCmd.Flags().StringP("artifact-name", "i", "flake-bot-progress", "The name of the artifact to save progress.")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

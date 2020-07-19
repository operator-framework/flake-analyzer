package reporter

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/operator-framework/flak-analyzer/pkg/github"
)

var ErrorNothingToReport error = errors.New("no error in test to report")

type HtmlFlakeReport struct {
	TotalTestCount   int             `json:"total_test_count",omitempty`   // All imported test reports have failures
	FlakeTestCount   int             `json:"flake_test_count",omitempty`   // Number of test suit report
	SkippedTestCount int             `json:"skipped_test_count",omitempty` // Number of test suit report
	FlakeTests       []HtmlTestEntry `json:"flake_tests",omitempty`        // Sorted by counts and number of commits
	SkippedTests     []HtmlTestEntry `json:"skipped_tests",omitempty`
}

type HtmlTestEntry struct {
	ClassName       string           `json:"class_name"`
	Name            string           `json:"name"`
	Counts          int              `json:"counts"`
	Details         []HtmlTestDetail `json:"details",omitempty`
	MeanDurationSec float64          `json:"mean_duration_sec"`
}

type HtmlTestDetail struct {
	Count int    `json:"count"`
	Error string `json:"error",omitempty`
}

func (f *FlakeReport) PostReportAsPullRequestComment(option ...filterOption) (*string, error) {
	if len(f.FlakeTests) == 0 && len(f.SkippedTests) == 0 {
		if _, err := f.GenerateReport(""); err != nil {
			return nil, err
		}
	}

	if f.filter.token == "" || f.filter.owner == "" || f.filter.repo == "" {
		return nil, fmt.Errorf("posting comments requires GitHub access token, repository owner and name information")
	}

	report, err := f.generateReportComment()
	if err != nil {
		return nil, err
	}
	if report == nil {
		return nil, ErrorNothingToReport
	}

	ctx := context.Background()
	client := github.NewRepositoryClient(ctx, f.filter.token, f.filter.owner, f.filter.repo, false)
	pr, err := strconv.Atoi(f.filter.pullRequest)
	if err != nil {
		return nil, err
	}
	err = client.PostPRComment(ctx, pr, report)

	return report, err
}

func (f *FlakeReport) generateReportComment() (*string, error) {
	var shortFlakeTests, shortSkippedTests []HtmlTestEntry

	for _, test := range f.FlakeTests {
		shortFlakeTests = append(shortFlakeTests, HtmlTestEntry{
			ClassName: test.ClassName,
			Name:      "**" + test.Name + "**",
			Counts:    test.Counts,
			Details: func() (details []HtmlTestDetail) {
				for _, d := range test.Details {
					details = append(details, HtmlTestDetail{
						Count: d.Count,
						Error: "\n\n" + d.Error.Error(),
					})
				}
				return
			}(),
			MeanDurationSec: test.MeanDurationSec,
		})
	}

	for _, test := range f.SkippedTests {
		shortSkippedTests = append(shortSkippedTests, HtmlTestEntry{
			ClassName: test.ClassName,
			Name:      "**" + test.Name + "**",
			Counts:    test.Counts,
			Details: func() (details []HtmlTestDetail) {
				for _, d := range test.Details {
					details = append(details, HtmlTestDetail{
						Count: d.Count,
						Error: d.Error.Error(),
					})
				}
				return
			}(),
			MeanDurationSec: test.MeanDurationSec,
		})
	}

	if shortFlakeTests == nil && shortSkippedTests == nil {
		return nil, ErrorNothingToReport
	}

	data, err := yaml.Marshal(HtmlFlakeReport{
		TotalTestCount:   f.TotalTestCount,
		FlakeTestCount:   f.FlakeTestCount,
		SkippedTestCount: f.SkippedTestCount,
		FlakeTests:       shortFlakeTests,
		SkippedTests:     shortSkippedTests,
	})
	if err != nil {
		return nil, err
	}

	report := fmt.Sprintf("This PR **failed tests for %d times** with %d individual failed tests and %d skipped tests."+
		" A test is considered flaky if failed on multiple commits. \n<details>\n\n %v",
		f.TotalTestCount, f.FlakeTestCount, f.SkippedTestCount, string(data))
	return &report, nil
}

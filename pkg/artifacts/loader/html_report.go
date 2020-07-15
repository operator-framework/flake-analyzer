package loader

import (
	"context"
	"fmt"
	"github.com/bowenislandsong/flak-analyzer/pkg/github"
	"gopkg.in/yaml.v3"
	"strconv"
)

type HtmlFlakReport struct {
	TotalTestCount   int             `json:"total_test_count",omitempty`   // All imported test reports have failures
	FlakTestCount    int             `json:"flak_test_count",omitempty`    // Number of test suit report
	SkippedTestCount int             `json:"skipped_test_count",omitempty` // Number of test suit report
	FlakTests        []HtmlTestEntry `json:"flak_tests",omitempty`         // Sorted by counts and number of commits
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

func (f *FlakReport) PostReportAsPullRequestComment(option ...filterOption) (*string, error) {
	if f.FlakTests == nil && f.SkippedTests == nil {
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

	ctx := context.Background()
	client := github.NewRepositoryClient(ctx, f.filter.token, f.filter.owner, f.filter.repo, false)
	pr, err := strconv.Atoi(f.filter.pullRequest)
	if err != nil {
		return nil, err
	}
	err = client.PostPRComment(ctx, pr, report)

	return report, err
}

func (f *FlakReport) generateReportComment() (*string, error) {
	var shortFlaksTests, shortSkippedTests []HtmlTestEntry

	for _, test := range f.FlakTests {
		shortFlaksTests = append(shortFlaksTests, HtmlTestEntry{
			ClassName: test.ClassName,
			Name:      test.Name,
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
			Name:      test.Name,
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

	data, err := yaml.Marshal(HtmlFlakReport{
		TotalTestCount:   f.TotalTestCount,
		FlakTestCount:    f.FlakTestCount,
		SkippedTestCount: f.SkippedTestCount,
		FlakTests:        shortFlaksTests,
		SkippedTests:     shortSkippedTests,
	})
	if err != nil {
		return nil, err
	}

	report := fmt.Sprintf("The PR **failed tests for %d times** with %d individual failed tests and %d skipped tests."+
		"\n<details>\n\n %v",
		f.TotalTestCount, f.FlakTestCount, f.SkippedTestCount, string(data))
	return &report, nil
}

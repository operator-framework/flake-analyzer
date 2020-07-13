package loader

import (
	"gopkg.in/yaml.v2"
)

type ShortFlakReport struct {
	TotalTestCount   int              `json:"total_test_count"`     // All imported test reports have failures
	FlakTestCount    int              `json:"flak_test_count"`      // Number of test suit report
	SkippedTestCount int              `json:"skipped_test_count"`   // Number of test suit report
	FlakTests        []ShortTestEntry `json:"flak_tests",omitempty` // Sorted by counts and number of commits
	SkippedTests     []ShortTestEntry `json:"skipped_tests",omitempty`
}

type ShortTestEntry struct {
	ClassName       string            `json:"class_name"`
	Name            string            `json:"name"`
	Counts          int               `json:"counts"`
	Details         []ShortTestDetail `json:"details",omitempty`
	MeanDurationSec float64           `json:"mean_duration_sec"`
}

type ShortTestDetail struct {
	Count int   `json:"count"`
	Error error `json:"error",omitempty`
}

func (f *FlakReport) GenerateShortReport() ([]byte, error) {
	var shortFlaksTests, shortSkippedTests []ShortTestEntry
	for _, test := range f.FlakTests {
		shortFlaksTests = append(shortFlaksTests, ShortTestEntry{
			ClassName: test.ClassName,
			Name:      test.Name,
			Counts:    test.Counts,
			Details: func() (details []ShortTestDetail) {
				for _, d := range test.Details {
					details = append(details, ShortTestDetail{
						Count: d.Count,
						Error: d.Error,
					})
				}
				return
			}(),
			MeanDurationSec: test.MeanDurationSec,
		})
	}

	for _, test := range f.SkippedTests {
		shortSkippedTests = append(shortSkippedTests, ShortTestEntry{
			ClassName: test.ClassName,
			Name:      test.Name,
			Counts:    test.Counts,
			Details: func() (details []ShortTestDetail) {
				for _, d := range test.Details {
					details = append(details, ShortTestDetail{
						Count: d.Count,
						Error: d.Error,
					})
				}
				return
			}(),
			MeanDurationSec: test.MeanDurationSec,
		})
	}

	return yaml.Marshal(ShortFlakReport{
		TotalTestCount:   f.TotalTestCount,
		FlakTestCount:    f.FlakTestCount,
		SkippedTestCount: f.SkippedTestCount,
		FlakTests:        shortFlaksTests,
		SkippedTests:     shortSkippedTests,
	})
}

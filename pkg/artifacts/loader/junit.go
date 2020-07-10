package loader

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"

	"github.com/bowenislandsong/flak-analyzer/pkg/artifacts/download"
)

type FlakReport struct {
	filter           reportFilter
	TotalTestCount   int         `json:"total_test_count"`     // All imported test reports have failures
	FlakTestCount    int         `json:"flak_test_count"`      // Number of test suit report
	SkippedTestCount int         `json:"skipped_test_count"`   // Number of test suit report
	FlakTests        []TestEntry `json:"flak_tests",omitempty` // Sorted by counts and number of commits
	SkippedTests     []TestEntry `json:"skipped_tests",omitempty`
	flakTestMap      testMap     // map[class name + test name]TestEntry
	skippedTestMap   testMap
}

type testMap map[string]TestEntry

type TestEntry struct {
	ClassName       string       `json:"class_name"`
	Name            string       `json:"name"`
	Counts          int          `json:"counts"`
	Details         []TestDetail `json:"details",omitempty`
	Commits         []string     `json:"commits"`
	MeanDurationSec float64      `json:"mean_duration_sec"`
}

type TestDetail struct {
	Count     int    `json:"count"`
	Error     error  `json:"error",omitempty`
	SystemOut string `json:"system_out",omitempty`
	SystemErr string `json:"system_err",omitempty`
}

type reportFilter struct {
	from, to    *time.Time
	owner, repo string
	token       string
	testsuite   string
	commit      string
	localPath   string
	tmpDir      string
}

type filterOption func(filter *reportFilter)

func FilterFrom(from time.Time) filterOption {
	return func(filter *reportFilter) {
		filter.from = &from
	}
}

func FilterTo(to time.Time) filterOption {
	return func(filter *reportFilter) {
		filter.from = &to
	}
}

func RepositoryInfo(owner, name string) filterOption {
	return func(filter *reportFilter) {
		filter.owner = owner
		filter.repo = name
	}
}

func WithToken(token string) filterOption {
	return func(filter *reportFilter) {
		filter.token = token
	}
}

func FilterTestSuite(testsuite string) filterOption {
	return func(filter *reportFilter) {
		filter.testsuite = testsuite
	}
}

func FilterCommit(commit string) filterOption {
	return func(filter *reportFilter) {
		filter.commit = commit
	}
}

func ImportFromLocalDirectory(dir string) filterOption {
	return func(filter *reportFilter) {
		filter.localPath = dir
	}
}

// WithTempDownloadDir specify the directory where artifacts will be temprarily downloaded for use.
func WithTempDownloadDir(dir string) filterOption {
	return func(filter *reportFilter) {
		filter.tmpDir = dir
	}
}

func (r *reportFilter) apply(options []filterOption) {
	for _, option := range options {
		option(r)
	}
}

func (r *reportFilter) complete() error {
	if (r.owner == "" || r.repo == "") && r.localPath == "" {
		return fmt.Errorf("please supply either owner, repository name, and filter info or report artifact directory")
	}

	if r.owner != "" && r.repo != "" && r.token == "" {
		return fmt.Errorf("please supply token for pulling artifacts from Github %s/%s", r.owner, r.repo)
	}

	if r.tmpDir == "" {
		r.tmpDir = "./"
	}

	return nil
}

func NewFlakReport() *FlakReport {
	return &FlakReport{
		filter:         reportFilter{},
		TotalTestCount: 0,
		FlakTestCount:  0,
		FlakTests:      []TestEntry{},
		SkippedTests:   []TestEntry{},
		flakTestMap:    map[string]TestEntry{},
		skippedTestMap: map[string]TestEntry{},
	}
}

func (f *FlakReport) LoadReport(option ...filterOption) error {
	f.filter.apply(option)
	if err := f.filter.complete(); err != nil {
		return err
	}

	if f.filter.owner != "" && f.filter.repo != "" {
		ctx := context.Background()
		tmpDir, err := ioutil.TempDir(f.filter.tmpDir, "artifacts-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		// Download from Github
		client := download.NewRepositoryClient(ctx, f.filter.token, f.filter.owner, f.filter.repo)
		arlist, err := client.ListAllArtifacts(ctx)
		if err != nil {
			return err
		}

		parttern := f.filter.testsuite + "-" + f.filter.commit
		d, err := client.DownloadArtifacts(ctx, arlist, tmpDir, parttern, f.filter.from, f.filter.to)
		if d == nil {
			return fmt.Errorf("no artifact has been downlaoded")
		}
		log.Infof("Downloaded %d artifacts from %s/%s with filter '%s' from %v to %v", len(d), f.filter.owner,
			f.filter.repo, parttern, f.filter.from, f.filter.to)

		// Parse it
		err = f.addTests(tmpDir)
		if err != nil {
			return err
		}
	}

	if f.filter.localPath != "" {
		if err := f.addTests(f.filter.localPath); err != nil {
			return err
		}
	}

	return nil
}

// GenerateReport sorts the report and converts the tests from map to arrays for print out. It generates a yaml report.
func (f *FlakReport) GenerateReport(outputFile string) ([]byte, error) {
	for _, test := range f.flakTestMap {
		f.FlakTests = append(f.FlakTests, test)
	}

	for _, test := range f.skippedTestMap {
		f.SkippedTests = append(f.SkippedTests, test)
	}

	f.FlakTestCount = len(f.FlakTests)

	sort.Slice(f.FlakTests, func(i, j int) bool {
		return f.FlakTests[i].Counts > f.FlakTests[j].Counts && len(f.FlakTests[i].Commits) > len(f.FlakTests[j].Commits)
	})

	f.SkippedTestCount = len(f.SkippedTests)

	sort.Slice(f.SkippedTests, func(i, j int) bool {
		return f.SkippedTests[i].Counts > f.SkippedTests[j].Counts && len(f.SkippedTests[i].Commits) > len(f.SkippedTests[j].Commits)
	})

	data, err := yaml.Marshal(f)
	if err != nil {
		return nil, err
	}

	if outputFile != "" {
		if err := os.MkdirAll(filepath.Dir(outputFile), 0700); err != nil {
			return nil, err
		}
		if err := ioutil.WriteFile(outputFile, data, 0644); err != nil {
			return nil, err
		}
	}

	return data, nil
}

func (f *FlakReport) addTests(dir string) error {
	artifacts, err := LoadZippedArtifactsFromDirectory(dir)
	if err != nil {
		return err
	}
	for _, ar := range artifacts {
		suits, err := ingestTestSuitesFromRawData(ar.rawData)
		if err != nil {
			return err
		}

		for _, s := range suits {
			for _, t := range s.Tests {
				switch t.Status {
				case junit.StatusPassed:
					continue
				case junit.StatusSkipped:
					f.skippedTestMap.loadTestEntries(t, ar.commit)
				default:
					// failed or errored
					f.flakTestMap.loadTestEntries(t, ar.commit)
				}
			}
		}

		f.TotalTestCount = f.TotalTestCount + 1
	}
	return nil
}

func (t *testMap) loadTestEntries(test junit.Test, commit string) {
	testName := test.Classname + "/" + test.Name
	if existing, ok := (*t)[testName]; !ok {
		(*t)[testName] = TestEntry{
			Commits:         []string{commit},
			Counts:          1,
			Name:            test.Name,
			ClassName:       test.Classname,
			MeanDurationSec: test.Duration.Seconds(),
			Details: func() []TestDetail {
				if test.Error == nil && test.SystemOut == "" && test.SystemErr == "" {
					return nil
				}
				return []TestDetail{
					{
						Count:     1,
						Error:     test.Error,
						SystemOut: test.SystemOut,
						SystemErr: test.SystemErr,
					},
				}
			}(),
		}
	} else {

		(*t)[testName] = TestEntry{
			Commits:         append(existing.Commits, commit),
			Counts:          existing.Counts + 1,
			Name:            test.Name,
			ClassName:       test.Classname,
			MeanDurationSec: (test.Duration.Seconds()-existing.MeanDurationSec)/float64(existing.Counts+1) + existing.MeanDurationSec,
			Details: func() []TestDetail {
				if test.Error == nil && test.SystemOut == "" && test.SystemErr == "" {
					return existing.Details
				}
				for i, detail := range existing.Details {
					if detail.SystemErr == test.SystemErr {
						existing.Details[i].Count = detail.Count + 1
						return existing.Details
					}
				}
				return append(existing.Details, TestDetail{
					Count:     1,
					Error:     test.Error,
					SystemOut: test.SystemOut,
					SystemErr: test.SystemErr,
				})
			}(),
		}
	}
}

func ingestTestSuitesFromRawData(rawData ...[]byte) ([]junit.Suite, error) {
	var testSuites []junit.Suite
	for _, raw := range rawData {
		suite, err := junit.Ingest(raw)
		if err != nil {
			return nil, err
		}

		testSuites = append(testSuites, suite...)
	}

	return testSuites, nil
}

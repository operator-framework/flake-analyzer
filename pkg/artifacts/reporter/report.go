package reporter

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"

	gh "github.com/google/go-github/v32/github"
	"github.com/operator-framework/flak-analyzer/pkg/github"
)

type FlakeReport struct {
	filter           reportFilter
	TotalTestCount   int         `json:"total_test_count"`      // All imported test reports have failures
	FlakeTestCount   int         `json:"flake_test_count"`      // Number of test suit report
	SkippedTestCount int         `json:"skipped_test_count"`    // Number of test suit report
	FlakeTests       []TestEntry `json:"flake_tests",omitempty` // Sorted by counts and number of commits
	SkippedTests     []TestEntry `json:"skipped_tests",omitempty`
	flakeTestMap     testMap     // map[class name + test name]TestEntry
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
	from, to          *time.Time
	owner, repo       string
	token             string
	testsuite         string
	commit            string
	pullRequest       string
	localPath         string
	tmpDir            string
	waitForQuotaReset bool
}

type filterOption func(filter *reportFilter)

func FilterFrom(from time.Time) filterOption {
	return func(filter *reportFilter) {
		filter.from = &from
	}
}

func FilterFromDaysAgo(days int) filterOption {
	from := time.Now().AddDate(0, 0, -days)
	return func(filter *reportFilter) {
		filter.from = &from
	}
}

func FilterTo(to time.Time) filterOption {
	return func(filter *reportFilter) {
		filter.to = &to
	}
}

func FilterToDaysAgo(days int) filterOption {
	to := time.Now().AddDate(0, 0, -days)
	return func(filter *reportFilter) {
		filter.to = &to
	}
}

func WaitWaitForQuotaReset(waitForReset bool) filterOption {
	return func(filter *reportFilter) {
		filter.waitForQuotaReset = waitForReset
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

func FilterPR(pr string) filterOption {
	return func(filter *reportFilter) {
		filter.pullRequest = pr
	}
}

func ImportFromLocalDirectory(dir string) filterOption {
	return func(filter *reportFilter) {
		filter.localPath = dir
	}
}

// WithTempDownloadDir specify the directory where artifacts will be temprarily downloaded for use.
func WithTempDownloadDir(tmpDir string) filterOption {
	return func(filter *reportFilter) {
		filter.tmpDir = tmpDir
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

func NewFlakeReport() *FlakeReport {
	return &FlakeReport{
		filter:         reportFilter{},
		TotalTestCount: 0,
		FlakeTestCount: 0,
		FlakeTests:     []TestEntry{},
		SkippedTests:   []TestEntry{},
		flakeTestMap:   map[string]TestEntry{},
		skippedTestMap: map[string]TestEntry{},
	}
}

func (f *FlakeReport) LoadReport(option ...filterOption) error {
	f.filter.apply(option)
	if err := f.filter.complete(); err != nil {
		return err
	}

	if f.filter.owner != "" && f.filter.repo != "" {
		ctx := context.Background()

		// Download from Github
		client := github.NewRepositoryClient(ctx, f.filter.token, f.filter.owner, f.filter.repo, f.filter.waitForQuotaReset)
		arlist, err := client.ListAllArtifacts(ctx)
		if err != nil {
			return err
		}

		if f.filter.pullRequest != "" {
			pr, err := strconv.Atoi(f.filter.pullRequest)
			if err != nil {
				return err
			}
			commits, err := client.ListCommitsFromPR(ctx, pr)
			if err != nil {
				return err
			}
			if f.filter.commit != "" {
				commits = append(commits, f.filter.commit)
			}
			f.filter.commit = strings.Join(commits, "|")
		}

		var pattern string
		if f.filter.testsuite != "" && f.filter.commit != "" {
			pattern = f.filter.testsuite + "-(" + f.filter.commit + ")"
		} else {
			pattern = f.filter.testsuite + f.filter.commit
		}

		chErr := make(chan error)
		var arNames []string
		var wg sync.WaitGroup
		var mutex = &sync.Mutex{}

		for index, artifacts := range splitStringSlice(arlist, 8) {
			wg.Add(1)
			go func(i int, ar []*gh.Artifact) {
				defer wg.Done()
				tmpDir, err := ioutil.TempDir(f.filter.tmpDir, "artifacts-")
				if err != nil {
					chErr <- err
					return
				}
				defer os.RemoveAll(tmpDir)
				d, err := client.DownloadArtifacts(ctx, ar, tmpDir, pattern, f.filter.from, f.filter.to)
				if err != nil {
					chErr <- fmt.Errorf("no artifact has been downlaoded %v", err)
					return
				}
				if d == nil {
					return
				}
				mutex.Lock()
				arNames = append(arNames, d...)
				// Parse it
				err = f.addTests(tmpDir)
				if err != nil {
					chErr <- err
					return
				}
				mutex.Unlock()
			}(index, artifacts)
		}
		wg.Wait()
		select {
		case err := <-chErr:
			return err
		default:
			log.Infof("Downloaded %d artifacts from %s/%s with filter '%s' from %v to %v", len(arNames), f.filter.owner,
				f.filter.repo, pattern, f.filter.from, f.filter.to)
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
func (f *FlakeReport) GenerateReport(outputFile string) ([]byte, error) {
	for _, test := range f.flakeTestMap {
		f.FlakeTests = append(f.FlakeTests, test)
	}

	for _, test := range f.skippedTestMap {
		f.SkippedTests = append(f.SkippedTests, test)
	}

	f.FlakeTestCount = len(f.FlakeTests)

	sort.Slice(f.FlakeTests, func(i, j int) bool {
		return f.FlakeTests[i].Counts > f.FlakeTests[j].Counts && len(f.FlakeTests[i].Commits) > len(f.FlakeTests[j].Commits)
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
		log.Infof("Writing report to %s", outputFile)
		if err := os.MkdirAll(filepath.Dir(outputFile), 0700); err != nil {
			return nil, err
		}
		if err := ioutil.WriteFile(outputFile, data, 0644); err != nil {
			return nil, err
		}
	}

	return data, nil
}

func (f *FlakeReport) addTests(dir string) error {
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
					f.flakeTestMap.loadTestEntries(t, ar.commit)
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

func splitStringSlice(s []*gh.Artifact, n int) (result [][]*gh.Artifact) {
	if len(s) < n {
		return append(result, s)
	}
	num := len(s) / n
	for i := 0; i < n; i++ {
		result = append(result, s[num*i:num*(i+1)])
	}

	for i := 0; i < len(s)%n; i++ {
		if result != nil && result[i] != nil {
			result[i] = append(result[i], s[len(s)-i-1])
		}
	}
	return
}

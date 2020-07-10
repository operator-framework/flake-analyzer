package loader

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"

	"github.com/bowenislandsong/flak-analyzer/pkg/artifacts/download"
)

type FlakReport struct {
	filter         reportFilter
	TotalCount     int         // All imported test reports have failures
	FlakTestCount  int         // Number of test suit report
	FlakTests      []TestEntry // Sorted by counts and number of commits
	SkippedTests   []TestEntry
	flakTestMap    testMap // map[class name + test name]TestEntry
	skippedTestMap testMap
}

type testMap map[string]TestEntry

type TestEntry struct {
	Commits         []string
	Counts          int
	Name            string
	ClassName       string
	MeanDurationSec float64
	Details         []TestDetail
}

type TestDetail struct {
	Count     int
	Reason    string
	Error     error
	SystemOut string
	SystemErr string
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

func RepoOwner(owner string) filterOption {
	return func(filter *reportFilter) {
		filter.owner = owner
	}
}

func RepoName(name string) filterOption {
	return func(filter *reportFilter) {
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

func NewFlakReport() (*FlakReport, error) {
	return &FlakReport{
		filter:         reportFilter{},
		TotalCount:     0,
		FlakTestCount:  0,
		FlakTests:      []TestEntry{},
		SkippedTests:   []TestEntry{},
		flakTestMap:    map[string]TestEntry{},
		skippedTestMap: map[string]TestEntry{},
	}, nil
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
		parttern := "*" + f.filter.testsuite + "-" + f.filter.commit + "*"
		d, err := client.DownloadArtifacts(ctx, arlist, tmpDir, parttern,
			f.filter.from, f.filter.to)
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
func (f *FlakReport) GenerateReport()error{

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

		f.TotalCount = f.TotalCount + 1
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
			Details: []TestDetail{
				{
					Count:     1,
					Error:     test.Error,
					SystemOut: test.SystemOut,
					SystemErr: test.SystemErr,
				},
			},
		}
	} else {
		(*t)[testName] = TestEntry{
			Commits:         []string{commit},
			Counts:          existing.Counts + 1,
			Name:            test.Name,
			ClassName:       test.Classname,
			MeanDurationSec: (test.Duration.Seconds()-existing.MeanDurationSec)/float64(existing.Counts+1) + existing.MeanDurationSec,
			Details: []TestDetail{
				{
					Count:     1,
					Error:     test.Error,
					SystemOut: test.SystemOut,
					SystemErr: test.SystemErr,
				},
			},
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

package commenter

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v32/github"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/operator-framework/flak-analyzer/pkg/artifacts/reporter"
	fgithub "github.com/operator-framework/flak-analyzer/pkg/github"
)

type CommenterFile struct {
	client       *fgithub.RepositoryClient
	owner        string
	repo         string
	artifactName string
	progressFile string
	Commented    []*Commenter `json:"commented"`
}

type Commenter struct {
	client          *fgithub.RepositoryClient
	token           string
	Owner           string              `json:"owner"`
	Repo            string              `json:"repo"`
	TestNameMatcher string              `json:"test_name_matcher"`
	RunIDs          map[string]struct{} `json:"run_id"`
}

func NewCommenter(commenterOwner, commenterRepo, commenterToken, artifactName, progressFile string) (*CommenterFile, error) {
	ctx := context.Background()
	f := &CommenterFile{
		client:       fgithub.NewRepositoryClient(ctx, commenterToken, commenterOwner, commenterRepo, false),
		owner:        commenterOwner,
		repo:         commenterRepo,
		artifactName: artifactName,
		progressFile: progressFile,
		Commented:    []*Commenter{},
	}

	if err := f.getLatestCommenterFile(ctx); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *CommenterFile) AddRepo(owner, repo, token, testNameMatcher string) error {
	if owner == "" || repo == "" || token == "" {
		return fmt.Errorf("commenting requires Owner, Repo, and Token to be not empty")
	}
	ctx := context.Background()

	for i, entry := range f.Commented {
		if entry.Owner == owner && entry.Repo == repo && entry.TestNameMatcher == testNameMatcher {
			f.Commented[i].client = fgithub.NewRepositoryClient(ctx, token, owner, repo, false)
			f.Commented[i].token = token
			return nil
		}
	}
	f.Commented = append(f.Commented, &Commenter{
		client:          fgithub.NewRepositoryClient(ctx, token, owner, repo, false),
		token:           token,
		Owner:           owner,
		Repo:            repo,
		TestNameMatcher: testNameMatcher,
		RunIDs:          map[string]struct{}{},
	})
	return nil
}

func (f *CommenterFile) GenerateComments() ([]*string, error) {
	ctx := context.Background()
	var comments []*string
	for _, c := range f.Commented {
		prcs, err := c.generatePRComments(ctx)
		if err != nil {
			return nil, err
		}
		for _, prc := range prcs {
			report := reporter.NewFlakeReport()
			if err = report.LoadReport(reporter.RepositoryInfo(c.Owner, c.Repo), reporter.WithToken(c.token),
				reporter.FilterPR(strconv.Itoa(prc.pr)),
				reporter.FilterTestSuite(c.TestNameMatcher)); err != nil {
				return nil, err
			}
			comment, err := report.PostReportAsPullRequestComment()
			if err != nil {
				if err == reporter.ErrorNothingToReport {
					continue
				}
				return nil, err
			}

			comments = append(comments, comment)
		}
	}

	if err := f.saveCommneterFile(); err != nil {
		return nil, err
	}

	return comments, nil
}

func (f *CommenterFile) saveCommneterFile() error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}

	if f.progressFile == "" {
		f.progressFile = "./commenter_progress_file.yaml"
	}
	if err := os.MkdirAll(filepath.Dir(f.progressFile), 0700); err != nil {
		return err
	}
	if err := ioutil.WriteFile(f.progressFile, data, 0644); err != nil {
		return err
	}
	logrus.Infof("Commenter progress file saved as %s", f.progressFile)
	return nil
}

func (f *CommenterFile) getLatestCommenterFile(ctx context.Context) error {
	commenterArtifacts, err := f.client.ListAllArtifacts(ctx)
	if err != nil {
		return err
	}

	if commenterArtifacts != nil {
		if len(commenterArtifacts) > 1 {
			sort.Slice(commenterArtifacts, func(i, j int) bool {
				return commenterArtifacts[i].GetCreatedAt().Time.After(commenterArtifacts[j].GetCreatedAt().Time)
			})
		}

		for _, ar := range commenterArtifacts {
			if ar.GetName() != f.artifactName {
				continue
			}
			dir, err := ioutil.TempDir("./", "commenter-progress-")
			if err != nil {
				return err
			}

			if _, err = f.client.DownloadArtifacts(ctx, []*github.Artifact{ar}, dir, "", nil, nil); err != nil {
				return err
			}
			content, err := reporter.Unzip(filepath.Join(dir, ar.GetName()+".zip"))
			if err != nil {
				return err
			}
			if err = yaml.Unmarshal(content, f); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

//get all open PR in repo
// get runID list
// get all artifacts in REPO
// get all runID from PR,commits
// filter runIDs
// write report for PR
// upload new runID list

type pullRequest struct {
	pr      int
	commits []string
	runID   []string
}

func (c *Commenter) generatePRComments(ctx context.Context) ([]pullRequest, error) {
	PRs, err := c.listPRs(ctx)
	if err != nil {
		return nil, err
	}
	artifacts, err := c.client.ListAllArtifacts(ctx)
	if err != nil {
		return nil, err
	}

	commitRunIDsMap := map[string][]string{}
	for _, ar := range artifacts {
		matchTestName, err := regexp.MatchString(c.TestNameMatcher, ar.GetName())
		if err != nil {
			return nil, err
		}
		if ar.GetExpired() || !matchTestName {
			continue
		}
		splits := strings.Split(ar.GetName(), "-")
		if len(splits) < 2 {
			continue
		}
		commitNum := splits[len(splits)-2]
		runID := splits[len(splits)-1]
		if runs, ok := commitRunIDsMap[commitNum]; !ok {
			commitRunIDsMap[commitNum] = []string{runID}
		} else {
			commitRunIDsMap[commitNum] = append(runs, runID)
		}
	}

	var pullRequests []pullRequest
	 updatedRunIDs := map[string]struct{}{}
	for _, pr := range PRs {
		commitNums, err := c.client.ListCommitsFromPR(ctx, pr.GetNumber())
		if err != nil {
			return nil, err
		}
		var runIDs []string
		for _, cNum := range commitNums {
			runIDs = append(runIDs, commitRunIDsMap[cNum]...)
		}

		for _, id := range runIDs {
			updatedRunIDs[id] = struct{}{}
		}

		if c.hasAllRunIDs(runIDs) {
			continue
		}

		pullRequests = append(pullRequests, pullRequest{
			pr:      pr.GetNumber(),
			commits: commitNums,
			runID:   runIDs,
		})
	}
	c.RunIDs = updatedRunIDs
	return pullRequests, nil

}

// hasAllRunIDs checks if commenter listed all the runIDs there are. It returns true is runIDs is nil]
func (c *Commenter) hasAllRunIDs(runIDs []string) bool {
	for _, id := range runIDs {
		if _, ok := c.RunIDs[id]; !ok {
			return false
		}
	}
	return true
}

func (c *Commenter) listPRs(ctx context.Context) ([]*github.PullRequest, error) {
	done := false
	page := 0
	var prList []*github.PullRequest
	for !done {
		prs, resp, err := c.client.PullRequests.List(ctx, c.Owner, c.Repo, &github.PullRequestListOptions{
			State:       "open",
			Sort:        "updated",
			ListOptions: github.ListOptions{Page: page, PerPage: 1000},
		})
		if err != nil {
			return nil, err
		}
		prList = append(prList, prs...)
		if page = resp.NextPage; page == 0 {
			done = true
		}
	}
	return prList, nil
}

package github

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/google/go-github/v32/github"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

func (r *RepositoryClient) ListAllArtifacts(ctx context.Context) ([]*github.Artifact, error) {
	var artifactList []*github.Artifact
	done := false
	page := 0
	for !done {
		list, resp, err := r.Actions.ListArtifacts(ctx, r.Owner, r.Repo, &github.ListOptions{Page: page, PerPage: 1000})
		if err != nil {
			return nil, err
		}
		if 0 == resp.NextPage || resp.Rate.Remaining <= len(artifactList)+2 {
			done = true
		}
		page = resp.NextPage
		artifactList = append(artifactList, list.Artifacts...)
	}
	return artifactList, nil
}

// DownloadArtifacts tries to download artifacts from a artifact list to a directory based on name and time filtering.
// The client is required to have authentication.
// The function returns a list of successfully downloaded artifacts and error.
// Github API related errors are aggrgated and do not stop its following operations due to possible throttling.
func (r *RepositoryClient) DownloadArtifacts(ctx context.Context, artifactList []*github.Artifact, dir,
	namePattern string, after, before *time.Time) ([]string, error) {

	if _, err := ioutil.ReadDir(dir); err != nil {
		if err := os.MkdirAll(dir, 0644); err != nil {
			return nil, err
		}
	}

	var errs []error
	var artifacts []string

	for _, l := range artifactList {
		// Artifacts expires after 90 days.
		if l.GetExpired() {
			continue
		}

		// Filter downloads by creation date.
		if before != nil && !l.GetCreatedAt().Time.Before(*before) {
			continue
		}
		if after != nil && !l.GetCreatedAt().Time.After(*after) {
			continue
		}

		// Filter downloads by name.
		if namePattern != "" {
			matched, err := regexp.MatchString(namePattern, l.GetName())
			if err != nil {
				return nil, fmt.Errorf("error matching artifact name, %v", err)
			}
			if !matched {
				continue
			}
		}

		err := r.downloadArtifact(ctx, l.GetID(), l.GetName(), dir)
		if err != nil {
			errs = append(errs, err)
		} else {
			artifacts = append(artifacts, l.GetName())
		}

	}
	return artifacts, utilerrors.NewAggregate(errs)
}

func (r *RepositoryClient) downloadArtifact(ctx context.Context, artifactID int64, name, dir string) error {
	url, res, err := r.Actions.DownloadArtifact(ctx, r.Owner, r.Repo, artifactID, false)
	if err != nil {
		return err
	}

	if r.WaitForQuotaReset {
		waitForQuota(res)
	}

	resp, err := http.Get(url.String())
	if err != nil {
		return err
	}

	out, err := os.Create(fmt.Sprintf("%s/%s.zip", path.Clean(dir), name))
	if err != nil {
		return fmt.Errorf("failed to create zip file at %s, %v", dir, err)
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func waitForQuota(response *github.Response) {
	if response.Rate.Remaining == 0 {
		logrus.Infof("Waiting for GitHub quota reset: %s",response.Rate.String())
		time.Sleep(response.Rate.Reset.Sub(time.Now()))
		// Waiting for an extra 5 seconds.
		time.Sleep(5 * time.Second)
	}
}

package download

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

type repositoryClient struct {
	*github.Client
	Owner string
	Repo  string
}

func NewRepositoryClient(ctx context.Context, accessToken, owner, repo string) *repositoryClient {
	if accessToken == "" {
		return &repositoryClient{
			Client: github.NewClient(nil),
			Owner:  owner,
			Repo:   repo,
		}
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &repositoryClient{
		Client: github.NewClient(tc),
		Owner:  owner,
		Repo:   repo,
	}
}

func (r *repositoryClient) ListAllArtifacts(ctx context.Context) (list *github.ArtifactList, err error) {
	list, _, err = r.Actions.ListArtifacts(ctx, r.Owner, r.Repo, &github.ListOptions{})
	return
}

// DownloadArtifacts tries to download artifacts from a artifact list to a directory based on name and time filtering.
// The client is required to have authentication.
// The function returns a list of successfully downloaded artifacts and error.
// Github API related errors are aggrgated and do not stop its following operations due to possible throttling.
func (r *repositoryClient) DownloadArtifacts(ctx context.Context, artifactList *github.ArtifactList, dir,
	namePattern string, after, before *time.Time) ([]string, error) {

	if _, err := ioutil.ReadDir(dir); err != nil {
		if err := os.MkdirAll(dir, 0644); err != nil {
			return nil, err
		}
	}

	var errs []error
	var artifacts []string

	for _, l := range artifactList.Artifacts {
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

func (r *repositoryClient) downloadArtifact(ctx context.Context, artifactID int64, name, dir string) error {
	url, _, err := r.Actions.DownloadArtifact(ctx, r.Owner, r.Repo, artifactID, false)
	if err != nil {
		return err
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

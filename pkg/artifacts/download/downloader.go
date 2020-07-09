package download

import (
	"context"
	"fmt"
	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
	"io"
	"io/ioutil"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"os"
	"path"
)

type artifactClient struct {
	*github.Client
	Owner string
	Repo  string
}

func NewArtifactClient(ctx context.Context, accessToken, owner, repo string) *artifactClient {
	if accessToken == "" {
		return &artifactClient{
			Client: github.NewClient(nil),
			Owner:  owner,
			Repo:   repo,
		}
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &artifactClient{
		Client: github.NewClient(tc),
		Owner:  owner,
		Repo:   repo,
	}
}

func (d *artifactClient) ListAllArtifacts(ctx context.Context) (list *github.ArtifactList, err error) {
	list, _, err = d.Actions.ListArtifacts(ctx, d.Owner, d.Repo, &github.ListOptions{})
	return
}

func (d *artifactClient) DownloadAllArtifacts(ctx context.Context, dir string) error {
	if _, err := ioutil.ReadDir(dir); err != nil {
		if err := os.MkdirAll(dir, 0644); err != nil {
			return err
		}
	}

	list, err := d.ListAllArtifacts(ctx)
	if err != nil {
		return err
	}

	var errs []error

	for _, l := range list.Artifacts {
		err = d.downloadArtifacts(ctx, *l.ID, *l.Name, dir)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func (d *artifactClient) downloadArtifacts(ctx context.Context, artifactID int64, name, dir string) error {
	_, resp, err := d.Actions.DownloadArtifact(ctx, d.Owner, d.Repo, artifactID, false)
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

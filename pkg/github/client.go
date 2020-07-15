package github

import (
	"context"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

type RepositoryClient struct {
	*github.Client
	Owner             string
	Repo              string
	WaitForQuotaReset bool
}

func NewRepositoryClient(ctx context.Context, accessToken, owner, repo string, waitForQuotaReset bool) *RepositoryClient {
	if accessToken == "" {
		return &RepositoryClient{
			Client:            github.NewClient(nil),
			Owner:             owner,
			Repo:              repo,
			WaitForQuotaReset: waitForQuotaReset,
		}
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &RepositoryClient{
		Client: github.NewClient(tc),
		Owner:  owner,
		Repo:   repo,
	}
}

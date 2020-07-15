package github

import (
	"context"

	"github.com/google/go-github/v32/github"
)

func (c *RepositoryClient) PostPRComment(ctx context.Context, pullNum int, data *string) error {
	_, _, err := c.Issues.CreateComment(ctx, c.Owner, c.Repo, pullNum, &github.IssueComment{
		Body: data,
	})
	return err
}

func (c *RepositoryClient) ListPRComments(ctx context.Context, pullNum int) ([]*github.IssueComment, error) {
	list, _, err := c.Issues.ListComments(ctx, c.Owner, c.Repo, pullNum, &github.IssueListCommentsOptions{})
	return list, err
}

func (c *RepositoryClient) ListCommitsFromPR(ctx context.Context, pullNum int) ([]string, error) {
	done := false
	var commits []string
	page := 0
	for !done {
		list, resp, err := c.PullRequests.ListCommits(ctx, c.Owner, c.Repo, pullNum,
			&github.ListOptions{Page: page, PerPage: 100})
		if err != nil {
			return nil, err
		}
		for _,l:= range list {
			commits = append(commits, l.GetSHA())
		}
		if page = resp.NextPage; page == 0 {
			done = true
		}

	}
	return commits, nil
}

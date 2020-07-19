package github

import (
	"context"

	"github.com/google/go-github/v32/github"
)


func (r *RepositoryClient) PostPRComment(ctx context.Context, pullNum int, data *string) error {
	_, _, err := r.Issues.CreateComment(ctx, r.Owner, r.Repo, pullNum, &github.IssueComment{
		Body: data,
	})
	return err
}

func (r *RepositoryClient) ListPRComments(ctx context.Context, pullNum int) ([]*github.IssueComment, error) {
	list, _, err := r.Issues.ListComments(ctx, r.Owner, r.Repo, pullNum, &github.IssueListCommentsOptions{})
	return list, err
}

func (r *RepositoryClient) ListCommitsFromPR(ctx context.Context, pullNum int) ([]string, error) {
	done := false
	page := 0
	var commits []string
	for !done {
		list, resp, err := r.PullRequests.ListCommits(ctx, r.Owner, r.Repo, pullNum,
			&github.ListOptions{Page: page, PerPage: 100})
		if err != nil {
			return nil, err
		}
		for _, l := range list {
			commits = append(commits, l.GetSHA())
		}
		if page = resp.NextPage; page == 0 {
			done = true
		}

	}
	return commits, nil
}

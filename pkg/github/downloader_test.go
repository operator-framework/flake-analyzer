package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	owner string = "operator-framework"
	repo  string = "operator-lifecycle-manager"
)

func TestListAllArtifacts(t *testing.T) {
	ctx := context.Background()

	c := NewRepositoryClient(ctx,"", owner, repo,true)
	list, err := c.ListAllArtifacts(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, list)

	l,err:=c.ListCommitsFromPR(ctx,1641)
	assert.NoError(t, err)
	assert.NotEmpty(t, l)
}

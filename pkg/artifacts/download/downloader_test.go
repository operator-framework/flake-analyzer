package download

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	owner string = "operator-framework"
	repo  string = "Operator-lifecycle-manager"
)

func TestListAllArtifacts(t *testing.T) {
	ctx := context.Background()

	client := NewRepositoryClient(ctx, "", owner, repo)
	list, err := client.ListAllArtifacts(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, list)
}

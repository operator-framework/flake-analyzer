package download

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	owner string = "operator-framework"
	repo  string = "Operator-lifecycle-manager"
)

func TestListAllArtifacts(t *testing.T) {
	ctx := context.Background()

	client := NewArtifactClient(ctx, "", owner, repo)
	list, err := client.ListAllArtifacts(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, list)
}

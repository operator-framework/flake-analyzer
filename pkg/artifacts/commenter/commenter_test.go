package commenter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	commenterRepo = "flake-analyzer"
	token         = ""
	owner         = "operator-framework"
	repo          = "operator-lifecycle-manager"
	testName      = "e2e-test-output"
)

func TestNewCommenter(t *testing.T) {
	cf, err := NewCommenter(owner, commenterRepo, token, "flake-bot-operator-fw-artifact", "")
	require.NoError(t, err)
	err = cf.AddRepo(owner, repo, token, testName)
	require.NoError(t, err)
	comments, err := cf.GenerateComments()
	require.NoError(t, err)
	for _, c := range comments {
		fmt.Println(*c)
	}
}

package loader

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLoadArtifactsFromDirectory(t *testing.T) {
	zipLocation := "./testData/zip/"
	ar, err := LoadZippedArtifactsFromDirectory(zipLocation)
	require.NoError(t, err)
	assert.NotEmpty(t, ar)
}

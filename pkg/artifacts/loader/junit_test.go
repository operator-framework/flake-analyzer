package loader

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIngestTestSuitesFromRawData(t *testing.T) {
	zipLocation := "./testData/zip/"
	artifacts, err := LoadZippedArtifactsFromDirectory(zipLocation)
	require.NoError(t, err)

	for _, ar := range artifacts {
		suite, err := ingestTestSuitesFromRawData(ar.rawData)
		assert.NoError(t, err)
		assert.NotEmpty(t, suite)
	}
}

func TestNewFlakReport(t *testing.T) {
	NewFlakReport()
}

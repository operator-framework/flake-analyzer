package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	owner string = "operator-framework"
	repo  string = "Operator-lifecycle-manager"
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
	report := NewFlakReport()
	err := report.LoadReport(ImportFromLocalDirectory("./testData/zip/"))
	assert.NoError(t, err)

	data, err := report.GenerateReport("./tmp/tmp.yaml")
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestGeneratingFlakReportFromOnline(t *testing.T) {
	report := NewFlakReport()
	err := report.LoadReport(RepositoryInfo(owner, repo))
	assert.NoError(t, err)

	data, err := report.GenerateReport("./online/report.yaml")
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

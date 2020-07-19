package reporter

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

func TestGeneratingNewFlakeReport(t *testing.T) {
	report := NewFlakeReport()
	err := report.LoadReport(ImportFromLocalDirectory("./testData/zip/"))
	assert.NoError(t, err)

	data, err := report.GenerateReport("")
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestGeneratingFlakeReportFromOnline(t *testing.T) {
	report := NewFlakeReport()
	err := report.LoadReport(RepositoryInfo(owner, repo),WithToken(""),FilterFromDaysAgo(3),
		FilterCommit("08c4c923c66c9d78895847c5b7e1a21c8887d89c|f5f69155b5c13b94ec56c42dc5e2ffc4236f543b"))
	assert.NoError(t, err)

	data, err := report.GenerateReport("./online/report.yaml")
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	_, err = report.PostReportAsPullRequestComment()
	assert.NoError(t, err)
}

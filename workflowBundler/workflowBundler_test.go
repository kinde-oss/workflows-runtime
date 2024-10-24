package builder

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_WorkflowBundler(t *testing.T) {

	workflowPath, _ := filepath.Abs("../testData/kindeSrc/environment/workflows/evTest")

	workflowBuilder := NewWorkflowBundler(BundlerOptions{
		WorkingFolder: workflowPath,
		EntryPoints:   []string{"tokensWorkflow.ts"},
	})
	bundlerResult := workflowBuilder.Bundle()

	assert := assert.New(t)
	assert.Nil(bundlerResult.Errors, "errors were not expected")
	assert.NotEmpty(bundlerResult.Content.Source)
	assert.Equal("tokenGen", bundlerResult.Content.Settings.ID)
	assert.Equal("onTokenGeneration", bundlerResult.Content.Settings.Other["trigger"])
	assert.Equal("stop", string(bundlerResult.Content.Settings.FailurePolicy.Action))
	assert.NotEmpty(bundlerResult.Content.BundleHash)
}

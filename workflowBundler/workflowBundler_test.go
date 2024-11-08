package builder

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/stretchr/testify/assert"
)

func Test_WorkflowBundler(t *testing.T) {

	workflowPath, _ := filepath.Abs("../testData/kindeSrc/environment/workflows/evTest")

	pluginSetupWasCalled := false

	workflowBuilder := NewWorkflowBundler(BundlerOptions{
		WorkingFolder: workflowPath,
		EntryPoints:   []string{"tokensWorkflow.ts"},
	})
	ctx := WithBundlerPlugins(context.Background(), []api.Plugin{
		{
			Name: "tokenGen",
			Setup: func(build api.PluginBuild) {
				pluginSetupWasCalled = true
			},
		},
	})
	bundlerResult := workflowBuilder.Bundle(ctx)

	assert := assert.New(t)
	assert.True(pluginSetupWasCalled, "plugin setup was not called")
	assert.Nil(bundlerResult.Errors, "errors were not expected")
	assert.NotEmpty(bundlerResult.Content.Source)
	assert.Equal("tokenGen", bundlerResult.Content.Settings.ID)
	assert.Equal("onTokenGeneration", bundlerResult.Content.Settings.Other["trigger"])
	assert.NotEmpty(bundlerResult.Content.BundleHash)
}

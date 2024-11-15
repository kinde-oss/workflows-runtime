package builder

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/stretchr/testify/assert"
)

func Test_WorkflowBundler(t *testing.T) {

	type workflowSettings struct {
		ID      string `json:"id"`
		Trigger string `json:"trigger"`
	}

	workflowPath, _ := filepath.Abs("../testData/kindeSrc/environment/workflows/evTest")

	pluginSetupWasCalled := false

	workflowBuilder := NewWorkflowBundler[workflowSettings](BundlerOptions[workflowSettings]{
		WorkingFolder:       workflowPath,
		EntryPoints:         []string{"tokensWorkflow.ts"},
		IntrospectionExport: "workflowSettings",
		OnDiscovered: func(bundle *BundlerResult[workflowSettings]) {
			bundle.Errors = append(bundle.Errors, "ID is required")
		},
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
	assert.Equal(bundlerResult.Errors[0], "ID is required")
	assert.NotEmpty(bundlerResult.Content.Source)
	assert.Equal("tokenGen", bundlerResult.Content.Settings.Other.ID)
	assert.Equal("onTokenGeneration", bundlerResult.Content.Settings.Other.Trigger)
	assert.NotEmpty(bundlerResult.Content.BundleHash)
}

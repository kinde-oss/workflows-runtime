package project_bundler

import (
	"context"
	"path/filepath"
	"testing"

	bundler "github.com/kinde-oss/workflows-runtime/workflowBundler"
	"github.com/stretchr/testify/assert"
)

func Test_ProjectBunler(t *testing.T) {

	assert := assert.New(t)

	somePathInsideProject, _ := filepath.Abs("../testData/kindeSrc/environment") //starting in a middle of nowhere, so we need to go up to the root of the project

	type workflowSettings struct {
		ID string `json:"id"`
	}

	type pageSettings struct {
	}

	onWorkflowDiscoveredCalled := 0
	onPageDiscoveredCalled := 0
	projectBundler := NewProjectBundler(DiscoveryOptions[workflowSettings, pageSettings]{
		StartFolder: somePathInsideProject,
		OnWorkflowDiscovered: func(ctx context.Context, bundle *bundler.BundlerResult[workflowSettings]) {
			onWorkflowDiscoveredCalled++
			settings := ctx.Value(ProjectSettingsContextKey)
			assert.NotNil(settings)
		},
		OnPageDiscovered: func(ctx context.Context, bundle *bundler.BundlerResult[pageSettings]) {
			onPageDiscoveredCalled++
			settings := ctx.Value(ProjectSettingsContextKey)
			assert.NotNil(settings)
		},
	})

	kindeProject, discoveryError := projectBundler.Discover(context.Background())

	if !assert.Nil(discoveryError) {
		t.FailNow()
	}

	assert.Equal(3, onWorkflowDiscoveredCalled)
	assert.Equal(2, onPageDiscoveredCalled)

	assert.Equal("2024-12-09", kindeProject.Configuration.Version)
	assert.Equal("kindeSrc", kindeProject.Configuration.RootDir)
	assert.Equal(3, len(kindeProject.Environment.Workflows))
	assert.Empty(kindeProject.Environment.Workflows[0].Bundle.Errors)
	assert.Empty(kindeProject.Environment.Workflows[1].Bundle.Errors)

	assert.Equal(2, len(kindeProject.Environment.Pages))
	assert.Empty(kindeProject.Environment.Pages[0].Bundle.Errors)
	assert.Empty(kindeProject.Environment.Pages[1].Bundle.Errors)

}

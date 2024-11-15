package project_bundler

import (
	"context"
	"path/filepath"
	"testing"

	bundler "github.com/kinde-oss/workflows-runtime/workflowBundler"
	"github.com/stretchr/testify/assert"
)

func Test_ProjectBunler(t *testing.T) {
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
		OnWorkflowDiscovered: func(bundle *bundler.BundlerResult[workflowSettings]) {
			onWorkflowDiscoveredCalled++
		},
		OnPageDiscovered: func(bundle *bundler.BundlerResult[pageSettings]) {
			onPageDiscoveredCalled++
		},
	})

	kindeProject, discoveryError := projectBundler.Discover(context.Background())

	assert := assert.New(t)

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

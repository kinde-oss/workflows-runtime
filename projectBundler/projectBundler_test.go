package project_bundler

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ProjectBunler(t *testing.T) {
	somePathInsideProject, _ := filepath.Abs("../testData/kindeSrc/environment/workflows") //starting in a middle of nowhere, so we need to go up to the root of the project

	projectBundler := NewProjectBundler(DiscoveryOptions{
		StartFolder: somePathInsideProject,
	})

	kindeProject, discoveryError := projectBundler.Discover()

	assert := assert.New(t)

	assert.Nil(discoveryError)
	assert.Equal("2024-12-09", kindeProject.Configuration.Version)
	assert.Equal("kindeSrc", kindeProject.Configuration.RootDir)

}

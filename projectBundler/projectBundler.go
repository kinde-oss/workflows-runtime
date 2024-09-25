package project_bundler

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

type (
	ProjectConfiguration struct {
		Version     string `json:"version"`
		RootDir     string `json:"rootDir"`
		AbsLocation string `json:"location"`
	}

	KindeWorkflow struct {
		WorkflowRootDirectory string `json:"workflow_root_directory"`
	}
	KindeWorkflows struct {
		Workflows []KindeWorkflow `json:"workflows"`
	}

	KindeEnvironment struct {
		Workflows KindeWorkflows `json:"workflows"`
	}

	KindeProject struct {
		Configuration ProjectConfiguration `json:"configuration"`
		Environment   KindeEnvironment     `json:"environment"`
	}

	DiscoveryOptions struct {
		StartFolder string
	}

	ProjectBundler interface {
		Discover() (*KindeProject, error)
	}

	projectBundler struct {
		options DiscoveryOptions
	}
)

// Discover implements ProjectBundler.
func (p *projectBundler) Discover() (*KindeProject, error) {
	result := &KindeProject{}
	configuration, err := discoverKindeRoot(p.options.StartFolder)
	if err != nil {
		return nil, fmt.Errorf("error discoving project root: %w", err)
	}
	result.Configuration = *configuration
	return result, nil
}

func discoverKindeRoot(startFolder string) (*ProjectConfiguration, error) {

	currentDirectory, _ := filepath.Abs(startFolder)

	fileInfo, _ := os.Stat(currentDirectory)
	if !fileInfo.IsDir() {
		currentDirectory = filepath.Dir(currentDirectory)
	}
	for {
		filePath := filepath.Join(currentDirectory, "kinde.json")
		configFile, _ := os.Stat(filePath)
		if configFile != nil {
			return readProjectConfiguration(filePath)
		}
		parentPath := filepath.Join(currentDirectory, "..")
		currentDirectory, _ = filepath.Abs(parentPath)
		if currentDirectory == "" {
			return nil, fmt.Errorf("could not find kinde.json")
		}
	}

}

func NewProjectBundler(discoveryOptions DiscoveryOptions) ProjectBundler {
	return &projectBundler{
		options: discoveryOptions,
	}
}

func readProjectConfiguration(configFileInfo string) (*ProjectConfiguration, error) {
	confiHandler, _ := os.Open(configFileInfo)
	configFile, err := io.ReadAll(confiHandler)
	if err != nil {
		return nil, fmt.Errorf("error reading project configuration: %w", err)
	}
	result := &ProjectConfiguration{}
	json.Unmarshal(configFile, result)
	result.AbsLocation = path.Dir(configFileInfo)
	return result, nil
}

package project_bundler

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	bundler "github.com/kinde-oss/workflows-runtime/workflowBundler"
	"github.com/rs/zerolog/log"
)

type (
	ProjectConfiguration struct {
		Version     string `json:"version"`
		RootDir     string `json:"rootDir"`
		AbsLocation string `json:"location"`
	}

	KindeWorkflow struct {
		WorkflowRootDirectory string                `json:"workflow_root_directory"`
		EntryPoints           []string              `json:"entry_points"`
		Bundle                bundler.BundlerResult `json:"bundle"`
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

func (kw *KindeWorkflows) discover(absLocation string) {
	//environment/workflows
	workflowsPath := filepath.Join(absLocation, "environment", "workflows")
	//check if the folder exists
	_, err := os.Stat(workflowsPath)
	if err != nil {
		log.Warn().Msgf("could not find workflows folder: %s", workflowsPath)
		return
	}
	workflows, _ := os.ReadDir(workflowsPath)
	for _, workflow := range workflows {
		if workflow.IsDir() {
			workflowsPath := filepath.Join(workflowsPath, workflow.Name())

			files, _ := os.ReadDir(workflowsPath)
			for _, file := range files {
				fileName := strings.ToLower(file.Name())
				if strings.HasSuffix(fileName, "workflow.ts") || strings.HasSuffix(fileName, "workflow.js") {
					discoveredWorkflow := KindeWorkflow{
						WorkflowRootDirectory: workflowsPath,
						EntryPoints:           []string{file.Name()},
					}
					discoveredWorkflow.bundleAndIntrospect()
					kw.Workflows = append(kw.Workflows, discoveredWorkflow)

				}
			}

		}
	}
}

// Discover implements ProjectBundler.
func (p *projectBundler) Discover() (*KindeProject, error) {
	result := &KindeProject{}
	err := result.discoverKindeRoot(p.options.StartFolder)
	if err != nil {
		return nil, err
	}

	result.Environment.Workflows.discover(filepath.Join(result.Configuration.AbsLocation, result.Configuration.RootDir))

	return result, nil
}

func (kp *KindeProject) discoverKindeRoot(startFolder string) error {

	currentDirectory, _ := filepath.Abs(startFolder)

	fileInfo, _ := os.Stat(currentDirectory)
	if !fileInfo.IsDir() {
		currentDirectory = filepath.Dir(currentDirectory)
	}
	for {
		filePath := filepath.Join(currentDirectory, "kinde.json")
		configFile, _ := os.Stat(filePath)
		if configFile != nil {
			config, error := kp.readProjectConfiguration(filePath)
			if error != nil {
				return error
			}
			kp.Configuration = *config
			return nil

		}
		parentPath := filepath.Join(currentDirectory, "..")
		currentDirectory, _ = filepath.Abs(parentPath)
		if currentDirectory == "/" {
			return fmt.Errorf("could not find kinde.json")
		}
	}

}

func NewProjectBundler(discoveryOptions DiscoveryOptions) ProjectBundler {
	return &projectBundler{
		options: discoveryOptions,
	}
}

func (*KindeProject) readProjectConfiguration(configFileInfo string) (*ProjectConfiguration, error) {
	confiHandler, _ := os.Open(configFileInfo)
	configFile, err := io.ReadAll(confiHandler)
	if err != nil {
		return nil, fmt.Errorf("error reading project configuration: %w", err)
	}
	result := &ProjectConfiguration{}
	json.Unmarshal(configFile, result)
	result.AbsLocation = path.Dir(configFileInfo)
	if result.RootDir == "" {
		result.RootDir = "kindeSrc"
	}
	return result, nil
}

func (kw *KindeWorkflow) bundleAndIntrospect() {
	workflowBuilder := bundler.NewWorkflowBundler(bundler.BundlerOptions{
		WorkingFolder: kw.WorkflowRootDirectory,
		EntryPoints:   kw.EntryPoints,
	})
	bundlerResult := workflowBuilder.Bundle()
	kw.Bundle = bundlerResult

}

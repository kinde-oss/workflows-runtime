package project_bundler

import (
	"context"
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

	KindeWorkflow[TSettings any] struct {
		WorkflowRootDirectory string    `json:"workflow_root_directory"`
		EntryPoints           []string  `json:"entry_points"`
		Bundle                TSettings `json:"bundle"`
	}

	KindeEnvironment[TWorkflowSettings any] struct {
		Workflows []KindeWorkflow[TWorkflowSettings] `json:"workflows"`
	}

	KindeProject[TWorkflowSettings any] struct {
		Configuration ProjectConfiguration                `json:"configuration"`
		Environment   KindeEnvironment[TWorkflowSettings] `json:"environment"`
	}

	DiscoveryOptions[TWorkflowSettings any] struct {
		StartFolder string
	}

	ProjectBundler[TWorkflowSettings any] interface {
		Discover(ctx context.Context) (*KindeProject[TWorkflowSettings], error)
	}

	projectBundler[TWorkflowSettings any] struct {
		options DiscoveryOptions[TWorkflowSettings]
	}
)

func (kw *KindeEnvironment[TWorkflowSettings]) discover(ctx context.Context, absLocation string) {
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
				maybeAddWorkflow(ctx, file.Name(), workflowsPath, kw)
			}

		} else {
			maybeAddWorkflow(ctx, workflow.Name(), workflowsPath, kw)
		}
	}
}

func maybeAddWorkflow[TWorkflowSettings any](ctx context.Context, file string, rootDirectory string, kw *KindeEnvironment[TWorkflowSettings]) {
	fileName := strings.ToLower(file)
	if strings.HasSuffix(fileName, "workflow.ts") || strings.HasSuffix(fileName, "workflow.js") {
		discoveredWorkflow := KindeWorkflow[TWorkflowSettings]{
			WorkflowRootDirectory: rootDirectory,
			EntryPoints:           []string{file},
		}
		discoveredWorkflow.bundleAndIntrospect(ctx)
		kw.Workflows = append(kw.Workflows, discoveredWorkflow)
	}
}

// Discover implements ProjectBundler.
func (p *projectBundler[TWorkflowSettings]) Discover(ctx context.Context) (*KindeProject[TWorkflowSettings], error) {
	result := &KindeProject[TWorkflowSettings]{}
	err := result.discoverKindeRoot(p.options.StartFolder)
	if err != nil {
		return nil, err
	}

	result.Environment.discover(ctx, filepath.Join(result.Configuration.AbsLocation, result.Configuration.RootDir))

	return result, nil
}

func (kp *KindeProject[TWorkflowSettings]) discoverKindeRoot(startFolder string) error {

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

func NewProjectBundler[TWorkflowSettings any](discoveryOptions DiscoveryOptions[TWorkflowSettings]) ProjectBundler[TWorkflowSettings] {
	return &projectBundler[TWorkflowSettings]{
		options: discoveryOptions,
	}
}

func (*KindeProject[TWorkflowSettings]) readProjectConfiguration(configFileInfo string) (*ProjectConfiguration, error) {
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

func (kw *KindeWorkflow[TWorkflowSettings]) bundleAndIntrospect(ctx context.Context) {
	workflowBuilder := bundler.NewWorkflowBundler(bundler.BundlerOptions[TWorkflowSettings]{
		WorkingFolder: kw.WorkflowRootDirectory,
		EntryPoints:   kw.EntryPoints,
	})
	bundlerResult := workflowBuilder.Bundle(ctx)
	kw.Bundle = bundlerResult

}

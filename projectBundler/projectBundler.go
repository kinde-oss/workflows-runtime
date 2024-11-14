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
		WorkflowRootDirectory string                           `json:"workflow_root_directory"`
		EntryPoints           []string                         `json:"entry_points"`
		Bundle                bundler.BundlerResult[TSettings] `json:"bundle"`
	}

	KindePage[TSettings any] struct {
		RootDirectory string                           `json:"root_directory"`
		EntryPoints   []string                         `json:"entry_points"`
		Bundle        bundler.BundlerResult[TSettings] `json:"bundle"`
	}

	KindeEnvironment[TWorkflowSettings, TPageSettings any] struct {
		Workflows []KindeWorkflow[TWorkflowSettings] `json:"workflows"`
		Pages     []KindePage[TPageSettings]         `json:"pages"`
	}

	KindeProject[TWorkflowSettings, TPageSettings any] struct {
		Configuration ProjectConfiguration                               `json:"configuration"`
		Environment   KindeEnvironment[TWorkflowSettings, TPageSettings] `json:"environment"`
	}

	DiscoveryOptions[TWorkflowSettings, TPageSettings any] struct {
		StartFolder string
	}

	ProjectBundler[TWorkflowSettings, TPageSettings any] interface {
		Discover(ctx context.Context) (*KindeProject[TWorkflowSettings, TPageSettings], error)
	}

	projectBundler[TWorkflowSettings, TPageSettings any] struct {
		options DiscoveryOptions[TWorkflowSettings, TPageSettings]
	}
)

func (kw *KindeEnvironment[TWorkflowSettings, TPageSettings]) discoverWorkflows(ctx context.Context, absLocation string) {
	//environment/workflows
	workflowsPath := filepath.Join(absLocation, "environment", "workflows")
	//check if the folder exists
	_, err := os.Stat(workflowsPath)
	if err != nil {
		log.Warn().Msgf("could not find workflows folder: %s", workflowsPath)
		return
	}

	filepath.Walk(workflowsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			maybeAddWorkflow(ctx, info.Name(), filepath.Dir(path), kw)
		}
		return nil
	})

}

func maybeAddWorkflow[TWorkflowSettings, TPageSettings any](ctx context.Context, file string, rootDirectory string, kw *KindeEnvironment[TWorkflowSettings, TPageSettings]) {
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

func (kw *KindeEnvironment[TWorkflowSettings, TPageSettings]) discoverPages(ctx context.Context, absLocation string) {
	pagesPath := filepath.Join(absLocation, "environment", "pages")
	_, err := os.Stat(pagesPath)
	if err != nil {
		log.Warn().Msgf("could not find pages folder: %s", pagesPath)
	}

	filepath.Walk(pagesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			maybeAddPage(ctx, info.Name(), filepath.Dir(path), kw)
		}
		return nil
	})

}

func maybeAddPage[TWorkflowSettings, TPageSettings any](ctx context.Context, file string, rootDirectory string, kw *KindeEnvironment[TWorkflowSettings, TPageSettings]) {
	fileName := strings.ToLower(file)
	if strings.HasSuffix(fileName, "page.ts") || strings.HasSuffix(fileName, "page.js") || strings.HasSuffix(fileName, "page.tsx") || strings.HasSuffix(fileName, "page.jsx") {
		discoveredPage := KindePage[TPageSettings]{
			RootDirectory: rootDirectory,
			EntryPoints:   []string{file},
		}
		discoveredPage.bundleAndIntrospect(ctx)
		kw.Pages = append(kw.Pages, discoveredPage)
	}
}

// Discover implements ProjectBundler.
func (p *projectBundler[TWorkflowSettings, TPageSettings]) Discover(ctx context.Context) (*KindeProject[TWorkflowSettings, TPageSettings], error) {
	result := &KindeProject[TWorkflowSettings, TPageSettings]{}
	err := result.discoverKindeRoot(p.options.StartFolder)
	if err != nil {
		return nil, err
	}

	result.Environment.discoverWorkflows(ctx, filepath.Join(result.Configuration.AbsLocation, result.Configuration.RootDir))
	result.Environment.discoverPages(ctx, filepath.Join(result.Configuration.AbsLocation, result.Configuration.RootDir))

	return result, nil
}

func (kp *KindeProject[TWorkflowSettings, TPageSettings]) discoverKindeRoot(startFolder string) error {

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

func NewProjectBundler[TWorkflowSettings, TPageSettings any](discoveryOptions DiscoveryOptions[TWorkflowSettings, TPageSettings]) ProjectBundler[TWorkflowSettings, TPageSettings] {
	return &projectBundler[TWorkflowSettings, TPageSettings]{
		options: discoveryOptions,
	}
}

func (*KindeProject[TWorkflowSettings, TPageSettings]) readProjectConfiguration(configFileInfo string) (*ProjectConfiguration, error) {
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

func (kw *KindeWorkflow[TSettings]) bundleAndIntrospect(ctx context.Context) {
	workflowBuilder := bundler.NewWorkflowBundler(bundler.BundlerOptions[TSettings]{
		WorkingFolder: kw.WorkflowRootDirectory,
		EntryPoints:   kw.EntryPoints,
	})
	bundlerResult := workflowBuilder.Bundle(ctx)
	kw.Bundle = bundlerResult
}

func (kw *KindePage[TSettings]) bundleAndIntrospect(ctx context.Context) {
	workflowBuilder := bundler.NewWorkflowBundler(bundler.BundlerOptions[TSettings]{
		WorkingFolder: kw.RootDirectory,
		EntryPoints:   kw.EntryPoints,
	})
	bundlerResult := workflowBuilder.Bundle(ctx)
	kw.Bundle = bundlerResult
}

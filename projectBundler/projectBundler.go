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

const (
	projectSettingsContextKey projectSettings = "projectSettings"
)

type (
	projectSettings string

	// ProjectConfiguration is the struct that holds the project configuration.
	ProjectConfiguration struct {
		Version     string `json:"version"`
		RootDir     string `json:"rootDir"`
		AbsLocation string `json:"location"`
	}

	// KindeWorkflow is the struct that holds the workflow configuration.
	KindeWorkflow[TSettings any] struct {
		WorkflowRootDirectory string                           `json:"workflow_root_directory"`
		EntryPoints           []string                         `json:"entry_points"`
		Bundle                bundler.BundlerResult[TSettings] `json:"bundle"`
	}

	// KindePage is the struct that holds the page configuration.
	KindePage[TSettings any] struct {
		RootDirectory string                           `json:"root_directory"`
		EntryPoints   []string                         `json:"entry_points"`
		Bundle        bundler.BundlerResult[TSettings] `json:"bundle"`
	}

	// KindeEnvironment is the struct that holds the workflows and pages.
	KindeEnvironment[TWorkflowSettings, TPageSettings any] struct {
		Workflows        []KindeWorkflow[TWorkflowSettings] `json:"workflows"`
		Pages            []KindePage[TPageSettings]         `json:"pages"`
		discoveryOptions DiscoveryOptions[TWorkflowSettings, TPageSettings]
	}

	// KindeProject is the struct that holds the project configuration and the environment.
	KindeProject[TWorkflowSettings, TPageSettings any] struct {
		Configuration ProjectConfiguration                               `json:"configuration"`
		Environment   KindeEnvironment[TWorkflowSettings, TPageSettings] `json:"environment"`
	}

	// DiscoveryOptions is the struct that holds the options for the project discovery.
	DiscoveryOptions[TWorkflowSettings, TPageSettings any] struct {
		StartFolder          string
		OnRootDiscovered     func(ctx context.Context, bundle ProjectConfiguration)
		OnWorkflowDiscovered func(ctx context.Context, bundle *bundler.BundlerResult[TWorkflowSettings])
		OnPageDiscovered     func(ctx context.Context, bundle *bundler.BundlerResult[TPageSettings])
	}

	// ProjectBundler is the interface that wraps the Discover method.
	ProjectBundler[TWorkflowSettings, TPageSettings any] interface {
		Discover(ctx context.Context) (*KindeProject[TWorkflowSettings, TPageSettings], error)
	}

	projectBundler[TWorkflowSettings, TPageSettings any] struct {
		options DiscoveryOptions[TWorkflowSettings, TPageSettings]
	}
)

// GetProjectConfiguration returns the project configuration from the context.
// If the configuration is not found, it returns nil.
func GetProjectConfiguration(ctx context.Context) *ProjectConfiguration {
	if val, ok := ctx.Value(projectSettingsContextKey).(ProjectConfiguration); ok {
		return &val
	}
	return nil
}

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
		discoveredWorkflow.bundleAndIntrospect(ctx, kw.discoveryOptions.OnWorkflowDiscovered)
		kw.Workflows = append(kw.Workflows, discoveredWorkflow)
	}
}

func (kw *KindeEnvironment[TWorkflowSettings, TPageSettings]) discoverPages(ctx context.Context, absLocation string) {
	pagesPath := filepath.Join(absLocation, "environment", "pages")
	_, err := os.Stat(pagesPath)
	if err != nil {
		log.Warn().Msgf("could not find pages folder: %s", pagesPath)
	}

	discoveredFolders := make(map[string]bool)

	filepath.Walk(pagesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if _, ok := discoveredFolders[filepath.Dir(path)]; ok {
			return nil //skip if we already discovered this folder, so double names not clash (precedence is js -> ts -> tsx -> jsx)
		}
		if !info.IsDir() {
			if maybeAddPage(ctx, info.Name(), filepath.Dir(path), kw) {
				discoveredFolders[filepath.Dir(path)] = true
			}
		}
		return nil
	})

}

func maybeAddPage[TWorkflowSettings, TPageSettings any](ctx context.Context, file string, rootDirectory string, kw *KindeEnvironment[TWorkflowSettings, TPageSettings]) bool {
	fileName := strings.ToLower(file)
	if strings.EqualFold(fileName, "page.ts") || strings.EqualFold(fileName, "page.js") || strings.EqualFold(fileName, "page.tsx") || strings.EqualFold(fileName, "page.jsx") {
		discoveredPage := KindePage[TPageSettings]{
			RootDirectory: rootDirectory,
			EntryPoints:   []string{file},
		}
		discoveredPage.bundleAndIntrospect(ctx, kw.discoveryOptions.OnPageDiscovered)
		kw.Pages = append(kw.Pages, discoveredPage)
		return true
	}
	return false
}

// Discover discovers the project and returns the project configuration and the environment.
func (p *projectBundler[TWorkflowSettings, TPageSettings]) Discover(ctx context.Context) (*KindeProject[TWorkflowSettings, TPageSettings], error) {
	project := &KindeProject[TWorkflowSettings, TPageSettings]{
		Environment: KindeEnvironment[TWorkflowSettings, TPageSettings]{
			discoveryOptions: p.options,
		},
	}

	err := project.discoverKindeRoot(p.options.StartFolder)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, projectSettingsContextKey, project.Configuration)

	if p.options.OnRootDiscovered != nil {
		p.options.OnRootDiscovered(ctx, project.Configuration)
	}

	project.Environment.discoverWorkflows(ctx, filepath.Join(project.Configuration.AbsLocation, project.Configuration.RootDir))
	project.Environment.discoverPages(ctx, filepath.Join(project.Configuration.AbsLocation, project.Configuration.RootDir))

	return project, nil
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

// NewProjectBundler returns a new instance of ProjectBundler.
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

func (kw *KindeWorkflow[TSettings]) bundleAndIntrospect(ctx context.Context, onDiscovered func(ctx context.Context, bundle *bundler.BundlerResult[TSettings])) {
	workflowBuilder := bundler.NewWorkflowBundler(bundler.BundlerOptions[TSettings]{
		WorkingFolder:       kw.WorkflowRootDirectory,
		EntryPoints:         kw.EntryPoints,
		IntrospectionExport: "workflowSettings",
		OnDiscovered:        onDiscovered,
	})
	bundlerResult := workflowBuilder.Bundle(ctx)
	kw.Bundle = bundlerResult
}

func (kw *KindePage[TSettings]) bundleAndIntrospect(ctx context.Context, onDiscovered func(ctx context.Context, bundle *bundler.BundlerResult[TSettings])) {
	workflowBuilder := bundler.NewWorkflowBundler(bundler.BundlerOptions[TSettings]{
		WorkingFolder:       kw.RootDirectory,
		EntryPoints:         kw.EntryPoints,
		IntrospectionExport: "pageSettings",
		OnDiscovered:        onDiscovered,
	})
	bundlerResult := workflowBuilder.Bundle(ctx)
	kw.Bundle = bundlerResult
}

package builder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	_ "github.com/kinde-oss/workflows-runtime/gojaRuntime"
	runtimesRegistry "github.com/kinde-oss/workflows-runtime/registry"
)

const pluginsKey bundlerContext = "bundlerPlugins"

type (
	bundlerContext string

	WorkflowSettings[TSettings any] struct {
		Bindings map[string]runtimesRegistry.BindingSettings `json:"bindings"`
		Other    TSettings                                   `json:"other"`
	}

	BundledContent[TSettings any] struct {
		Source          []byte                      `json:"source"`
		BundleHash      string                      `json:"hash"`
		Settings        WorkflowSettings[TSettings] `json:"settings"`
		BundlingOptions BundlerOptions[TSettings]   `json:"bundling_options"`
	}

	BundlerResult[TSettings any] struct {
		Content           BundledContent[TSettings] `json:"bundle"`
		Errors            []string                  `json:"errors"`
		CompilationErrors []interface{}             `json:"compilation_errors"`
		Warnings          []string                  `json:"warnings"`
		DefaultExports    []string                  `json:"default_exports"`
	}

	BundlerOptions[TSettings any] struct {
		WorkingFolder       string                                                      `json:"working_folder"`
		EntryPoints         []string                                                    `json:"entry_points"`
		IntrospectionExport string                                                      `json:"introspection_export"`
		OnDiscovered        func(ctx context.Context, bundle *BundlerResult[TSettings]) `json:"-"`
	}

	WorkflowBundler[TSettings any] interface {
		Bundle(ctx context.Context) BundlerResult[TSettings]
	}

	builder[TSettings any] struct {
		bundleOptions BundlerOptions[TSettings]
	}
)

func NewWorkflowBundler[TSettings any](options BundlerOptions[TSettings]) WorkflowBundler[TSettings] {
	return &builder[TSettings]{
		bundleOptions: options,
	}
}

func WithBundlerPlugins(ctx context.Context, plugins []api.Plugin) context.Context {
	return context.WithValue(ctx, pluginsKey, plugins)
}

func (b *builder[TSettings]) getContextPlugins(ctx context.Context) []api.Plugin {
	if ctx == nil {
		return nil
	}

	if plugins, ok := ctx.Value(pluginsKey).([]api.Plugin); ok {
		return plugins
	}

	return nil
}

func (b *builder[TSettings]) Bundle(ctx context.Context) BundlerResult[TSettings] {
	opts := api.BuildOptions{
		Loader: map[string]api.Loader{
			".js":  api.LoaderJS,
			".tsx": api.LoaderTSX,
			".ts":  api.LoaderTS,
			".jsx": api.LoaderJSX,
		},
		AbsWorkingDir:    b.bundleOptions.WorkingFolder,
		Target:           api.ESNext,
		Format:           api.FormatCommonJS,
		Sourcemap:        api.SourceMapInline,
		SourcesContent:   api.SourcesContentInclude,
		LegalComments:    api.LegalCommentsNone,
		Platform:         api.PlatformDefault,
		LogLevel:         api.LogLevelSilent,
		Charset:          api.CharsetUTF8,
		EntryPoints:      b.bundleOptions.EntryPoints,
		Bundle:           true,
		Write:            false,
		TreeShaking:      api.TreeShakingTrue,
		MinifyWhitespace: true,
		MinifySyntax:     true,
		Outdir:           "output",
		Plugins:          b.getContextPlugins(ctx),
	}
	tr := api.Build(opts)

	result := BundlerResult[TSettings]{}

	if len(tr.OutputFiles) > 0 {

		if len(tr.OutputFiles) > 1 {
			result.addError(errors.New("build produced multiple files, a single output is supported only"))
		}

		file := tr.OutputFiles[0]

		settings, settingsDiscoveryWarn, discoveryErr := result.discoverSettingsAndExports(b.bundleOptions.IntrospectionExport, file.Contents)

		if settingsDiscoveryWarn != nil {
			result.addWarning(settingsDiscoveryWarn)
		}

		if discoveryErr != nil {
			result.addError(discoveryErr)
		}

		result.Content = BundledContent[TSettings]{
			Source:          file.Contents,
			BundleHash:      file.Hash,
			Settings:        settings,
			BundlingOptions: b.bundleOptions,
		}

	}

	for _, buildError := range tr.Errors {
		result.addCompilationError(buildError)
	}

	if b.bundleOptions.OnDiscovered != nil {
		b.bundleOptions.OnDiscovered(ctx, &result)
	}

	return result
}

func (br *BundlerResult[TSettings]) HasOutput() bool {
	return len(br.Content.Source) > 0
}

func (br *BundlerResult[TSettings]) addCompilationError(err interface{}) {
	br.CompilationErrors = append(br.CompilationErrors, err)
}

func (br *BundlerResult[TSettings]) addError(err error) {
	br.Errors = append(br.Errors, err.Error())
}

func (br *BundlerResult[TSettings]) addWarning(warn error) {
	br.Warnings = append(br.Warnings, warn.Error())
}

func (br *BundlerResult[TSettings]) discoverSettingsAndExports(exportName string, source []byte) (r WorkflowSettings[TSettings], warn error, err error) {
	goja, _ := runtimesRegistry.ResolveRuntime("goja")
	introspectResult, err := goja.Introspect(context.Background(),
		runtimesRegistry.WorkflowDescriptor{
			ProcessedSource: runtimesRegistry.SourceDescriptor{
				Source:     source,
				SourceType: runtimesRegistry.Source_ContentType_Text,
			},
			Limits: runtimesRegistry.RuntimeLimits{
				MaxExecutionDuration: 30 * time.Second,
			},
		},
		runtimesRegistry.IntrospectionOptions{
			Exports: []string{exportName},
		})

	if introspectResult == nil {
		return WorkflowSettings[TSettings]{}, fmt.Errorf("export %v not found", exportName), err
	}

	if err != nil {
		return WorkflowSettings[TSettings]{}, nil, err
	}

	export := introspectResult.GetExport(exportName)

	asMap := export.ValueAsMap()
	res, _ := json.Marshal(asMap)

	result := WorkflowSettings[TSettings]{}

	json.Unmarshal(res, &result)

	return result, nil, nil
}

func (settings *WorkflowSettings[TSettings]) UnmarshalJSON(data []byte) error {

	type bindings struct {
		Bindings map[string]runtimesRegistry.BindingSettings `json:"bindings"`
	}

	mapData := bindings{}
	err := json.Unmarshal(data, &mapData)
	if err != nil {
		return err
	}
	settings.Bindings = mapData.Bindings

	set := new(TSettings)
	err = json.Unmarshal(data, set)

	settings.Other = *set

	return err

}

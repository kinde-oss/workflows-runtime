package builder

import (
	"context"
	"errors"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	_ "github.com/kinde-oss/workflows-runtime/gojaRuntime"
	runtimesRegistry "github.com/kinde-oss/workflows-runtime/registry"
)

type (
	WorkflowSettings struct {
		ID    string                 `json:"id"`
		Other map[string]interface{} `json:"other"`
	}
	BundledContent struct {
		Source     []byte           `json:"source"`
		BundleHash string           `json:"hash"`
		Settings   WorkflowSettings `json:"settings"`
	}

	BundlerResult struct {
		Bundle BundledContent `json:"bundle"`
		Errors []error        `json:"errors"`
	}

	BundlerOptions struct {
		WorkingFolder string
		EntryPoints   []string
	}

	WorkflowBundler interface {
		Bundle() BundlerResult
	}

	builder struct {
		bundleOptions BundlerOptions
	}
)

func newWorkflowBundler(options BundlerOptions) WorkflowBundler {
	return &builder{
		bundleOptions: options,
	}
}

func (b *builder) Bundle() BundlerResult {
	opts := api.BuildOptions{
		Loader: map[string]api.Loader{
			".js":  api.LoaderJS,
			".tsx": api.LoaderTSX,
			".ts":  api.LoaderTS,
		},
		AbsWorkingDir:    b.bundleOptions.WorkingFolder,
		Target:           api.ESNext,
		Format:           api.FormatCommonJS,
		Sourcemap:        api.SourceMapInline,
		SourcesContent:   api.SourcesContentInclude,
		LegalComments:    api.LegalCommentsNone,
		Platform:         api.PlatformDefault,
		LogLevel:         api.LogLevelError,
		Charset:          api.CharsetUTF8,
		EntryPoints:      b.bundleOptions.EntryPoints,
		Bundle:           true,
		Write:            false,
		TreeShaking:      api.TreeShakingTrue,
		MinifyWhitespace: true,
		MinifySyntax:     true,
		Outdir:           "output",
	}
	tr := api.Build(opts)

	result := BundlerResult{}

	if len(tr.OutputFiles) > 0 {

		if len(tr.Errors) > 1 {
			result.addError(errors.New("build produced multiple files, a single output is supported only"))
		}

		file := tr.OutputFiles[0]
		result.Bundle = BundledContent{
			Source:     file.Contents,
			BundleHash: file.Hash,
			Settings:   result.discoverSettings(file.Contents),
		}
	}

	if result.Bundle.Settings.ID == "" {
		result.addError(errors.New("workflow id not found, please export workflowSettings.id"))
	}

	return result
}

func (br *BundlerResult) HasOutput() bool {
	return len(br.Bundle.Source) > 0
}

func (br *BundlerResult) addError(err error) {
	br.Errors = append(br.Errors, err)
}

func (br *BundlerResult) discoverSettings(source []byte) WorkflowSettings {
	goja, _ := runtimesRegistry.ResolveRuntime("goja")
	introspectResult, _ := goja.Introspect(context.Background(),
		runtimesRegistry.WorkflowDescriptor{
			ProcessedSource: runtimesRegistry.CodeDescriptor{
				Source:     source,
				SourceType: runtimesRegistry.Source_ContentType_Text,
			},
			Bindings: runtimesRegistry.Bindings{
				GlobalModules: map[string]runtimesRegistry.ModuleBinding{
					"console": {},
					"url":     {},
					"module":  {},
					"kinde":   {},
				},
			},
			Limits: runtimesRegistry.RuntimeLimits{
				MaxExecutionDuration: 30 * time.Second,
			},
		},
		runtimesRegistry.InstrospectionOptions{
			Exports: []string{"workflowSettings"},
		})

	settings := introspectResult.GetExport("workflowSettings").ValueAsMap()

	var workflowID string

	if id, ok := settings["id"]; ok {

		switch idTyped := id.(type) {
		case string:
			workflowID = idTyped
		}
	}

	return WorkflowSettings{
		ID:    workflowID,
		Other: settings,
	}
}

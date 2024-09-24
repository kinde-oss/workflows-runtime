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
		ID    string
		Name  string
		Other map[string]interface{}
	}
	BundledContent struct {
		Source   []byte
		Hash     string
		Settings WorkflowSettings
	}

	BundlerResult struct {
		Bundle BundledContent
		Errors []error
	}

	BundlerOptions struct {
		WorkingFolder string
		EntryPoints   []string
	}

	WorkflowBundler interface {
		Bundle() BundlerResult
	}

	builder struct {
		buildOptions BundlerOptions
	}
)

func newWorkflowBundler(options BundlerOptions) WorkflowBundler {
	return &builder{
		buildOptions: options,
	}
}

func (b *builder) Bundle() BundlerResult {
	opts := api.BuildOptions{
		Loader: map[string]api.Loader{
			".js":  api.LoaderJS,
			".tsx": api.LoaderTSX,
			".ts":  api.LoaderTS,
		},
		AbsWorkingDir:    b.buildOptions.WorkingFolder,
		Target:           api.ESNext,
		Format:           api.FormatCommonJS,
		Sourcemap:        api.SourceMapInline,
		SourcesContent:   api.SourcesContentInclude,
		LegalComments:    api.LegalCommentsNone,
		Platform:         api.PlatformDefault,
		LogLevel:         api.LogLevelError,
		Charset:          api.CharsetUTF8,
		EntryPoints:      b.buildOptions.EntryPoints,
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
			Source:   file.Contents,
			Hash:     file.Hash,
			Settings: result.discoverSettings(file.Contents),
		}
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
	introspectResult, _ := goja.Introspect(context.Background(), runtimesRegistry.WorkflowDescriptor{
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
	})

	return WorkflowSettings{
		ID: introspectResult.GetID(),
	}
}

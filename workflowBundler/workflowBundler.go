package builder

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	_ "github.com/kinde-oss/workflows-runtime/gojaRuntime"
	runtimesRegistry "github.com/kinde-oss/workflows-runtime/registry"
)

const (
	PolicyActionStop     PolicyAction = "stop"
	PolicyActionContinue PolicyAction = "continue"
)

type (
	PolicyAction  string
	FailurePolicy struct {
		Action PolicyAction `json:"action"`
	}

	WorkflowSettings struct {
		ID            string                                      `json:"id"`
		Other         map[string]interface{}                      `json:"other"`
		Bindings      map[string]runtimesRegistry.BindingSettings `json:"bindings"`
		FailurePolicy FailurePolicy                               `json:"failure_policy"`
	}
	BundledContent struct {
		Source     []byte           `json:"source"`
		BundleHash string           `json:"hash"`
		Settings   WorkflowSettings `json:"settings"`
	}

	BundlerResult struct {
		Content           BundledContent `json:"bundle"`
		Errors            []string       `json:"errors"`
		CompilationErrors []interface{}  `json:"compilation_errors"`
	}

	BundlerOptions struct {
		WorkingFolder       string
		EntryPoints         []string
		IntrospectionExport string
	}

	WorkflowBundler interface {
		Bundle() BundlerResult
	}

	builder struct {
		bundleOptions BundlerOptions
	}
)

func (a PolicyAction) IsValid() bool {
	switch a {
	case PolicyActionStop, PolicyActionContinue:
		return true
	}
	return false
}

func NewWorkflowBundler(options BundlerOptions) WorkflowBundler {
	if options.IntrospectionExport == "" {
		options.IntrospectionExport = "workflowSettings"
	}
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
		LogLevel:         api.LogLevelSilent,
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

		if len(tr.OutputFiles) > 1 {
			result.addError(errors.New("build produced multiple files, a single output is supported only"))
		}

		file := tr.OutputFiles[0]
		result.Content = BundledContent{
			Source:     file.Contents,
			BundleHash: file.Hash,
			Settings:   result.discoverSettings(b.bundleOptions.IntrospectionExport, file.Contents),
		}
	}

	for _, buildError := range tr.Errors {
		result.addCompilationError(buildError)

	}

	if result.Content.Settings.ID == "" {
		result.addError(errors.New("workflow id not found, please export workflowSettings.id"))
	}

	return result
}

func (br *BundlerResult) HasOutput() bool {
	return len(br.Content.Source) > 0
}

func (br *BundlerResult) addCompilationError(err interface{}) {
	br.CompilationErrors = append(br.CompilationErrors, err)
}

func (br *BundlerResult) addError(err error) {
	br.Errors = append(br.Errors, err.Error())
}

func (br *BundlerResult) discoverSettings(exportName string, source []byte) WorkflowSettings {
	goja, _ := runtimesRegistry.ResolveRuntime("goja")
	introspectResult, _ := goja.Introspect(context.Background(),
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

	settings := introspectResult.GetExport(exportName)

	var workflowID string

	if id, ok := settings.ValueAsMap()["id"]; ok {

		switch idTyped := id.(type) {
		case string:
			workflowID = idTyped
		}
	}

	var failurePolicy FailurePolicy
	if policy, ok := settings.ValueAsMap()["failurePolicy"]; ok {
		jsonData, err := json.Marshal(policy)
		if err != nil {
			br.addError(errors.New("failed to determine failure policy"))
		}
		err = json.Unmarshal(jsonData, &failurePolicy)
		if err != nil {
			br.addError(errors.New("failed to determine failure policy"))
		}

		if !PolicyAction(failurePolicy.Action).IsValid() {
			br.addError(errors.New("invalid failure policy action"))
			failurePolicy = FailurePolicy{}
		}
	} else {
		failurePolicy = FailurePolicy{
			Action: PolicyActionStop,
		}
	}

	return WorkflowSettings{
		ID:            workflowID,
		Other:         settings.ValueAsMap(),
		FailurePolicy: failurePolicy,
		Bindings:      settings.BindingsFrom(exportName),
	}
}

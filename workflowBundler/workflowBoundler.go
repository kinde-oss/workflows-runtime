package builder

import (
	"errors"

	"github.com/evanw/esbuild/pkg/api"
)

type (
	BundledContent struct {
		Source []byte
		//Path   string
		Hash string
	}

	BundlerResult struct {
		Boundle BundledContent
		Errors  []error
	}

	BundlerOptions struct {
		WorkingFolder string
		EntryPoint    string
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
		EntryPoints:      []string{b.buildOptions.EntryPoint},
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
		result.Boundle = BundledContent{
			Source: file.Contents,
			//Path:   file.Path,
			Hash: file.Hash,
		}
	}

	return result
}

func (br *BundlerResult) HasOutput() bool {
	return len(br.Boundle.Source) > 0
}

func (br *BundlerResult) addError(err error) {
	br.Errors = append(br.Errors, err)
}

package runtime_registry

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"time"
)

const (
	Source_ContentType_Text   SourceContentType = iota
	Source_ContentType_Binary                   = iota
)

type (
	SourceContentType int

	StartOptions struct {
		EntryPoint string
		Arguments  []interface{}
	}

	SourceDescriptor struct {
		Source     []byte            `json:"source"`
		SourceType SourceContentType `json:"source_type"`
		BuildHash  string            `json:"build_hash"`
	}

	ModuleBindingConfiguration struct {
		Settings map[string]interface{} `json:"configuration"`
	}
	ModuleBinding struct {
		Configuration ModuleBindingConfiguration `json:"configuration"`
		ContextKey    string                     `json:"context_key"`
	}

	Bindings struct {
		GlobalModules map[string]ModuleBinding `json:"global_modules"`
		KindeAPIs     map[string]ModuleBinding `json:"kinde_apis"`
	}

	RuntimeLimits struct {
		MaxExecutionDuration time.Duration `json:"max_execution_duration"`
	}

	WorkflowDescriptor struct {
		ProcessedSource SourceDescriptor `json:"processed_source"`
		Bindings        Bindings         `json:"bindings"`
		Limits          RuntimeLimits    `json:"runtime_limits"`
	}

	ExecutionResult interface {
		GetExitResult() interface{}
		GetConsoleLog() []interface{}
		GetConsoleError() []interface{}
		GetContext() map[string]interface{}
	}

	IntrospectedExport interface {
		HasExport() bool
		Value() interface{}
		ValueAsMap() map[string]interface{}
	}

	IntrospectionResult interface {
		GetExport(string) IntrospectedExport
	}

	IntrospectionOptions struct {
		Exports []string
	}

	Runner interface {
		Execute(ctx context.Context, workflow WorkflowDescriptor, startOptions StartOptions) (ExecutionResult, error)
		Introspect(ctx context.Context, workflow WorkflowDescriptor, options IntrospectionOptions) (IntrospectionResult, error)
	}
)

var runtimes map[string]func() Runner = map[string]func() Runner{}

// not thread safe, should be called as part of init
func RegisterRuntime(name string, factory func() Runner) {
	runtimes[name] = factory
}

// Resolves runtime from available registrations
func ResolveRuntime(name string) (Runner, error) {
	if factory, ok := runtimes[name]; ok {
		return factory(), nil
	}
	return nil, fmt.Errorf("runtime %v not found", name)
}

// Returns a hash of the workflow descriptor
func (wd *WorkflowDescriptor) GetHash() string {
	sha := sha256.New()
	sha.Write([]byte(wd.ProcessedSource.Source))
	result := base32.StdEncoding.EncodeToString(sha.Sum(nil))
	return fmt.Sprintf("%v", result)
}

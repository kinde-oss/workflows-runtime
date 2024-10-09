package runtime_registry

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"time"
)

const (
	Source_ContentType_Text   SourceContentType = iota
	Source_ContentType_Binary                   = iota

	LogLevelInfo LogLevel = iota
	LogLevelDebug
	LogLevelWarning
	LogLevelError
	LogLevelSilent
)

type (
	SourceContentType int

	LogLevel int

	Logger interface {
		Log(level LogLevel, params ...interface{})
	}

	StartOptions struct {
		EntryPoint string
		Arguments  []interface{}
		Loggger    Logger
	}

	SourceDescriptor struct {
		Source     []byte            `json:"source"`
		SourceType SourceContentType `json:"source_type"`
		BuildHash  string            `json:"build_hash"`
	}

	BindingSettings struct {
		Settings map[string]interface{} `json:"settings"`
	}

	RuntimeLimits struct {
		MaxExecutionDuration time.Duration `json:"max_execution_duration"`
	}

	WorkflowDescriptor struct {
		ProcessedSource   SourceDescriptor           `json:"processed_source"`
		RequestedBindings map[string]BindingSettings `json:"bindings"`
		Limits            RuntimeLimits              `json:"runtime_limits"`
	}

	RuntimeContext interface {
		GetValues() map[string]interface{}
		GetValueAsMap(key string) (map[string]interface{}, error)
	}

	ExecutionMetadata struct {
		StartedAt          time.Time     `json:"started_at"`
		ExecutionDuration  time.Duration `json:"execution_duration"`
		HasRunToCompletion bool          `json:"has_run_to_completion"`
	}
	ExecutionResult interface {
		ExecutionMetadata() ExecutionMetadata
		GetExitResult() interface{}
		GetContext() RuntimeContext
	}

	IntrospectedExport interface {
		HasExport() bool
		Value() interface{}
		ValueAsMap() map[string]interface{}
		BindingsFrom(exportName string) map[string]BindingSettings
	}

	IntrospectionResult interface {
		GetExport(string) IntrospectedExport
	}

	IntrospectionOptions struct {
		Exports []string
		Logger  Logger
	}

	Runner interface {
		Execute(ctx context.Context, workflow WorkflowDescriptor, startOptions StartOptions) (ExecutionResult, error)
		Introspect(ctx context.Context, workflow WorkflowDescriptor, options IntrospectionOptions) (IntrospectionResult, error)
	}
)

func (settings *BindingSettings) UnmarshalJSON(data []byte) error {
	jsonMap := map[string]interface{}{}
	err := json.Unmarshal(data, &jsonMap)
	settings.Settings = jsonMap
	return err
}

func (settings BindingSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(settings.Settings)
}

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

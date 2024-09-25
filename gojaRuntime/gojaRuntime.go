package goja_runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/kinde-oss/workflows-runtime/gojaRuntime/require"
	urlModule "github.com/kinde-oss/workflows-runtime/gojaRuntime/url"
	runtimesRegistry "github.com/kinde-oss/workflows-runtime/registry"
)

type (
	gojaRunnerV1 struct {
		Cache *gojaCache
	}

	actionResult struct {
		ConsoleLog   []interface{}          `json:"console_log"`
		ConsoleError []interface{}          `json:"console_error"`
		Context      map[string]interface{} `json:"context"`
		ExitResult   interface{}            `json:"exit_result"`
	}
	introspectedExport struct {
		value interface{}
	}

	introspectionResult struct {
		exports map[string]introspectedExport
	}
)

// HasExport implements runtime_registry.IntrospectedExport.
func (i introspectedExport) HasExport() bool {
	return i.value != nil
}

// Value implements runtime_registry.IntrospectedExport.
func (i introspectedExport) Value() interface{} {
	return i.value
}

// ValueAsMap implements runtime_registry.IntrospectedExport.
func (i introspectedExport) ValueAsMap() map[string]interface{} {
	if i.value == nil {
		return map[string]interface{}{}
	}
	return i.value.(map[string]interface{})
}

// GetExport implements runtime_registry.IntrospectionResult.
func (i introspectionResult) GetExport(name string) runtimesRegistry.IntrospectedExport {
	return i.exports[name]
}

func (i introspectionResult) recordExport(name string, value interface{}) {
	i.exports[name] = introspectedExport{
		value: value,
	}
}

// GetConsoleError implements runtime_registry.Result.
func (a *actionResult) GetConsoleError() []interface{} {
	return a.ConsoleError
}

// GetConsoleLog implements runtime_registry.Result.
func (a *actionResult) GetConsoleLog() []interface{} {
	return a.ConsoleLog
}

// GetContext implements runtime_registry.Result.
func (a *actionResult) GetContext() map[string]interface{} {
	return a.Context
}

func (a *actionResult) GetExitResult() interface{} {
	return a.ExitResult
}

var registry = new(require.Registry)

var availableModules = map[string]func(e *gojaRunnerV1, vm *goja.Runtime, mountingPoint *goja.Object, result *actionResult, binding runtimesRegistry.ModuleBinding){
	"console": func(runner *gojaRunnerV1, vm *goja.Runtime, mountingPoint *goja.Object, result *actionResult, binding runtimesRegistry.ModuleBinding) {
		vm.Set("console", vm.NewObject())
		consoleMountingPoint := vm.Get("console").(*goja.Object)
		runner.consoleEmulation(vm, consoleMountingPoint, result, binding)
	},
	"url": func(e *gojaRunnerV1, vm *goja.Runtime, mountingPoint *goja.Object, _ *actionResult, _ runtimesRegistry.ModuleBinding) {
		vm.Set("url", require.Require(vm, "url"))
	},
	"module": func(e *gojaRunnerV1, vm *goja.Runtime, mountingPoint *goja.Object, _ *actionResult, _ runtimesRegistry.ModuleBinding) {
		vm.Set("module", vm.NewObject())
	},
}

var kindeAPIs = map[string]func(runtimesRegistry.ModuleBinding, ...interface{}) (interface{}, error){}

func RegisterKindeAPI(apiName string, api func(runtimesRegistry.ModuleBinding, ...interface{}) (interface{}, error)) {
	kindeAPIs[apiName] = api
}

func init() {
	runtimesRegistry.RegisterRuntime("goja", newGojaRunner)

	registry.RegisterNativeModule("url", urlModule.Require)

}

func newGojaRunner() runtimesRegistry.Runner {
	runner := gojaRunnerV1{
		Cache: &gojaCache{
			cache: map[string]*goja.Program{},
		},
	}
	return &runner
}

// Introspect implements runtime_registry.Runner.
func (e *gojaRunnerV1) Introspect(ctx context.Context, workflow runtimesRegistry.WorkflowDescriptor, options runtimesRegistry.InstrospectionOptions) (runtimesRegistry.IntrospectionResult, error) {
	vm := goja.New()
	_, returnErr := setupVM(ctx, vm, e, workflow)

	if returnErr != nil {
		return nil, returnErr
	}
	module := vm.Get("module").ToObject(vm)
	exports := module.Get("exports").ToObject(vm)

	introspectionResult := introspectionResult{
		exports: map[string]introspectedExport{},
	}

	for _, exportToIntrospect := range options.Exports {
		exportIntrospect := exports.Get(exportToIntrospect)
		if exportIntrospect != nil {
			mapped := exportIntrospect.Export()
			introspectionResult.recordExport(exportToIntrospect, mapped)
		} else {
			introspectionResult.recordExport(exportToIntrospect, nil)
		}
	}

	return introspectionResult, nil
}

func (e *gojaRunnerV1) Execute(ctx context.Context, workflow runtimesRegistry.WorkflowDescriptor, startOptions runtimesRegistry.StartOptions) (runtimesRegistry.ExecutionResult, error) {

	vm := goja.New()
	executionResult, returnErr := setupVM(ctx, vm, e, workflow)

	if returnErr != nil {
		return executionResult, returnErr
	}

	module := vm.Get("module").ToObject(vm)
	exports := module.Get("exports").ToObject(vm)

	defaultExport := exports.Get("default")
	if defaultExport == nil {
		return nil, fmt.Errorf("no default export")
	}

	targetVmFunction := defaultExport.ToObject(vm).Get(startOptions.EntryPoint)
	if targetVmFunction == nil {
		return nil, fmt.Errorf("could not find default exported function %v", startOptions.EntryPoint)
	}
	var callableFunction goja.Callable
	vm.ExportTo(targetVmFunction, &callableFunction)

	functionParams := []goja.Value{}
	for _, arg := range startOptions.Arguments {
		functionParams = append(functionParams, vm.ToValue(arg))
	}

	result, err := callableFunction(nil, functionParams...)

	if err != nil {
		return nil, fmt.Errorf("%v", err.Error())
	}

	promise := result.Export().(*goja.Promise)
	for {
		if promise.State() == goja.PromiseStateRejected {
			returnedError := promise.Result().String()

			returnedError = strings.ReplaceAll(returnedError, "GoError: ", "")

			exportedResult := promise.Result().ToObject(vm)
			stackExport := exportedResult.Get("stack")
			if exportedResult != nil && stackExport != nil {
				errorText := fmt.Sprintf("%v", stackExport.Export())
				errorText = strings.ReplaceAll(errorText, "GoError: ", "")
				return nil, fmt.Errorf("%v", errorText)
			}

			return nil, fmt.Errorf("%v", returnedError)
		}
		if promise.State() != goja.PromiseStatePending {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}

	executionResult.ExitResult = promise.Result().Export()

	return executionResult, nil
}

func setupVM(ctx context.Context, vm *goja.Runtime, runner *gojaRunnerV1, workflow runtimesRegistry.WorkflowDescriptor) (*actionResult, error) {
	registry.Enable(vm)

	runner.maxExecutionTimeout(ctx, vm, workflow.Limits.MaxExecutionDuration)
	vm.SetTimeSource(func() time.Time { return time.Now() })

	executionResult := &actionResult{
		ConsoleLog:   []interface{}{},
		ConsoleError: []interface{}{},
		Context:      map[string]interface{}{},
	}

	for name, binding := range workflow.Bindings.GlobalModules {
		if module, ok := availableModules[name]; ok {
			module(runner, vm, vm.NewObject(), executionResult, binding)
		}
	}

	vm.Set("kinde", vm.NewObject())
	for name, binding := range workflow.Bindings.KindeAPIs {
		kindeMountPoint := vm.Get("kinde").(*goja.Object)
		if apiFunc, ok := kindeAPIs[name]; ok {
			kindeMountPoint.Set(name, runner.callRegisteredAPI(binding, apiFunc))
		}
	}

	workflowHash := workflow.GetHash()
	program, err := runner.Cache.cacheProgram(workflowHash, func() (*goja.Program, error) {
		ast, err := goja.Parse("main", string(workflow.ProcessedSource.Source))

		if err != nil {
			return nil, fmt.Errorf("error parsing %w", err)
		}

		program, err := goja.CompileAST(ast, false)

		if err != nil {
			return nil, fmt.Errorf("error compiling %w", err)
		}

		return program, nil

	})

	if err != nil {
		return nil, err
	}

	_, err = vm.RunProgram(program)
	if err != nil {
		return nil, fmt.Errorf("%v", err.Error())
	}
	return executionResult, nil
}

func (*gojaRunnerV1) maxExecutionTimeout(ctx context.Context, vm *goja.Runtime, maxExecutionDuration time.Duration) {
	go func() {
		timer := time.NewTimer(maxExecutionDuration)
		select {
		case <-ctx.Done():
			vm.Interrupt("execution time exceeded")
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
			vm.Interrupt("execution time exceeded")
		}
	}()
}

func (*gojaRunnerV1) consoleEmulation(_ *goja.Runtime, mountingPoint *goja.Object, result *actionResult, _ runtimesRegistry.ModuleBinding) {

	infoFunc := func(arguments ...interface{}) (interface{}, error) {
		result.ConsoleLog = append(result.ConsoleLog, arguments)
		return arguments, nil
	}

	errorFunc := func(arguments ...interface{}) (interface{}, error) {
		result.ConsoleError = append(result.ConsoleError, arguments)
		return arguments, nil
	}

	mountingPoint.Set("log", infoFunc)
	mountingPoint.Set("info", infoFunc)
	mountingPoint.Set("debug", infoFunc)
	mountingPoint.Set("warn", errorFunc)
	mountingPoint.Set("error", errorFunc)
}

func (*gojaRunnerV1) callRegisteredAPI(binding runtimesRegistry.ModuleBinding, registeredFunc func(runtimesRegistry.ModuleBinding, ...interface{}) (interface{}, error)) func(...interface{}) (interface{}, error) {

	wrapped := func(args ...interface{}) (interface{}, error) {
		result, err := registeredFunc(binding, args...)
		return result, err
	}
	return wrapped
}

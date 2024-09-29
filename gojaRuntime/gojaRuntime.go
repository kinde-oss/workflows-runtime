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
	nativeModules struct {
		registered map[string]*NativeModule
	}

	GojaRunnerV1 struct {
		cache         *gojaCache
		nativeModules nativeModules
	}

	actionResult struct {
		ConsoleLog   []interface{} `json:"console_log"`
		ConsoleError []interface{} `json:"console_error"`
		Context      jsContext     `json:"context"`
		ExitResult   interface{}   `json:"exit_result"`
	}
	introspectedExport struct {
		value interface{}
	}

	introspectionResult struct {
		exports map[string]introspectedExport
	}

	JsContext interface {
		SetValue(key string, value interface{})
	}
	jsContext struct {
		data map[string]interface{}
	}
)

// SetValue implements JsContext.
func (j jsContext) SetValue(key string, value interface{}) {
	j.data[key] = value
}

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
func (a *actionResult) GetContext() runtimesRegistry.RuntimeContext {
	return a.Context
}

// GetValue implements runtimesRegistry.RuntimeContext.
func (j jsContext) GetValues() map[string]interface{} {
	return j.data
}

// GetValueAsMap implements runtime_registry.RuntimeContext.
func (j jsContext) GetValueAsMap(key string) (map[string]interface{}, error) {
	if value, ok := j.data[key]; ok {
		switch v := value.(type) {
		case map[string]interface{}:
			return v, nil
		default:
			return nil, fmt.Errorf("value is not a map")
		}
	}
	return nil, fmt.Errorf("key not found")
}

func (a *actionResult) GetExitResult() interface{} {
	return a.ExitResult
}

var registry = new(require.Registry)

var builtInModules = map[string]func(e *GojaRunnerV1, vm *goja.Runtime, mountingPoint *goja.Object, result *actionResult, binding runtimesRegistry.ModuleBinding){
	"console": func(runner *GojaRunnerV1, vm *goja.Runtime, mountingPoint *goja.Object, result *actionResult, binding runtimesRegistry.ModuleBinding) {
		vm.Set("console", vm.NewObject())
		consoleMountingPoint := vm.Get("console").(*goja.Object)
		runner.consoleEmulation(vm, consoleMountingPoint, result, binding)
	},
	"url": func(e *GojaRunnerV1, vm *goja.Runtime, mountingPoint *goja.Object, _ *actionResult, _ runtimesRegistry.ModuleBinding) {
		vm.Set("url", require.Require(vm, "url"))
	},
	"module": func(e *GojaRunnerV1, vm *goja.Runtime, mountingPoint *goja.Object, _ *actionResult, _ runtimesRegistry.ModuleBinding) {
		vm.Set("module", vm.NewObject())
	},
}

func init() {
	runtimesRegistry.RegisterRuntime("goja", newGojaRunner)
	registry.RegisterNativeModule("url", urlModule.Require)
}

func newGojaRunner() runtimesRegistry.Runner {
	runner := GojaRunnerV1{
		cache: &gojaCache{
			cache: map[string]*goja.Program{},
		},
		nativeModules: nativeModules{
			registered: map[string]*NativeModule{},
		},
	}
	return &runner
}

func (nm *NativeModule) setupModuleForVM(vm *goja.Runtime, actionResult *actionResult, parent *goja.Object, requestedName string, binding runtimesRegistry.ModuleBinding) {
	for _, name := range strings.Split(requestedName, ".")[:1] {
		if function, ok := nm.functions[name]; ok {
			wrappedFunc := func(binding runtimesRegistry.ModuleBinding, jsContext jsContext) func(args ...interface{}) (interface{}, error) {
				return func(args ...interface{}) (interface{}, error) {
					return function(binding, jsContext, args...)
				}
			}
			parent.Set(name, vm.ToValue(wrappedFunc(binding, actionResult.Context)))
		}

		if name == "" {
			for fname, function := range nm.functions {

				wrappedFunc := func(binding runtimesRegistry.ModuleBinding, jsContext jsContext) func(args ...interface{}) (interface{}, error) {
					return func(args ...interface{}) (interface{}, error) {
						return function(binding, jsContext, args...)
					}
				}

				parent.Set(fname, vm.ToValue(wrappedFunc(binding, actionResult.Context)))
			}
			return
		}

		if module, ok := nm.modules[name]; ok {
			registeredModule := vm.NewObject()
			parent.Set(name, registeredModule)
			module.setupModuleForVM(vm, actionResult, registeredModule, strings.Join(strings.Split(requestedName, ".")[1:], "."), binding)
		}
	}
}

func (nm *nativeModules) setupModuleForVM(vm *goja.Runtime, actionResult *actionResult, requestedName string, binding runtimesRegistry.ModuleBinding) {

	for _, name := range strings.Split(requestedName, ".")[:1] {
		if module, ok := nm.registered[name]; ok {
			registeredModule := vm.Get(name)
			if registeredModule == nil {
				registeredModule = vm.NewObject()
				vm.Set(name, registeredModule)
			}
			module.setupModuleForVM(vm, actionResult, registeredModule.(*goja.Object), strings.Join(strings.Split(requestedName, ".")[1:], "."), binding)
		}
	}

}

type NativeModule struct {
	functions map[string]func(binding runtimesRegistry.ModuleBinding, jsContext JsContext, args ...interface{}) (interface{}, error)
	modules   map[string]*NativeModule
	name      string
}

func (runner GojaRunnerV1) RegisterNativeAPI(name string) *NativeModule {
	result := &NativeModule{
		functions: map[string]func(binding runtimesRegistry.ModuleBinding, jsContext JsContext, args ...interface{}) (interface{}, error){},
		modules:   map[string]*NativeModule{},
		name:      name,
	}
	runner.nativeModules.registered[name] = result
	return result
}

func (module *NativeModule) RegisterNativeFunction(name string, fn func(binding runtimesRegistry.ModuleBinding, jsContext JsContext, args ...interface{}) (interface{}, error)) {

	module.functions[name] = fn
}

func (module *NativeModule) RegisterNativeAPI(name string) *NativeModule {
	result := &NativeModule{
		functions: map[string]func(binding runtimesRegistry.ModuleBinding, jsContext JsContext, args ...interface{}) (interface{}, error){},
		modules:   map[string]*NativeModule{},
		name:      name,
	}
	module.modules[name] = result
	return result
}

// Introspect implements runtime_registry.Runner.
func (e *GojaRunnerV1) Introspect(ctx context.Context, workflow runtimesRegistry.WorkflowDescriptor, options runtimesRegistry.IntrospectionOptions) (runtimesRegistry.IntrospectionResult, error) {
	vm := goja.New()
	_, returnErr := e.setupVM(ctx, vm, workflow)

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

func (e *GojaRunnerV1) Execute(ctx context.Context, workflow runtimesRegistry.WorkflowDescriptor, startOptions runtimesRegistry.StartOptions) (runtimesRegistry.ExecutionResult, error) {

	vm := goja.New()
	executionResult, returnErr := e.setupVM(ctx, vm, workflow)

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

func (runner *GojaRunnerV1) setupVM(ctx context.Context, vm *goja.Runtime, workflow runtimesRegistry.WorkflowDescriptor) (*actionResult, error) {
	registry.Enable(vm)

	runner.maxExecutionTimeout(ctx, vm, workflow.Limits.MaxExecutionDuration)
	vm.SetTimeSource(func() time.Time { return time.Now() })

	executionResult := &actionResult{
		ConsoleLog:   []interface{}{},
		ConsoleError: []interface{}{},
		Context: jsContext{
			data: map[string]interface{}{},
		},
	}

	for name, binding := range workflow.RequestedBindings.Global {
		if module, ok := builtInModules[name]; ok {
			module(runner, vm, vm.NewObject(), executionResult, binding)
		}
	}

	for requestedName, requestedBinding := range workflow.RequestedBindings.Native {
		runner.nativeModules.setupModuleForVM(vm, executionResult, requestedName, requestedBinding)
	}

	workflowHash := workflow.GetHash()
	program, err := runner.cache.cacheProgram(workflowHash, func() (*goja.Program, error) {
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

func (*GojaRunnerV1) maxExecutionTimeout(ctx context.Context, vm *goja.Runtime, maxExecutionDuration time.Duration) {
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

func (*GojaRunnerV1) consoleEmulation(_ *goja.Runtime, mountingPoint *goja.Object, result *actionResult, _ runtimesRegistry.ModuleBinding) {

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

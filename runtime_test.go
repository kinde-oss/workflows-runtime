package workflows_runtime

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"

	gojaRuntime "github.com/kinde-oss/workflows-runtime/gojaRuntime"
	url "github.com/kinde-oss/workflows-runtime/gojaRuntime/url"
	projectBundler "github.com/kinde-oss/workflows-runtime/projectBundler"
	registry "github.com/kinde-oss/workflows-runtime/registry"
)

const testContextValue contextValue = "contextValue"
const testContextValueVm contextValue = "contextValue2"

type contextValue string

type workflowSettings struct {
	ID string `json:"id"`
}

type pageSettings struct {
}

func Test_GojaLogVmInits(t *testing.T) {
	setupWasCalled := false
	gojaRuntime.BeforeVMSetupFunc(func(ctx context.Context, vm *goja.Runtime) context.Context {
		return context.WithValue(ctx, testContextValueVm, vm)
	})
	gojaRuntime.AfterVMSetupFunc(func(ctx context.Context, vm *goja.Runtime) {
		setupWasCalled = true
	})

	gojaRuntime.RegisterNativeAPI("test").RegisterNativeFunction("test2", func(ctx context.Context, binding registry.BindingSettings, jsContext gojaRuntime.JsContext, args ...interface{}) (interface{}, error) {
		if ctx.Value(testContextValueVm) == nil {
			panic("test context value not found")
		}

		searchParams := args[0].(*url.UrlSearchParams)
		if (searchParams.SearchParams[0] != url.SearchParam{Name: "grant_type", Value: "client_credentials"}) {
			panic("search params not as expected")
		}
		if (searchParams.SearchParams[1] != url.SearchParam{Name: "client_id", Value: "123"}) {
			panic("search params not as expected")
		}
		if (searchParams.SearchParams[2] != url.SearchParam{Name: "client_secret", Value: "456"}) {
			panic("search params not as expected")
		}

		return nil, nil
	})

	runner := getGojaRunner()
	_, err := runner.Execute(context.Background(), registry.WorkflowDescriptor{
		Limits: registry.RuntimeLimits{
			MaxExecutionDuration: 30 * time.Second,
		}, ProcessedSource: registry.SourceDescriptor{
			Source: []byte(`
				var r=Object.defineProperty;var a=Object.getOwnPropertyDescriptor;var s=Object.getOwnPropertyNames;var g=Object.prototype.hasOwnProperty;var f=(t,e)=>{for(var n in e)r(t,n,{get:e[n],enumerable:!0})},i=(t,e,n,l)=>{if(e&&typeof e=="object"||typeof e=="function")for(let o of s(e))!g.call(t,o)&&o!==n&&r(t,o,{get:()=>e[o],enumerable:!(l=a(e,o))||l.enumerable});return t};var u=t=>i(r({},"__esModule",{value:!0}),t);var h={};
				f(h,{default:()=>c,workflowSettings:()=>w});module.exports=u(h);const w={resetClaims:!0};
				var c={async handle(t){
					return console.log("logging from workflow"), test.test2(new URLSearchParams({
						grant_type: 'client_credentials',
						client_id: '123',
						client_secret: '456'
					})
			); }
				};
			`),
			SourceType: registry.Source_ContentType_Text,
		},
		RequestedBindings: map[string]registry.BindingSettings{
			"console":           {},
			"url":               {},
			"kinde.fetch":       {},
			"kinde.idToken":     {},
			"kinde.accessToken": {},
			"test":              {},
		},
	}, registry.StartOptions{
		EntryPoint: "handle",
	})
	assert.True(t, setupWasCalled)
	assert.Nil(t, err)
}

func Test_GojaPrecompiledRuntime(t *testing.T) {
	runner := getGojaRunner()

	for i := range 2 {

		workflowRuncontext := context.WithValue(context.Background(), testContextValue, "test"+fmt.Sprint(i))

		result, err := runner.Execute(workflowRuncontext, registry.WorkflowDescriptor{
			Limits: registry.RuntimeLimits{
				MaxExecutionDuration: 30 * time.Second,
			},
			ProcessedSource: registry.SourceDescriptor{
				Source: []byte(`
				var r=Object.defineProperty;var a=Object.getOwnPropertyDescriptor;var s=Object.getOwnPropertyNames;var g=Object.prototype.hasOwnProperty;var f=(t,e)=>{for(var n in e)r(t,n,{get:e[n],enumerable:!0})},i=(t,e,n,l)=>{if(e&&typeof e=="object"||typeof e=="function")for(let o of s(e))!g.call(t,o)&&o!==n&&r(t,o,{get:()=>e[o],enumerable:!(l=a(e,o))||l.enumerable});return t};var u=t=>i(r({},"__esModule",{value:!0}),t);var h={};
				f(h,{default:()=>c,workflowSettings:()=>w});module.exports=u(h);const w={resetClaims:!0};
				var c={async handle(t){
					return console.log("logging from workflow",{balh:"blah"}), console.error("error"), kinde.idToken.setCustomClaim('aaa', 'bbb'), kinde.accessToken.setCustomClaim('ccc', 'ddd'), kinde.fetch("hello.com")}
				};//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsibWFpbiJdLAogICJzb3VyY2VzQ29udGVudCI6IFsiXG5cbiAgICAgICAgICAgICAgICAgICAgZXhwb3J0IGNvbnN0IHdvcmtmbG93U2V0dGluZ3MgPSB7XG4gICAgICAgICAgICAgICAgICAgICAgICByZXNldENsYWltczogdHJ1ZVxuICAgICAgICAgICAgICAgICAgICB9O1xuXG5cdFx0XHRcdFx0ZXhwb3J0IGRlZmF1bHQge1xuICAgICAgICAgICAgICAgICAgICAgICAgYXN5bmMgaGFuZGxlKGV2ZW50OiBhbnkpIHtcbiAgICAgICAgICAgICAgICAgICAgICAgICAgICBjb25zb2xlLmxvZygnbG9nZ2luZyBmcm9tIHdvcmtmbG93Jywge1wiYmFsaFwiOiBcImJsYWhcIn0pO1xuICAgICAgICAgICAgICAgICAgICAgICAgICAgIHJldHVybiAndGVzdGluZyByZXR1cm4nO1xuICAgICAgICAgICAgICAgICAgICAgICAgfVxuXG4gICAgICAgICAgICAgICAgICAgIH1cbiAgICAgICAgICAgICAgICAgICAgIl0sCiAgIm1hcHBpbmdzIjogIjRaQUFBLElBQUFBLEVBQUEsR0FBQUMsRUFBQUQsRUFBQSxhQUFBRSxFQUFBLHFCQUFBQyxJQUFBLGVBQUFDLEVBQUFKLEdBRTJCLE1BQU1HLEVBQW1CLENBQzVCLFlBQWEsRUFDakIsRUFFZixJQUFPRCxFQUFRLENBQ0ksTUFBTSxPQUFPRyxFQUFZLENBQ3JCLGVBQVEsSUFBSSx3QkFBeUIsQ0FBQyxLQUFRLE1BQU0sQ0FBQyxFQUM5QyxnQkFDWCxDQUVKIiwKICAibmFtZXMiOiBbIm1haW5fZXhwb3J0cyIsICJfX2V4cG9ydCIsICJtYWluX2RlZmF1bHQiLCAid29ya2Zsb3dTZXR0aW5ncyIsICJfX3RvQ29tbW9uSlMiLCAiZXZlbnQiXQp9Cg=="}
			`),
				SourceType: registry.Source_ContentType_Text,
			},
			RequestedBindings: map[string]registry.BindingSettings{
				"console":           {},
				"url":               {},
				"kinde.fetch":       {},
				"kinde.idToken":     {},
				"kinde.accessToken": {},
			},
		}, registry.StartOptions{
			EntryPoint: "handle",
		})

		assert := assert.New(t)
		assert.Nil(err)
		assert.Equal("fetch response", fmt.Sprintf("%v", result.GetExitResult()))

		idTokenMap, err := result.GetContext().GetValueAsMap("idToken")
		assert.Nil(err)
		assert.Equal("bbb", idTokenMap["aaa"])
		assert.Greater(result.ExecutionMetadata().ExecutionDuration.Nanoseconds(), int64(1))
		assert.False(result.ExecutionMetadata().StartedAt.IsZero())
	}
}

func Test_GojaPrecompiledRuntimeTimeout(t *testing.T) {
	runner := getGojaRunner()

	for i := 0; i < 2; i++ {

		workflowRuncontext := context.WithValue(context.Background(), testContextValue, "test"+fmt.Sprint(i))

		result, err := runner.Execute(workflowRuncontext, registry.WorkflowDescriptor{
			Limits: registry.RuntimeLimits{
				MaxExecutionDuration: 2 * time.Second,
			},
			ProcessedSource: registry.SourceDescriptor{
				Source: []byte(`

				var r=Object.defineProperty;var a=Object.getOwnPropertyDescriptor;var s=Object.getOwnPropertyNames;var g=Object.prototype.hasOwnProperty;var f=(t,e)=>{for(var n in e)r(t,n,{get:e[n],enumerable:!0})},i=(t,e,n,l)=>{if(e&&typeof e=="object"||typeof e=="function")for(let o of s(e))!g.call(t,o)&&o!==n&&r(t,o,{get:()=>e[o],enumerable:!(l=a(e,o))||l.enumerable});return t};var u=t=>i(r({},"__esModule",{value:!0}),t);var h={};
				f(h,{default:()=>c,workflowSettings:()=>w});module.exports=u(h);const w={resetClaims:!0};
				var c={async handle(t){
				while (true) {
				}
					return console.log("logging from workflow",{balh:"blah"}), console.error("error"), kinde.idToken.setCustomClaim('aaa', 'bbb'), kinde.accessToken.setCustomClaim('ccc', 'ddd'), kinde.fetch("hello.com")}
				};//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsibWFpbiJdLAogICJzb3VyY2VzQ29udGVudCI6IFsiXG5cbiAgICAgICAgICAgICAgICAgICAgZXhwb3J0IGNvbnN0IHdvcmtmbG93U2V0dGluZ3MgPSB7XG4gICAgICAgICAgICAgICAgICAgICAgICByZXNldENsYWltczogdHJ1ZVxuICAgICAgICAgICAgICAgICAgICB9O1xuXG5cdFx0XHRcdFx0ZXhwb3J0IGRlZmF1bHQge1xuICAgICAgICAgICAgICAgICAgICAgICAgYXN5bmMgaGFuZGxlKGV2ZW50OiBhbnkpIHtcbiAgICAgICAgICAgICAgICAgICAgICAgICAgICBjb25zb2xlLmxvZygnbG9nZ2luZyBmcm9tIHdvcmtmbG93Jywge1wiYmFsaFwiOiBcImJsYWhcIn0pO1xuICAgICAgICAgICAgICAgICAgICAgICAgICAgIHJldHVybiAndGVzdGluZyByZXR1cm4nO1xuICAgICAgICAgICAgICAgICAgICAgICAgfVxuXG4gICAgICAgICAgICAgICAgICAgIH1cbiAgICAgICAgICAgICAgICAgICAgIl0sCiAgIm1hcHBpbmdzIjogIjRaQUFBLElBQUFBLEVBQUEsR0FBQUMsRUFBQUQsRUFBQSxhQUFBRSxFQUFBLHFCQUFBQyxJQUFBLGVBQUFDLEVBQUFKLEdBRTJCLE1BQU1HLEVBQW1CLENBQzVCLFlBQWEsRUFDakIsRUFFZixJQUFPRCxFQUFRLENBQ0ksTUFBTSxPQUFPRyxFQUFZLENBQ3JCLGVBQVEsSUFBSSx3QkFBeUIsQ0FBQyxLQUFRLE1BQU0sQ0FBQyxFQUM5QyxnQkFDWCxDQUVKIiwKICAibmFtZXMiOiBbIm1haW5fZXhwb3J0cyIsICJfX2V4cG9ydCIsICJtYWluX2RlZmF1bHQiLCAid29ya2Zsb3dTZXR0aW5ncyIsICJfX3RvQ29tbW9uSlMiLCAiZXZlbnQiXQp9Cg=="}
			`),
				SourceType: registry.Source_ContentType_Text,
			},
			RequestedBindings: map[string]registry.BindingSettings{
				"console":           {},
				"url":               {},
				"kinde.fetch":       {},
				"kinde.idToken":     {},
				"kinde.accessToken": {},
			},
		}, registry.StartOptions{
			EntryPoint: "handle",
		})

		assert := assert.New(t)
		assert.NotNil(err)
		assert.Greater(result.ExecutionMetadata().ExecutionDuration.Nanoseconds(), int64(1))
		assert.False(result.ExecutionMetadata().StartedAt.IsZero())
	}
}

func Test_ProjectBunlerE2E(t *testing.T) {
	somePathInsideProject, _ := filepath.Abs("./testData/kindeSrc/environment/workflows") //starting in a middle of nowhere, so we need to go up to the root of the project

	projectBundler := projectBundler.NewProjectBundler(projectBundler.DiscoveryOptions[workflowSettings, pageSettings]{
		StartFolder: somePathInsideProject,
	})

	kindeProject, discoveryError := projectBundler.Discover(context.Background())

	assert := assert.New(t)

	if !assert.Nil(discoveryError) {
		t.FailNow()
	}
	assert.Equal("2024-12-09", kindeProject.Configuration.Version)
	assert.Equal("kindeSrc", kindeProject.Configuration.RootDir)
	assert.Equal(3, len(kindeProject.Environment.Workflows))
	assert.Empty(kindeProject.Environment.Workflows[0].Bundle.Errors)
	assert.Empty(kindeProject.Environment.Workflows[1].Bundle.Errors)
	assert.Empty(kindeProject.Environment.Workflows[2].Bundle.Errors)

	for _, workflow := range kindeProject.Environment.Workflows {
		t.Run(fmt.Sprintf("Test_ExecuteWorkflowWithGoja - %v", workflow.WorkflowRootDirectory), testExecution(workflow, assert))
	}

}

func testExecution(workflow projectBundler.KindeWorkflow[workflowSettings], assert *assert.Assertions) func(t *testing.T) {
	return func(t *testing.T) {
		runner := getGojaRunner()
		logger := testLogger{}
		workflowRuncontext := context.WithValue(context.Background(), testContextValue, "test")
		result, err := runner.Execute(workflowRuncontext, registry.WorkflowDescriptor{
			Limits: registry.RuntimeLimits{
				MaxExecutionDuration: 30 * time.Second,
			},
			ProcessedSource: registry.SourceDescriptor{
				Source:     workflow.Bundle.Content.Source,
				SourceType: registry.Source_ContentType_Text,
			},
			RequestedBindings: workflow.Bundle.Content.Settings.Bindings,
		}, registry.StartOptions{
			EntryPoint: "handle",
			Loggger:    &logger,
		})

		if !assert.Nil(err) {
			t.FailNow()
		}
		assert.Equal("testing return", fmt.Sprintf("%v", result.GetExitResult()))

		idTokenMap, err := result.GetContext().GetValueAsMap("idToken")
		assert.Nil(err)
		assert.Equal("test", idTokenMap["random"])

		accessTokenMap, err := result.GetContext().GetValueAsMap("accessToken")
		assert.Nil(err)
		assert.NotNil(accessTokenMap["test2"])

		logMessage := logger.info.([]interface{})[0].(string)
		assert.Equal("logging from action", logMessage)
	}
}

var contextRead = 0

func getGojaRunner() registry.Runner {
	runtime, _ := GetRuntime("goja")

	kindeAPI := gojaRuntime.RegisterNativeAPI("kinde")
	kindeAPI.RegisterNativeFunction("fetch", func(ctx context.Context, binding registry.BindingSettings, jsContext gojaRuntime.JsContext, args ...interface{}) (interface{}, error) {
		return "fetch response", nil
	})

	kindeAPI.RegisterNativeAPI("idToken").RegisterNativeFunction("setCustomClaim", func(ctx context.Context, binding registry.BindingSettings, jsContext gojaRuntime.JsContext, args ...interface{}) (interface{}, error) {

		testValue, ok := ctx.Value(testContextValue).(string)
		if !ok {
			return nil, fmt.Errorf("context value not found")
		}
		if testValue != "test"+fmt.Sprint(contextRead) && testValue != "test" {
			return nil, fmt.Errorf("context value not as expected")
		}
		contextRead++

		if len(args) != 2 {
			return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
		}
		name, ok1 := args[0].(string)
		if !ok1 {
			return nil, fmt.Errorf("first argument must be string")
		}
		jsContext.SetValue("idToken", map[string]interface{}{name: args[1]})
		return nil, nil
	})

	kindeAPI.RegisterNativeAPI("accessToken").RegisterNativeFunction("setCustomClaim", func(ctx context.Context, binding registry.BindingSettings, jsContext gojaRuntime.JsContext, args ...interface{}) (interface{}, error) {

		if len(args) != 2 {
			return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
		}
		name, ok1 := args[0].(string)
		if !ok1 {
			return nil, fmt.Errorf("first argument must be string")
		}
		at := jsContext.GetValue("accessToken")
		if at == nil {
			at = make(map[string]interface{})
		}
		at.(map[string]interface{})[name] = args[1]

		jsContext.SetValue("accessToken", at)
		return nil, nil
	})
	return runtime
}

type testLogger struct {
	info    interface{}
	debug   interface{}
	err     interface{}
	warning interface{}
}

func (l *testLogger) Log(level registry.LogLevel, args ...interface{}) {
	switch level {
	case registry.LogLevelDebug:
		l.debug = args
	case registry.LogLevelError:
		l.err = args
	case registry.LogLevelInfo:
		l.info = args
	case registry.LogLevelWarning:
		l.warning = args
	default:
		panic(fmt.Sprintf("unexpected runtime_registry.LogLevel: %#v", level))
	}
}

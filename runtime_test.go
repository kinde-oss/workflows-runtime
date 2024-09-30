package workflows_runtime

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	gojaRuntime "github.com/kinde-oss/workflows-runtime/gojaRuntime"
	projectBundler "github.com/kinde-oss/workflows-runtime/projectBundler"
	registry "github.com/kinde-oss/workflows-runtime/registry"
)

func Test_GojaPrecompiledRuntime(t *testing.T) {
	runner := getGojaRunner()

	for i := 0; i < 2; i++ {

		result, err := runner.Execute(context.Background(), registry.WorkflowDescriptor{
			Limits: registry.RuntimeLimits{
				MaxExecutionDuration: 30 * time.Second,
			},
			ProcessedSource: registry.SourceDescriptor{
				Source: []byte(`
				var r=Object.defineProperty;var a=Object.getOwnPropertyDescriptor;var s=Object.getOwnPropertyNames;var g=Object.prototype.hasOwnProperty;var f=(t,e)=>{for(var n in e)r(t,n,{get:e[n],enumerable:!0})},i=(t,e,n,l)=>{if(e&&typeof e=="object"||typeof e=="function")for(let o of s(e))!g.call(t,o)&&o!==n&&r(t,o,{get:()=>e[o],enumerable:!(l=a(e,o))||l.enumerable});return t};var u=t=>i(r({},"__esModule",{value:!0}),t);var h={};
				f(h,{default:()=>c,workflowSettings:()=>w});module.exports=u(h);const w={resetClaims:!0};
				var c={async handle(t){
					return console.log("logging from workflow",{balh:"blah"}), console.error("error"), kinde.idToken.setCustomClaim('aaa', 'bbb'), kinde.fetch("hello.com")}
				};//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsibWFpbiJdLAogICJzb3VyY2VzQ29udGVudCI6IFsiXG5cbiAgICAgICAgICAgICAgICAgICAgZXhwb3J0IGNvbnN0IHdvcmtmbG93U2V0dGluZ3MgPSB7XG4gICAgICAgICAgICAgICAgICAgICAgICByZXNldENsYWltczogdHJ1ZVxuICAgICAgICAgICAgICAgICAgICB9O1xuXG5cdFx0XHRcdFx0ZXhwb3J0IGRlZmF1bHQge1xuICAgICAgICAgICAgICAgICAgICAgICAgYXN5bmMgaGFuZGxlKGV2ZW50OiBhbnkpIHtcbiAgICAgICAgICAgICAgICAgICAgICAgICAgICBjb25zb2xlLmxvZygnbG9nZ2luZyBmcm9tIHdvcmtmbG93Jywge1wiYmFsaFwiOiBcImJsYWhcIn0pO1xuICAgICAgICAgICAgICAgICAgICAgICAgICAgIHJldHVybiAndGVzdGluZyByZXR1cm4nO1xuICAgICAgICAgICAgICAgICAgICAgICAgfVxuXG4gICAgICAgICAgICAgICAgICAgIH1cbiAgICAgICAgICAgICAgICAgICAgIl0sCiAgIm1hcHBpbmdzIjogIjRaQUFBLElBQUFBLEVBQUEsR0FBQUMsRUFBQUQsRUFBQSxhQUFBRSxFQUFBLHFCQUFBQyxJQUFBLGVBQUFDLEVBQUFKLEdBRTJCLE1BQU1HLEVBQW1CLENBQzVCLFlBQWEsRUFDakIsRUFFZixJQUFPRCxFQUFRLENBQ0ksTUFBTSxPQUFPRyxFQUFZLENBQ3JCLGVBQVEsSUFBSSx3QkFBeUIsQ0FBQyxLQUFRLE1BQU0sQ0FBQyxFQUM5QyxnQkFDWCxDQUVKIiwKICAibmFtZXMiOiBbIm1haW5fZXhwb3J0cyIsICJfX2V4cG9ydCIsICJtYWluX2RlZmF1bHQiLCAid29ya2Zsb3dTZXR0aW5ncyIsICJfX3RvQ29tbW9uSlMiLCAiZXZlbnQiXQp9Cg=="}
			`),
				SourceType: registry.Source_ContentType_Text,
			},
			RequestedBindings: map[string]registry.BindingSettings{
				"console":       {},
				"url":           {},
				"kinde.fetch":   {},
				"kinde.idToken": {},
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

func Test_ProjectBunlerE2E(t *testing.T) {
	somePathInsideProject, _ := filepath.Abs("./testData/kindeSrc/environment/workflows") //starting in a middle of nowhere, so we need to go up to the root of the project

	projectBundler := projectBundler.NewProjectBundler(projectBundler.DiscoveryOptions{
		StartFolder: somePathInsideProject,
	})

	kindeProject, discoveryError := projectBundler.Discover()

	assert := assert.New(t)

	if !assert.Nil(discoveryError) {
		t.FailNow()
	}
	assert.Equal("2024-12-09", kindeProject.Configuration.Version)
	assert.Equal("kindeSrc", kindeProject.Configuration.RootDir)
	assert.Equal(2, len(kindeProject.Environment.Workflows))
	assert.Empty(kindeProject.Environment.Workflows[0].Bundle.Errors)
	assert.Empty(kindeProject.Environment.Workflows[1].Bundle.Errors)

	for _, workflow := range kindeProject.Environment.Workflows {
		t.Run(fmt.Sprintf("Test_ExecuteWorkflowWithGoja - %v", workflow.WorkflowRootDirectory), testExecution(workflow, assert))
	}

}

func testExecution(workflow projectBundler.KindeWorkflow, assert *assert.Assertions) func(t *testing.T) {
	return func(t *testing.T) {
		runner := getGojaRunner()
		result, err := runner.Execute(context.Background(), registry.WorkflowDescriptor{
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
	}
}

func getGojaRunner() registry.Runner {
	runtime, _ := GetRuntime("goja")

	kindeAPI := gojaRuntime.RegisterNativeAPI("kinde")
	kindeAPI.RegisterNativeFunction("fetch", func(binding registry.BindingSettings, jsContext gojaRuntime.JsContext, args ...interface{}) (interface{}, error) {
		return "fetch response", nil
	})

	kindeAPI.RegisterNativeAPI("idToken").RegisterNativeFunction("setCustomClaim", func(binding registry.BindingSettings, jsContext gojaRuntime.JsContext, args ...interface{}) (interface{}, error) {
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

	kindeAPI.RegisterNativeAPI("accessToken").RegisterNativeFunction("setCustomClaim", func(binding registry.BindingSettings, jsContext gojaRuntime.JsContext, args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
		}
		name, ok1 := args[0].(string)
		if !ok1 {
			return nil, fmt.Errorf("first argument must be string")
		}
		jsContext.SetValue("accessToken", map[string]interface{}{name: args[1]})
		return nil, nil
	})
	return runtime
}

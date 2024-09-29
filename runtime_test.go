package workflows_runtime

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	goja_runtime "github.com/kinde-oss/workflows-runtime/gojaRuntime"
	registry "github.com/kinde-oss/workflows-runtime/registry"
)

func Test_GojaRuntime(t *testing.T) {
	runtime, _ := GetRuntime("goja")

	kindeAPI := runtime.(*goja_runtime.GojaRunnerV1).RegisterNativeAPI("kinde")
	kindeAPI.RegisterNativeFunction("fetch", func(binding registry.ModuleBinding, jsContext goja_runtime.JsContext, args ...interface{}) (interface{}, error) {
		return "fetch response", nil
	})

	idTokenAPI := kindeAPI.RegisterNativeAPI("idToken")
	idTokenAPI.RegisterNativeFunction("setCustomClaim", func(binding registry.ModuleBinding, jsContext goja_runtime.JsContext, args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
		}
		name, ok1 := args[0].(string)
		value, ok2 := args[1].(string)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("arguments must be strings")
		}
		jsContext.SetValue("idToken", map[string]interface{}{name: value})
		return nil, nil
	})

	for i := 0; i < 2; i++ {

		result, err := runtime.Execute(context.Background(), registry.WorkflowDescriptor{
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
			RequestedBindings: registry.Bindings{
				Global: map[string]registry.ModuleBinding{
					"console": {},
					"url":     {},
					"module":  {},
				},
				Native: map[string]registry.ModuleBinding{
					"kinde.fetch":   {},
					"kinde.idToken": {},
				},
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
	}
}

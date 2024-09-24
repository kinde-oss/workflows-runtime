package builder

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_WorkflowBundler(t *testing.T) {

	workflowPath, _ := filepath.Abs("../testData/kindeSrc/environment/workflows/evTest")

	workflowBuilder := newWorkflowBundler(BundlerOptions{
		WorkingFolder: workflowPath,
		EntryPoints:   []string{"workflow.ts"},
	})
	bundlerResult := workflowBuilder.Bundle()

	assert := assert.New(t)
	assert.Nil(bundlerResult.Errors)
	assert.NotEmpty(bundlerResult.Bundle.Source)
	assert.Equal("tokenGen", bundlerResult.Bundle.Settings.ID)
}

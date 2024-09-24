package project_bundler

type (
	ProjectConfiguration struct {
		Version string `json:"version"`
		RootDir string `json:"root_dir"`
	}

	KindeWorkflow struct {
		WorkflowRootDirectory string `json:"workflow_root_directory"`
	}
	KindeWorkflows struct {
		Workflows []KindeWorkflow `json:"workflows"`
	}

	KindeEnvironment struct {
		Workflows KindeWorkflows `json:"workflows"`
	}

	KindeProject struct {
		Configuration ProjectConfiguration `json:"configuration"`
		Environment   KindeEnvironment     `json:"environment"`
	}

	ProjectBundler interface {
		Discover() KindeProject
	}
)

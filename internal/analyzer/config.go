package analyzer

import "fmt"

// AnalysisConfig represents the configuration for analysis
type AnalysisConfig struct {
	ProjectPath  string
	OutputPath   string
	RootFunction *FunctionId
}

// NewAnalysisConfig creates a new analysis configuration
func NewAnalysisConfig(projectPath, outputPath string, rootFunction *FunctionId) (*AnalysisConfig, error) {
	if projectPath == "" {
		return nil, fmt.Errorf("project path is required")
	}

	return &AnalysisConfig{
		ProjectPath:  projectPath,
		OutputPath:   outputPath,
		RootFunction: rootFunction,
	}, nil
}

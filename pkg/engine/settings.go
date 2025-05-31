package engine

import (
	"path/filepath"
)

type Settings struct {
	Var       map[string]any      `json:"vars,omitempty"`
	Commands  map[string][]string `json:"commands,omitempty"`
	OutputDir string              `json:"output_dir,omitempty"`
	InputDir  string              `json:"input_dir,omitempty"`
}

func (s Settings) StylePath(filename string) string {
	return filepath.Join(s.OutputDir, "style", filename)
}

func (s Settings) ScriptPath(filename string) string {
	return filepath.Join(s.OutputDir, "script", filename)
}

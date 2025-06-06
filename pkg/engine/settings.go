package engine

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

type ModuleType string

func (mt *ModuleType) UnmarshalJSON(raw []byte) error {
	switch strings.ToLower(string(raw)) {
	case "", "text/javascript":
		*mt = "text/javascript"
	case "ecma", "module":
		*mt = "module"
	default:
		return errors.New("Unsupported module type, allowed only 'text/javascript' (default) or 'ecma'")
	}
	return nil
}

type Settings struct {
	Var       map[string]any      `json:"vars,omitempty"`
	Commands  map[string][]string `json:"commands,omitempty"`
	OutputDir string              `json:"output_dir,omitempty"`
	InputDir  string              `json:"input_dir,omitempty"`
	Module    ModuleType          `json:"module,omitempty"`
}

func (s Settings) StylePath(filename string) string {
	return filepath.Join(s.OutputDir, "style", filename+".css")
}

func (s Settings) ScriptPath(filename string) string {
	return filepath.Join(s.OutputDir, "script", filename+".js")
}

func (s *Settings) StyleURL(name string) string {
	link, err := url.JoinPath("style", name+".css")
	if err != nil {
		panic(err)
	}
	return link
}

func (s *Settings) ScriptURL(name string) string {
	link, err := url.JoinPath("script", name+".js")
	if err != nil {
		panic(err)
	}
	return link
}

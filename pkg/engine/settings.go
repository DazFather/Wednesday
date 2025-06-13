package engine

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
)

type ModuleType string

const (
	ecmaModule ModuleType = "module"
	noModule   ModuleType = "text/javascript"
)

func ParseModuleType(val string) (ModuleType, error) {
	switch val = strings.Trim(val, `"'`); strings.ToLower(val) {
	case "", string(noModule):
		return noModule, nil
	case "ecma", string(ecmaModule):
		return ecmaModule, nil
	}
	return "", errors.New("Unsupported module type '" + val + "', allowed only 'text/javascript' (default) or 'ecma'")
}

func (mt *ModuleType) UnmarshalJSON(raw []byte) error {
	val, err := ParseModuleType(string(raw))
	if err == nil {
		*mt = val
	}
	return err
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

func (s Settings) StyleURL(name string) string {
	link, err := url.JoinPath("style", name+".css")
	if err != nil {
		panic(err)
	}
	return link
}

func (s Settings) ScriptURL(name string) string {
	link, err := url.JoinPath("script", name+".js")
	if err != nil {
		panic(err)
	}
	return link
}

func (s Settings) ScriptTag(name string, deferred bool, overrideModule *ModuleType) string {
	d := ""
	if deferred {
		d = "defer "
	}

	modType := s.Module
	if overrideModule != nil {
		modType = *overrideModule
	}

	return `<script ` + d + `type="` + string(modType) + `" src="` + s.ScriptURL(name) + `"></script>`
}

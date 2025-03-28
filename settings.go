package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
)

type Settings struct {
	Vars      map[string]string   `json:"vars"`
	Commands  map[string][]string `json:"commands"`
	OutputDir string              `json:"output_dir"`
	InputDir  string              `json:"input_dir"`
}

// NewSettingsFromJSON creates a new Settings instance from a JSON string.
func NewSettingsFromJSON(path string) (s Settings, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}

	if err = json.Unmarshal(b, &s); err == nil {
		s.OutputDir, s.InputDir = filepath.Clean(s.OutputDir), filepath.Clean(s.InputDir)
	}
	return
}

func LoadSettings(flags *flag.FlagSet, args []string, defaultValue string) (s Settings, err error) {
	path := flags.String("settings", defaultValue, "path for the settings json file")
	flags.StringVar(path, "s", defaultValue, "shorthand for 'settings'")

	if err = flags.Parse(args); err != nil {
		return
	}

	return NewSettingsFromJSON(filepath.Clean(*path))
}

func (s Settings) StylePath(filename string) string {
	return filepath.Join(s.OutputDir, "style", filename)
}

func (s Settings) ScriptPath(filename string) string {
	return filepath.Join(s.OutputDir, "script", filename)
}

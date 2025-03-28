package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Settings struct {
	Vars      map[string]string   `json:"vars",omitempty`
	Commands  map[string][]string `json:"commands",omitempty`
	OutputDir string              `json:"output_dir",omitempty`
	InputDir  string              `json:"input_dir",omitempty`
	from      string
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

func (s Settings) StylePath(filename string) string {
	return filepath.Join(s.OutputDir, "style", filename)
}

func (s Settings) ScriptPath(filename string) string {
	return filepath.Join(s.OutputDir, "script", filename)
}

func (s *Settings) Set(val string) error {
	var sets, err = NewSettingsFromJSON(val)
	*s = sets
	s.from = filepath.Clean(val)
	return err
}

func (s Settings) String() string {
	return s.from
}

type BaseFlags struct {
	Settings Settings
	*flag.FlagSet
}

func NewBaseFlags(name string, using ...func(*BaseFlags)) (bf BaseFlags) {
	bf.FlagSet = flag.NewFlagSet(name, flag.ExitOnError)
	bf.Var(&bf.Settings, "settings", "path for the settings json file")
	bf.Var(&bf.Settings, "s", "shorthand for 'settings'")

	for _, useFn := range using {
		useFn(&bf)
	}
	return
}

func (f *BaseFlags) Parse(arguments []string) (err error) {
	f.Usage = func() {
		_, name := filepath.Split(os.Args[0])
		fmt.Fprintf(os.Stderr, "Usage: %s %s [flags]\n\n", strings.TrimSuffix(name, ".exe"), f.Name())
		f.PrintDefaults()
		os.Exit(0)
	}

	if err = f.FlagSet.Parse(arguments); err != nil {
		return
	}

	if f.Settings.from == "" {
		if err := f.Settings.Set("wed-settings.json"); !os.IsNotExist(err) {
			return err
		}
		f.Settings.OutputDir = "build"
	}

	return
}

func portFlag(port *string, defPort string) func(*BaseFlags) {
	return func(f *BaseFlags) {
		f.StringVar(port, "port", defPort, "port for the local server")
		f.StringVar(port, "p", defPort, "shorthand for 'port'")
	}
}

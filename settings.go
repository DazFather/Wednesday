package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileSettings struct {
	Vars      map[string]string   `json:"vars,omitempty"`
	Commands  map[string][]string `json:"commands,omitempty"`
	OutputDir string              `json:"output_dir,omitempty"`
	InputDir  string              `json:"input_dir,omitempty"`
	from      string
}

type Settings struct {
	FileSettings
	port string
	arg  string
}

// NewSettingsFromJSON creates a new Settings instance from a JSON string.
func NewSettingsFromJSON(path string) (s FileSettings, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}

	if err = json.Unmarshal(b, &s); err == nil {
		s.OutputDir, s.InputDir = filepath.Clean(s.OutputDir), filepath.Clean(s.InputDir)
	}
	return
}

func (s FileSettings) StylePath(filename string) string {
	return filepath.Join(s.OutputDir, "style", filename)
}

func (s FileSettings) ScriptPath(filename string) string {
	return filepath.Join(s.OutputDir, "script", filename)
}

func (s *FileSettings) Set(spath string) (err error) {
	*s, err = NewSettingsFromJSON(spath)
	return
}

func (s FileSettings) String() string {
	return "wed-settings.json"
}

func serveFlags() (s Settings) {
	var f = flag.NewFlagSet("serve", flag.ContinueOnError)

	f.StringVar(&s.port, "port", ":8080", "port for the local server")
	f.StringVar(&s.port, "p", ":8080", "shorthand for 'port'")
	f.Var(&s.FileSettings, "settings", "path for the settings json file")
	f.Var(&s.FileSettings, "s", "shorthand for 'settings'")
	// TODO: replace this with the appropriate help message.
	f.Usage = func() { fmt.Println("TODO"); os.Exit(1) }

	if err := f.Parse(os.Args[2:]); err != nil {
		f.Usage()
	}

	s.port = strings.TrimSpace(s.port)
	if len(s.port) > 0 && s.port[0] != ':' {
		s.port = ":" + s.port
	}

	return
}

func initFlags() (s Settings) {
	var f = flag.NewFlagSet("init", flag.ContinueOnError)

	f.Var(&s.FileSettings, "settings", "path for the settings json file")
	f.Var(&s.FileSettings, "s", "shorthand for 'settings'")
	// TODO: replace this with the appropriate help message.
	f.Usage = func() { fmt.Println("TODO"); os.Exit(1) }

	if err := f.Parse(os.Args[2:]); err != nil {
		f.Usage()
	}
	return
}

func buildFlags() (s Settings) {
	var f = flag.NewFlagSet("build", flag.ExitOnError)

	f.Var(&s.FileSettings, "settings", "path for the settings json file")
	f.Var(&s.FileSettings, "s", "shorthand for 'settings'")
	// TODO: replace this with the appropriate help message.
	f.Usage = func() { fmt.Println("TODO"); os.Exit(1) }

	if err := f.Parse(os.Args[2:]); err != nil {
		f.Usage()
	}
	return
}

func runFlags() (s Settings) {
	var f = flag.NewFlagSet("run", flag.ExitOnError)

	f.Var(&s.FileSettings, "settings", "path for the settings json file")
	f.Var(&s.FileSettings, "s", "shorthand for 'settings'")
	// TODO: replace this with the appropriate help message.
	f.Usage = func() { fmt.Println("TODO"); os.Exit(1) }

	if err := f.Parse(os.Args[2:]); err != nil {
		f.Usage()
	}
	s.arg = f.Arg(0)
	return
}

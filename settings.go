package main

import (
	"encoding/json"
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type FileSettings struct {
	Var       map[string]any      `json:"vars,omitempty"`
	Commands  map[string][]string `json:"commands,omitempty"`
	OutputDir string              `json:"output_dir,omitempty"`
	InputDir  string              `json:"input_dir,omitempty"`
	from      string
}

type Settings struct {
	FileSettings
	reload  time.Duration
	port    string
	name    string
	tags    string
	arg     string
	noColor bool
}

// NewSettingsFromJSON creates a new Settings instance from a JSON string.
func NewSettingsFromJSON(path string) (s FileSettings, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		s.OutputDir = "build"
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
	s.from = spath
	return
}

func (s FileSettings) String() string {
	return s.from
}

func serveFlags() (s Settings) {
	var f = flag.NewFlagSet("serve", flag.ExitOnError)

	f.StringVar(&s.port, "port", ":8080", "port for the local server")
	f.StringVar(&s.port, "p", ":8080", "shorthand for 'port'")
	f.DurationVar(&s.reload, "live", 0, "reload server each time interval")
	f.DurationVar(&s.reload, "l", 0, "shorthand for 'live'")

	parseDefault(f, &s, os.Args[2:], serveUsage)

	s.port = strings.TrimSpace(s.port)
	if len(s.port) > 0 && s.port[0] != ':' {
		s.port = ":" + s.port
	}

	return
}

func initFlags() (s Settings) {
	parseDefault(
		flag.NewFlagSet("init", flag.ExitOnError),
		&s,
		os.Args[2:],
		initUsage,
	)

	if s.arg != "" {
		s.InputDir, s.OutputDir = s.arg, filepath.Join(s.arg, s.OutputDir)
	}

	return
}

func buildFlags() (s Settings) {
	parseDefault(
		flag.NewFlagSet("build", flag.ExitOnError),
		&s,
		os.Args[2:],
		buildUsage,
	)

	return
}

func runFlags() (s Settings) {
	parseDefault(
		flag.NewFlagSet("run", flag.ExitOnError),
		&s,
		os.Args[2:],
		runUsage,
	)

	return
}

func libUseFlag(args []string) (s Settings) {
	parseDefault(
		flag.NewFlagSet("lib use", flag.ExitOnError),
		&s,
		args,
		libUseUsage,
	)

	return
}

func libTrustFlag(args []string) (s Settings) {
	var f = flag.NewFlagSet("lib trust", flag.ExitOnError)

	f.StringVar(&s.name, "rename", "", "rename locally the trusted library")
	f.StringVar(&s.name, "n", "", "shorthand for 'rename'")

	parseDefault(f, &s, args, libTrustUsage)

	if s.name == "" {
		if url, err := cleanURL(s.arg); err == nil {
			s.name = cutExt(path.Base(url))
		} else {
			s.name = cutExt(filepath.Base(s.arg))
		}
	}

	return
}

func libSearchFlag(args []string) (s Settings) {
	var f = flag.NewFlagSet("lib search", flag.ExitOnError)

	insensitive := f.Bool("i", false, "insensitive case pattern matching")
	f.StringVar(&s.tags, "tags", "", "specify another pattern for tag matching")
	f.StringVar(&s.tags, "t", "", "shorthand for 'tags'")

	parseDefault(f, &s, args, libSearchUsage)

	if *insensitive {
		s.arg = "(?i)" + s.arg
		if s.tags != "" {
			s.tags = "(?i)" + s.tags
		}
	}

	return
}

func helpFlags() (s Settings) {
	var f = flag.NewFlagSet("help", flag.ExitOnError)

	f.BoolVar(&s.noColor, "no-color", false, "force disable of colored output'")
	f.BoolVar(&s.noColor, "nc", false, "shorthand for 'no-color'")
	f.Usage = func() { doHelp(s); os.Exit(1) }

	if err := f.Parse(os.Args[2:]); err != nil {
		f.Usage()
	}

	s.name, s.arg = f.Arg(0), f.Arg(1)

	return
}

func parseDefault(f *flag.FlagSet, s *Settings, args []string, usage func(bool)) {
	f.Var(&s.FileSettings, "settings", "path for the settings json file")
	f.Var(&s.FileSettings, "s", "shorthand for 'settings'")
	f.BoolVar(&s.noColor, "no-color", false, "force disable of colored output'")
	f.BoolVar(&s.noColor, "nc", false, "shorthand for 'no-color'")
	f.Usage = func() { usage(s.noColor); os.Exit(1) }

	if err := f.Parse(args); err != nil {
		f.Usage()
	}

	if s.from == "" {
		if err := s.FileSettings.Set("wed-settings.json"); err != nil && !os.IsNotExist(err) {
			panic(err)
		}
	}

	if s.arg == "" {
		s.arg = f.Arg(0)
	}

	return
}

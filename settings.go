package main

import (
	"encoding/json"
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/DazFather/brush"
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
	reload   time.Duration
	port     string
	name     string
	tags     string
	arg      string
	download bool
	quiet    bool
}

var settings Settings

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

func serveFlags() {
	var f = flag.NewFlagSet("serve", flag.ExitOnError)

	f.StringVar(&settings.port, "port", ":8080", "port for the local server")
	f.StringVar(&settings.port, "p", ":8080", "shorthand for 'port'")
	f.DurationVar(&settings.reload, "live", 0, "reload server each time interval")
	f.DurationVar(&settings.reload, "l", 0, "shorthand for 'live'")

	parseDefault(f, os.Args[2:], serveUsage)

	settings.port = strings.TrimSpace(settings.port)
	if len(settings.port) > 0 && settings.port[0] != ':' {
		settings.port = ":" + settings.port
	}

	return
}

func initFlags() {
	parseDefault(
		flag.NewFlagSet("init", flag.ExitOnError),
		os.Args[2:],
		initUsage,
	)

	if settings.arg != "" {
		settings.InputDir, settings.OutputDir = settings.arg, filepath.Join(settings.arg, settings.OutputDir)
	}

	return
}

func buildFlags() {
	parseDefault(
		flag.NewFlagSet("build", flag.ExitOnError),
		os.Args[2:],
		buildUsage,
	)

	return
}

func runFlags() {
	parseDefault(
		flag.NewFlagSet("run", flag.ExitOnError),
		os.Args[2:],
		runUsage,
	)

	return
}

func libUseFlag(args []string) {
	parseDefault(
		flag.NewFlagSet("lib use", flag.ExitOnError),
		args,
		libUseUsage,
	)

	return
}

func libTrustFlag(args []string) {
	var f = flag.NewFlagSet("lib trust", flag.ExitOnError)

	f.StringVar(&settings.name, "rename", "", "rename locally the trusted library")
	f.StringVar(&settings.name, "r", "", "shorthand for 'rename'")
	f.BoolVar(&settings.download, "download", false, "download a copy of library for offline usage")
	f.BoolVar(&settings.download, "i", false, "shorthand for 'download'")

	parseDefault(f, args, libTrustUsage)

	if settings.name == "" {
		if url, err := cleanURL(settings.arg); err == nil {
			settings.name = cutExt(path.Base(url))
		} else {
			settings.name = cutExt(filepath.Base(settings.arg))
		}
	}

	return
}

func libUntrustFlag(args []string) {
	parseDefault(
		flag.NewFlagSet("lib untrust", flag.ExitOnError),
		args,
		libUntrustUsage,
	)

	return
}

func libSearchFlag(args []string) {
	var f = flag.NewFlagSet("lib search", flag.ExitOnError)

	insensitive := f.Bool("i", false, "insensitive case pattern matching")
	f.StringVar(&settings.tags, "tags", "", "specify another pattern for tag matching")
	f.StringVar(&settings.tags, "t", "", "shorthand for 'tags'")

	parseDefault(f, args, libSearchUsage)

	if *insensitive {
		settings.arg = "(?i)" + settings.arg
		if settings.tags != "" {
			settings.tags = "(?i)" + settings.tags
		}
	}

	return
}

func helpFlags() {
	var f = flag.NewFlagSet("help", flag.ExitOnError)

	f.BoolVar(&brush.Disable, "no-color", false, "force disable of colored output'")
	f.BoolVar(&brush.Disable, "nc", false, "shorthand for 'no-color'")
	f.Usage = func() { doHelp(); os.Exit(1) }

	if err := f.Parse(os.Args[2:]); err != nil {
		f.Usage()
	}

	settings.name, settings.arg = f.Arg(0), f.Arg(1)

	return
}

func parseDefault(f *flag.FlagSet, args []string, usage func()) {
	f.Var(&settings.FileSettings, "settings", "path for the settings json file")
	f.Var(&settings.FileSettings, "s", "shorthand for 'settings'")
	f.BoolVar(&settings.quiet, "quiet", false, "suppress feedback messages")
	f.BoolVar(&settings.quiet, "q", false, "shorthand for 'quiet'")
	f.BoolVar(&brush.Disable, "no-color", false, "force disable of colored output'")
	f.BoolVar(&brush.Disable, "nc", false, "shorthand for 'no-color'")
	f.Usage = func() { usage(); os.Exit(1) }

	if err := f.Parse(args); err != nil {
		f.Usage()
	}

	if settings.from == "" {
		if err := settings.FileSettings.Set("wed-settings.json"); err != nil && !os.IsNotExist(err) {
			panic(err)
		}
	}

	if settings.arg == "" {
		settings.arg = f.Arg(0)
	}

	return
}

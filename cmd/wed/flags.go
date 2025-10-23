package main

import (
	"encoding/json"
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/DazFather/Wednesday/pkg/engine"
	"github.com/DazFather/Wednesday/pkg/shared"

	"github.com/DazFather/brush"
)

type FileSettings struct {
	from string
	engine.Settings
}

type FlagSettings struct {
	reload *time.Duration
	port   string
	name   string
	tags   string
	arg    string
	FileSettings
	download bool
	quiet    bool
}

var settings FlagSettings

// NewSettingsFromJSON creates a new Settings instance from a JSON string.
func NewSettingsFromJSON(spath string) (s FileSettings, err error) {
	s = FileSettings{spath, engine.Settings{
		OutputDir: "build",
		InputDir:  ".",
		Module:    "text/javascript",
	}}

	b, err := os.ReadFile(spath)
	if err != nil {
		return
	}

	if err = json.Unmarshal(b, &s); err == nil {
		s.OutputDir, s.InputDir = filepath.Clean(s.OutputDir), filepath.Clean(s.InputDir)
	}
	return
}

func (s *FileSettings) Set(spath string) (err error) {
	*s, err = NewSettingsFromJSON(spath)
	return
}

func (s FileSettings) String() string {
	return s.from
}

func (s *FlagSettings) parseLiveFlag(sduration string) error {
	switch sduration {
	case "false":
		// skip
	case "", "true":
		s.reload = new(time.Duration)
	default:
		if t, err := time.ParseDuration(sduration); err == nil {
			s.reload = &t
		} else {
			return err
		}
	}

	return nil
}

func serveFlags() {
	var f = flag.NewFlagSet("serve", flag.ExitOnError)

	f.StringVar(&settings.port, "port", ":8080", "port for the local server")
	f.StringVar(&settings.port, "p", ":8080", "shorthand for 'port'")
	f.BoolFunc("live", "reload server each time interval", settings.parseLiveFlag)
	f.BoolFunc("l", "shorthand for 'live'", settings.parseLiveFlag)

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
}

func buildFlags() {
	parseDefault(
		flag.NewFlagSet("build", flag.ExitOnError),
		os.Args[2:],
		buildUsage,
	)

}

func runFlags() {
	parseDefault(
		flag.NewFlagSet("run", flag.ExitOnError),
		os.Args[2:],
		runUsage,
	)
}

func libUseFlag(args []string) {
	parseDefault(
		flag.NewFlagSet("lib use", flag.ExitOnError),
		args,
		libUseUsage,
	)
}

func libTrustFlag(args []string) {
	var f = flag.NewFlagSet("lib trust", flag.ExitOnError)

	f.StringVar(&settings.name, "rename", "", "rename locally the trusted library")
	f.StringVar(&settings.name, "r", "", "shorthand for 'rename'")
	f.BoolVar(&settings.download, "download", false, "download a copy of library for offline usage")
	f.BoolVar(&settings.download, "i", false, "shorthand for 'download'")

	parseDefault(f, args, libTrustUsage)

	if settings.name == "" {
		if url, err := shared.CleanURL(settings.arg); err == nil {
			settings.name = cutExt(path.Base(url))
		} else {
			settings.name = cutExt(filepath.Base(settings.arg))
		}
	}
}

func libUntrustFlag(args []string) {
	parseDefault(
		flag.NewFlagSet("lib untrust", flag.ExitOnError),
		args,
		libUntrustUsage,
	)
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
}

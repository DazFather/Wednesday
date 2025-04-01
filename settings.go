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
	Var       map[string]string   `json:"vars,omitempty"`
	Commands  map[string][]string `json:"commands,omitempty"`
	OutputDir string              `json:"output_dir,omitempty"`
	InputDir  string              `json:"input_dir,omitempty"`
	from      string
}

type Settings struct {
	FileSettings
	reload time.Duration
	port   string
	name   string
	tags   string
	arg    string
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

	parseDefault(f, &s, os.Args[2:], func() {
		commandUsage("serve", `Build the project and serve it statically.
If build phase fails program exit without running any server


Command Flags:

  -p, --port <port>      Specify the server port by default :8080 will be used.
                         Character ':' at the beginning is optional

  -l, --live <interval>  Enable rebuilding at a specified time interval.
                         If 0 or value not provided there will be no rebuilding
`)
	})

	s.port = strings.TrimSpace(s.port)
	if len(s.port) > 0 && s.port[0] != ':' {
		s.port = ":" + s.port
	}

	return
}

func initFlags() (s Settings) {
	help := func() {
		commandUsage("init [dir]", `Create a default project.
Optionally you can specify a directory just after the command.
If not provided the current working directory will be used instead.
If provided but do not exist it be created
`)
	}

	parseDefault(
		flag.NewFlagSet("init", flag.ExitOnError),
		&s,
		os.Args[2:],
		help,
	)

	if s.arg != "" {
		s.InputDir, s.OutputDir = s.arg, filepath.Join(s.arg, s.OutputDir)
	}

	return
}

func buildFlags() (s Settings) {
	help := func() {
		commandUsage("build", `Compile the project into a static site.
If not specified otherwise ('InputDir' project settings file) the current working
directory will be used as entrypoint and all subdirectory will be checked recursively.
The program will treats all files with extention '.wed.html' as components and
'.tmpl' as pages.

Where do things go to:
The output will be located at 'OutputDir' ('./build' by default). In the specific
  CSS styles into 'style' subdirectory
  JS scripts into 'script' subdirectory
All the pages at the top level inside 'OutputDir'
`)
	}

	parseDefault(
		flag.NewFlagSet("build", flag.ExitOnError),
		&s,
		os.Args[2:],
		help,
	)

	return
}

func runFlags() (s Settings) {
	help := func() {
		commandUsage("run <command>", `Execute a user-defined command.
Douring execution if one fails the program terminates.
On Windows environment the os variable COMSPEC will be used to detect preference.
if not found all command will be launched via 'cmd' 
On other environments the os variable SHELL will be used instead and
if not found 'sh' will be used


How to set a command:
They can be set using the 'Commands' property on the project file settings.
The property is a map: command name -> sequence of operation to execute.
Therefore two commands cannot have an identical name.
For example:
...
"Commands": {
	"update": [
		"git fetch",
		"git pull"
	],
	"live": ["wed serve --port=4200 --live=10s"]
}
...

`)
	}

	parseDefault(
		flag.NewFlagSet("run", flag.ExitOnError),
		&s,
		os.Args[2:],
		help,
	)

	return
}

func libUseFlag(args []string) (s Settings) {
	help := func() {
		commandUsage("lib use [component]", `Use a component in the current project.
The component must be from an already trusted library.
It's possible to specify the name by only using the name, or to avoid homonymous
by prefixing it with the name of the library followed by '/'
When using this command http call(s) will be made to download the component and
if present it's dependencies.


Where do things go to:
All components will be download inside a subdirectory of 'InputDir' (by default
the current directory) called with the same library name of the requested one
In order to avoid homonymous they will be renamed by prefixing it with the
library name of the requested one followed by '-'
`)
	}

	parseDefault(
		flag.NewFlagSet("lib use", flag.ExitOnError),
		&s,
		args,
		help,
	)

	return
}

func libTrustFlag(args []string) (s Settings) {
	var f = flag.NewFlagSet("lib trust", flag.ExitOnError)

	f.StringVar(&s.name, "rename", "", "rename locally the trusted library")
	f.StringVar(&s.name, "n", "", "shorthand for 'rename'")

	parseDefault(f, &s, args, func() {
		commandUsage("lib trust <link>", `Trust a library and download it's manifest by following the provided link.
If starts with 'http' then an HTTP GET request will be made to retreive the file.
Or else it's assumed is a path to a local file and therefore it will be copied it


Where do things go to:
The library manifest will be created in the user configuration directory at
wednesday/trusted/<library name>.json


Command Flags:

--rename, -r <name>  Provide a name for the library. By default the name is
                     extrapolated from the provided link
`)
	})

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

	parseDefault(f, &s, args, func() {
		commandUsage("lib search [pattern]", `Obtain a detailed view of matching components within trusted libraries.
By default the provided pattern will be used for matching components name or tags


Command Flags:

  -i                    Enable case-insensitive pattern matching

  --tags, -t <pattern>  Filter only matching component tags with provided pattern
`)
	})

	if *insensitive {
		s.arg = "(?i)" + s.arg
		if s.tags != "" {
			s.tags = "(?i)" + s.tags
		}
	}

	return
}

func parseDefault(f *flag.FlagSet, s *Settings, args []string, usage func()) {
	f.Var(&s.FileSettings, "settings", "path for the settings json file")
	f.Var(&s.FileSettings, "s", "shorthand for 'settings'")
	f.Usage = func() { usage(); os.Exit(1) }

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

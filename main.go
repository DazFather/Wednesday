package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	_ "embed"
)

var (
	//go:embed templates/index.tmpl
	indexTemplate []byte
	//go:embed templates/app.wed.html
	appTemplate []byte
	//go:embed resources/style.default.css
	defStyleContent []byte
	//go:embed resources/wed-utils.js
	defScriptContent []byte
)

const (
	defStyleName      = "wed-style.css"
	defScriptName     = "wed-utils.js"
	indexTemplateName = "index.tmpl"
	defAppName        = "app.wed.html"
)

func initCmd(args []string) (err error) {
	var flags = flag.NewFlagSet("init", flag.ContinueOnError)

	s, err := LoadSettings(flags, args, "wed-settings.json")
	if os.IsNotExist(err) {
		fmt.Println("WARNING: Missing settings file, using default settings")
		s.OutputDir = "build"
	} else if err != nil {
		return
	}

	m := map[string][]byte{
		s.StylePath(defStyleName):                    defStyleContent,
		s.ScriptPath(defScriptName):                  defScriptContent,
		filepath.Join(s.InputDir, indexTemplateName): indexTemplate,
	}

	for name, content := range m {
		if err = os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			return
		}
		if err = os.WriteFile(name, content, 0644); err != nil {
			return
		}
	}

	return os.WriteFile(filepath.Join(s.InputDir, defAppName), appTemplate, 0644)
}

func buildCmd(args []string) (err error) {
	var flags = flag.NewFlagSet("build", flag.ContinueOnError)

	s, err := LoadSettings(flags, args, "wed-settings.json")
	if os.IsNotExist(err) {
		fmt.Println("WARNING: Missing settings file, using default settings")
		s.OutputDir = "build"
	} else if err != nil {
		return
	}

	td, err := NewTemplateData(s)
	if err != nil {
		return
	}

	if err = td.Walk(); err == nil {
		err = td.Build("index.html")
	}
	return
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing command\nUse 'help' for usage")
		return
	}

	var err error

	switch os.Args[1] {
	case "build":
		err = buildCmd(os.Args[2:])
	case "init":
		err = initCmd(os.Args[2:])
	case "help", "h", "-h", "--h", "-help", "--help":
		flag.Usage()
	case "version", "v", "-v", "--v", "-version", "--version":
		fmt.Println("1.0 pre-alpha")
	default:
		err = fmt.Errorf("Unknown given command:", os.Args[1], "\nUse 'help' for usage")
	}

	if err != nil {
		panic(err)
	}
}

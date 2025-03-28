package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	var flags = NewBaseFlags("init")

	if err = flags.Parse(args); err != nil {
		return
	}

	m := map[string][]byte{
		flags.Settings.StylePath(defStyleName):                    defStyleContent,
		flags.Settings.ScriptPath(defScriptName):                  defScriptContent,
		filepath.Join(flags.Settings.InputDir, indexTemplateName): indexTemplate,
	}

	for name, content := range m {
		if err = os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			return
		}
		if err = os.WriteFile(name, content, 0644); err != nil {
			return
		}
	}

	return os.WriteFile(filepath.Join(flags.Settings.InputDir, defAppName), appTemplate, 0644)
}

func buildCmd(args []string) (err error) {
	var flags = NewBaseFlags("build")

	if err = flags.Parse(args); err != nil {
		return
	}

	td, err := NewTemplateData(flags.Settings)
	if err != nil {
		return
	}

	if err = td.Walk(); err == nil {
		err = td.Build("index.html")
	}
	return
}

func serveCmd(args []string) (err error) {
	var (
		port  string
		flags = NewBaseFlags("serve", portFlag(&port, ":8080"))
	)
	if err = flags.Parse(args); err != nil {
		return
	}

	port = strings.TrimSpace(port)
	if len(port) > 0 && port[0] != ':' {
		port = ":" + port
	}

	return http.ListenAndServe(port, http.StripPrefix("/",
		http.FileServer(http.Dir("./"+flags.Settings.OutputDir))),
	)
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
	case "serve":
		err = serveCmd(os.Args[2:])
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

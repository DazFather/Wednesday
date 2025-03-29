package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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

func doInit(s Settings) (err error) {
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

func doBuild(s Settings) (err error) {
	var td = NewTemplateData(s.FileSettings)

	if err = td.Walk(); err == nil {
		err = td.Build()
	}
	return err
}

func doServe(s Settings) error {
	return http.ListenAndServe(
		s.port,
		http.StripPrefix("/", http.FileServer(http.Dir("./"+s.OutputDir))),
	)
}

func defaultShell() (sh, flag string) {
	if runtime.GOOS == "windows" {
		if sh = os.Getenv("COMSPEC"); sh == "" {
			sh = "cmd.exe"
		}
		flag = "/c"
	} else {
		if sh = os.Getenv("SHELL"); sh == "" {
			sh = "/bin/sh"
		}
		flag = "-c"
	}
	return
}

func doRun(s Settings) error {
	commands, ok := s.Commands[s.arg]
	if !ok {
		return fmt.Errorf("unknown command: %s", s.arg)
	}

	sh, flag := defaultShell()
	for _, c := range commands {
		cmd := exec.Command(sh, flag, c)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing command\nUse 'help' for usage")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "build":
		err = doBuild(buildFlags())
	case "init":
		err = doInit(initFlags())
	case "serve":
		err = doServe(serveFlags())
	case "run":
		err = doRun(runFlags())
	case "help", "h", "-h", "--h", "-help", "--help":
		flag.Usage()
	case "version", "v", "-v", "--v", "-version", "--version":
		fmt.Println("1.0 pre-alpha")
	default:
		err = fmt.Errorf("Unknown given command: %v\n Use 'help' for usage\n", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

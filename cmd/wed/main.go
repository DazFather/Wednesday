package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	_ "embed"
)

var Version string = "alpha"

var (
	//go:embed templates/index.tmpl
	indexTemplate []byte
	//go:embed templates/app.wed.html
	appTemplate []byte
	//go:embed resources/style.default.css
	defStyleContent []byte
	//go:embed resources/wed-utils.js
	defScriptContent []byte
	//go:embed resources/wed-utils.mjs
	defScriptModuleContent []byte
	//go:embed resources/wed-http.mjs
	defHttpScriptModuleContent []byte
)

func doInit() (err error) {
	m := map[string][]byte{
		settings.StylePath("wed-style"):                defStyleContent,
		settings.ScriptPath("wed", "utils.js"):         defScriptContent,
		settings.ScriptPath("wed", "utils.mjs"):        defScriptModuleContent,
		settings.ScriptPath("wed", "http.mjs"):         defHttpScriptModuleContent,
		filepath.Join(settings.InputDir, "index.tmpl"): indexTemplate,
	}

	for name, content := range m {
		if err = os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			return
		}
		if err = os.WriteFile(name, content, 0644); err != nil {
			return
		}
	}

	if err = os.WriteFile(filepath.Join(settings.InputDir, "app.wed.html"), defaultAppComponent(), 0644); err == nil {
		printlnDone("init", "Successfully created scaffolding project at", gray.Paint(settings.InputDir))
	}
	return
}

func doBuild() (err error) {
	i := 0
	for err := range build() {
		i++
		fmt.Println(red.Paint(i), err)
		printlnStackTrace(err)
	}
	if i > 0 {
		return fmt.Errorf("Failed to build site errors: %d", i)
	}

	printlnDone("build", "Site successfully built at", gray.Paint(settings.OutputDir))
	return nil
}

func doServe() error {
	if err := doBuild(); err != nil {
		return err
	}

	if settings.reload != nil {
		go func() {
			for errs := range liveReload() {
				switch len(errs) {
				case 0:
					printlnDone("build", "Site successfully rebuilt, no error found\n")
				case 1:
					printlnFailed("build", "Cannot rebuild site fond:\n", errs[0])
					printlnStackTrace(errs[0])
				default:
					printlnFailed("build", "Cannot rebuild site fond", len(errs), "errors:")
					for i, err := range errs {
						fmt.Println(red.Paint(i+1), err)
						printlnStackTrace(err)
					}
				}
			}
		}()
	}

	hint("Listening at: ", gray.Paint(settings.port), "\nServing directory: ", gray.Paint(settings.OutputDir), "\n")
	return http.ListenAndServe(
		settings.port,
		http.StripPrefix("/", http.FileServer(http.Dir("./"+settings.OutputDir))),
	)
}

func doRun() error {
	commands, ok := settings.Commands[settings.arg]
	if !ok {
		return fmt.Errorf("unknown command: '%s'\nUse 'help run' for usage", settings.arg)
	}

	sh, flag := defaultShell()
	for _, c := range commands {
		cmd := exec.Command(sh, flag, c)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		fmt.Println(c)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	printlnDone("run", "All commands on the pipeline have been executed successfully")

	return nil
}

func doLib(args []string) (err error) {
	if len(args) < 1 {
		return fmt.Errorf("not enough argument.\nUse 'help lib' for usage")
	}

	switch args[0] {
	case "search":
		libSearchFlag(args)
		err = doLibSearch()
	case "trust":
		libTrustFlag(args)
		err = doLibTrust()
	case "untrust":
		libUntrustFlag(args)
		err = doLibUntrust()
	case "use":
		libUseFlag(args)
		err = doLibUse()
	case "help", "h", "-h", "--h", "-help", "--help":
		libUsage()
	default:
		err = fmt.Errorf("unknown given lib subcommand: '%s'\nUse 'help lib' for usage", args[0])
	}

	return
}

func doHelp() error {
	switch settings.name {
	case "":
		mainUsage()
	case "lib":
		switch settings.arg {
		case "":
			libUsage()
		case "search":
			libSearchUsage()
		case "trust":
			libTrustUsage()
		case "untrust":
			libUntrustUsage()
		case "use":
			libUseUsage()
		default:
			return fmt.Errorf("unknown given subcommand: %q\n Use 'help %s' for usage\n", settings.arg, settings.name)
		}
	case "build":
		buildUsage()
	case "init":
		initUsage()
	case "serve":
		serveUsage()
	case "run":
		runUsage()
	default:
		return fmt.Errorf("unknown given command: %q\n Use 'help' for usage\n", settings.name)
	}

	return nil
}

func main() {
	var (
		command string
		err     error
	)
	defer func() {
		if err != nil {
			printlnFailed(command, err)
			os.Exit(1)
		}
	}()

	if len(os.Args) < 2 {
		err = fmt.Errorf("Missing command\nUse 'help' for usage")
		return
	}

	switch command = os.Args[1]; command {
	case "lib":
		err = doLib(os.Args[2:])
	case "build":
		buildFlags()
		err = doBuild()
	case "init":
		initFlags()
		err = doInit()
	case "serve":
		serveFlags()
		err = doServe()
	case "run":
		runFlags()
		err = doRun()
	case "help", "h", "-h", "--h", "-help", "--help":
		helpFlags()
		err = doHelp()
	case "version", "v", "-v", "--v", "-version", "--version":
		fmt.Println(Version)
	default:
		err = fmt.Errorf("unknown given command: %q\n Use 'help' for usage\n", command)
	}
}

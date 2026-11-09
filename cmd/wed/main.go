package main

import (
	"context"
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
	m := defAssets()
	m[filepath.Join(settings.InputDir, "index.tmpl")] = indexTemplate
	m[filepath.Join(settings.InputDir, "app.wed.html")] = defaultAppComponent()

	for name, content := range m {
		if settings.force {
			hint(gray.Paint("reinitializing ", name, " with default values"), "\n")
		} else if _, err := os.Stat(name); err == nil {
			continue
		}
		if dir := filepath.Dir(name); dir != "." {
			if err = os.MkdirAll(dir, 0750); err != nil {
				return
			}
		}
		if err = os.WriteFile(name, content, 0600); err != nil {
			return
		}
	}

	printlnDone("init", "Successfully created scaffolding project at", gray.Paint(settings.InputDir))
	return
}

func doBuild() (err error) {
	var errs []error
	for name, content := range defAssets() {
		if dir := filepath.Dir(name); dir != "." {
			if e := os.MkdirAll(dir, 0750); e != nil {
				errs = append(errs, e)
			}
		}
		if _, e := os.Stat(name); e != nil && os.IsNotExist(e) {
			hint(gray.Paint("Detected missing assets '", name, "' recreating it\n"))
			if e = os.WriteFile(name, content, 0600); e != nil {
				errs = append(errs, e)
			}
		}
	}
	if n := len(errs); n > 0 {
		for i, err := range errs {
			fmt.Println(red.Paint(i), err)
			printlnStackTrace(err)
		}
		return fmt.Errorf("failed to recreate missing assets, errors: %d", n)
	}

	i := 0
	for err := range build() {
		i++
		fmt.Println(red.Paint(i), err)
		printlnStackTrace(err)
	}
	if i > 0 {
		return fmt.Errorf("failed to build site, errors: %d", i)
	}

	printlnDone("build", "Site successfully built at", gray.Paint(settings.OutputDir))
	return nil
}

func doServe() error {
	if err := doBuild(); err != nil {
		return err
	}

	server, han := serveDir(settings.port, settings.OutputDir)
	ctx, stop := useStopper(nil, server.Shutdown, func(_ context.Context) error {
		hint(gray.Paint("\nGracefully shutting down server\n"))
		return nil
	})
	defer stop()

	if settings.reload != nil {
		go func() {
			for errs := range liveReload(ctx, han) {
				switch len(errs) {
				case 0:
					printlnDone("build", "Site successfully rebuilt, no error found\n")
				case 1:
					printlnFailed("build", "Cannot rebuild site found:\n", errs[0])
					printlnStackTrace(errs[0])
				default:
					printlnFailed("build", "Cannot rebuild site found", len(errs), "errors:")
					for i, err := range errs {
						fmt.Println(red.Paint(i+1), err)
						printlnStackTrace(err)
					}
				}
			}
		}()
	}

	hint("Listening at: ", gray.Paint(settings.port), "\nServing directory: ", gray.Paint(settings.OutputDir), "\n")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func doRun() error {
	commands, ok := settings.Commands[settings.arg]
	if !ok {
		return fmt.Errorf("unknown command: '%s'\nUse 'help run' for usage", settings.arg)
	}

	sh, flag := defaultShell()
	for _, c := range commands {
		cmd := exec.Command(sh, flag, c) // #nosec G204 -- executes user-configured commands from wed-settings.json by design
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
			return fmt.Errorf("unknown given subcommand: %q (use 'help %s' for usage)", settings.arg, settings.name)
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
		return fmt.Errorf("unknown given command: %q (use 'help' for usage)", settings.name)
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
		err = fmt.Errorf("missing command (use 'help' for usage)")
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
		err = fmt.Errorf("unknown given command: %q (use 'help' for usage)", command)
	}
}

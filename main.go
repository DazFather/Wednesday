package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	command   string
	arguments []string
	flags     map[string]string
)

func main() {
	wed(os.Args[1:]...)
}

func wed(args ...string) (err error) {
	if len(args) == 0 {
		err = errors.New("Invalid given arguments")
		danger(err, "None given", usageSnip)
		return
	}

	switch command, arguments, flags = LoadFlags(args); command {
	case "help", "h":
		doUsage()
	case "init":
		doInit()
	case "build":
		_, err = doBuild()
	case "serve":
		err = doServe()
	case "run":
		err = doRun()
	default:
		err = errors.New("Invalid given arguments")
		danger(err, `Unknown command "`+command+`"`, usageSnip)
		return
	}

	if err != nil {
		danger(command, err)
	}
	return
}

func LoadFlags(rawArgs []string) (cmd string, args []string, f map[string]string) {
	const defaultPort = ":8080"
	if len(rawArgs) == 0 {
		return
	}

	// default values
	f = map[string]string{
		"port":     defaultPort,
		"live":     "",
		"settings": "",
	}

	var flagRgx = regexp.MustCompile(`^--?(\w+)=?"?(.+)?"?$`)
	for _, arg := range rawArgs {
		matches := flagRgx.FindStringSubmatch(arg)
		if len(matches) != 3 {
			args = append(args, arg)
			continue
		}
		argKey, value := matches[1], matches[2]

		found := false
		for key := range f {
			if strings.HasPrefix(key, argKey) {
				f[key] = value
				found = true
				break
			}
		}
		if !found {
			warn("Invalid flag", `Unknown given flag name "`+argKey+`"`)
		}
	}

	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}

	if val, found := f["port"]; found {
		if correct, _ := regexp.MatchString(`^:?[0-9]{1,4}$`, val); !correct {
			warn("Invalid flag", `Invalid "port" property value (`+val+`). Default "`+defaultPort+`" will be used`)
		} else if val[0] != ':' {
			f["port"] = ":" + val
		}
	}

	if val, found := f["live"]; found {
		if poll, err := time.ParseDuration(val); err == nil {
			f["live"] = strconv.Itoa(int(poll))
		} else {
			warn("Invalid flag", `Invalid "live" property value (`+val+`). Server will not rebuild application onece started`)
			f["port"] = ""
		}
	}
	return
}

func doInit() (s *Settings) {
	var (
		err      error
		fileName = flags["settings"]
		mountDir = flags["mount"]
	)

	switch fileName {
	case "", ".":
		fileName = settingsFileName
	default:
		// load settings and generate missing directories
		if sets, w := LoadSettings(fileName); w == nil {
			s = &sets
		}
	}

	if s == nil {
		s = &Settings{
			HomeTempl: homeFileName,
			HomeDir:   "build",
			ScriptDir: "scripts",
			StyleDir:  "styles",
		}
		if content, err := json.MarshalIndent(s, "", "\t"); err == nil {
			_, err = genFile(false, content, mountDir, fileName)
		}
		if err != nil {
			warn(`Cannot create "`+fileName+`" settings file`, err)
		}
	}

	// gen directories
	if err = s.genDirs(mountDir); err != nil {
		warn(`Cannot create needed directory`, err)
	}

	// gen app component
	if _, err = genFile(false, appComponentContent, mountDir, appFileName); err != nil {
		warn(`Cannot create default app component`, err)
	}

	// gen home template
	if _, err = genFile(false, homeTemplateContent, mountDir, s.HomeTempl); err != nil {
		warn(`Cannot create default "`+s.HomeTempl+`" template`, err)
	}

	// gen wed-utils.js
	if _, err = genFile(false, defScriptContent, mountDir, s.HomeDir, s.ScriptDir, scriptFileName); err != nil {
		warn(`Cannot create default "`+scriptFileName+`" script`, err)
	}

	// gen wed-style.css
	if _, err = genFile(false, defStyleContent, mountDir, s.HomeDir, s.StyleDir, styleFileName); err != nil {
		warn(`Cannot create default "`+styleFileName+`" stylesheet`, err)
	}

	success("init", "Initialization process completed")
	return
}

func doBuild() (builtAt string, err error) {
	// load settings and generate missing directories
	settings, warnings := LoadSettings(flags["settings"])
	if warnings != nil {
		warn("Cannot properly load settings", warnings)
	}

	// build application with specified settings taking component from specified componentsDir
	switch len(arguments) {
	case 1:
		builtAt = arguments[0]
		fallthrough
	case 0:
		if err = Build(builtAt, &settings); err != nil {
			err = errors.New("Cannot build application, " + err.Error())
		} else {
			builtAt = link(builtAt, settings.HomeDir)
			success("build", "Find all the files in the directory", `"`+builtAt+`"`)
		}
	default:
		err = errors.New("Too many arguments. " + usageSnip)
	}

	return
}

func doUsage() (err error) {
	ShowUsage()
	return
}

func doServe() (err error) {
	var port = flags["port"]

	appDir, err := doBuild()
	if err != nil {
		return
	}

	if v := flags["live"]; v != "" {
		var each time.Duration
		if nano, e := strconv.Atoi(v); e == nil && nano > 0 {
			each = time.Duration(nano)
		} else {
			panic(e)
		}

		tick := time.NewTicker(each)
		defer tick.Stop()
		go func() {
			for range tick.C {
				running("Live server", "building each", each.String())
				if _, err := doBuild(); err != nil {
					warn("Build failed", err)
				}
			}
		}()
	}

	if appDir != "." {
		appDir = "/" + appDir + "/"
	}

	success("serve", "Launching server\n port:", port, "\n assets:", appDir)
	return http.ListenAndServe(
		port,
		http.StripPrefix("/", http.FileServer(http.Dir("."+appDir))),
	)
}

func doRun() (err error) {
	// load settings and generate missing directories
	settings, err := LoadSettings(flags["settings"])
	if err != nil {
		return errors.New("Cannot properly load settings, " + err.Error())
	}
	if len(settings.Run) == 0 {
		return errors.New(`No stage in "run" pipeline`)
	}

	spaces := regexp.MustCompile(`\s+`)
	for i, step := range settings.Run {
		args := spaces.Split(step, -1)
		if len(args) == 0 {
			warn("Invalid run step", "Skipping empty given "+strconv.Itoa(i)+" step")
			continue
		}

		running(step, "running step ", i+1, "/", len(settings.Run))
		if args[0] == "wed" {
			err = wed(args[1:]...)
		} else {
			err = run(args...)
		}
		if err != nil {
			err = errors.New("Error at step " + strconv.Itoa(i) + ` "` + step + `": ` + err.Error())
			return
		}
	}

	return
}

func run(args ...string) (err error) {
	var cmd = exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = NewIndentWriter(" ", "\n\n", nil)
	// cmd.Stderr = os.Stderr
	return cmd.Run()
}

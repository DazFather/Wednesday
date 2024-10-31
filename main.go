package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var (
	command   string
	arguments []string
	flags     map[string]string
)

func main() {
	if len(os.Args) <= 1 {
		danger("Empty arguments", usageSnip)
		return
	}

	var err error

	switch command, arguments, flags = LoadFlags(os.Args[1:]); command {
	case "help", "h":
		doUsage()
	case "init":
		doInit()
	case "build":
		_, err = doBuild()
	case "serve":
		err = doServe()
	default:
		danger(`Unknown command "`+command+`"`, usageSnip)
		return
	}

	if err != nil {
		danger(command, err)
	}
}

func LoadFlags(rawArgs []string) (cmd string, args []string, f map[string]string) {
	const defaultPort = ":8080"
	if len(rawArgs) == 0 {
		return
	}

	// default values
	f = map[string]string{
		"port":     defaultPort,
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

	if appDir != "." {
		appDir = "/" + appDir + "/"
	}

	success("serve", "Launching server\n port:", port, "\n assets:", appDir)
	return http.ListenAndServe(
		port,
		http.StripPrefix("/", http.FileServer(http.Dir("."+appDir))),
	)
}

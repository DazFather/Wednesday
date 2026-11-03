package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/DazFather/Wednesday/pkg/engine"
	"github.com/DazFather/Wednesday/pkg/shared"

	"github.com/fsnotify/fsnotify"
)

func defAssets() map[string][]byte {
	return map[string][]byte{
		settings.StylePath("wed-style"):      defStyleContent,
		settings.ScriptPath("wed-utils.js"):  defScriptContent,
		settings.ScriptPath("wed-utils.mjs"): defScriptModuleContent,
		settings.ScriptPath("wed-http.mjs"):  defHttpScriptModuleContent,
	}
}

func cutExt(s string) string {
	return s[:len(s)-len(filepath.Ext(s))]
}

func autoheal(path string) error {
	path = filepath.Clean(path)

	for _, subdir := range []string{"style", "script"} {
		if err := os.MkdirAll(filepath.Join(settings.OutputDir, subdir), 0755); err != nil {
			return err
		}
	}

	assets := defAssets()
	if content, found := assets[path]; found {
		return os.WriteFile(path, content, 0644)
	}

	for name, content := range assets {
		if err := os.WriteFile(name, content, 0644); err != nil {
			return err
		}
	}
	return nil
}

func getWedConfigDir(pices ...string) (dir string, err error) {
	if dir, err = os.UserConfigDir(); err == nil {
		if len(pices) == 0 {
			dir = filepath.Join(dir, "wednesday")
		} else {
			temp := make([]string, len(pices)+2)
			temp[0], temp[1] = dir, "wednesday"
			for i := range pices {
				temp[i+2] = pices[i]
			}
			dir = filepath.Join(temp...)
		}
	}
	return
}

func extractLibName(s string) (lib, name string) {
	if ind := strings.IndexAny(s, `\/`); ind == -1 {
		name = s
	} else {
		name, lib = s[:ind], s[ind+1:]
	}
	return
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

func watch(rootdir, builddir string, stop <-chan bool) (<-chan string, error) {

	isInsideDir := func(path string) bool {
		rel, err := filepath.Rel(builddir, path)
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}

	isNotAssetDir := func(path string) bool {
		prefix := builddir + string(filepath.Separator)
		return path != builddir && path != prefix+"style" && path != prefix+"script"
	}

	stopHeal, stopBuild := make(chan bool), make(chan bool)
	go func() {
		for v := range stop {
			if v {
				stopHeal <- true
				stopBuild <- true
			}
		}
	}()

	notifyHeal, err := shared.WatchDir(builddir, isNotAssetDir, stopHeal)
	if err != nil {
		return nil, err
	}

	notifyBuild, err := shared.WatchDir(rootdir, isInsideDir, stopBuild)
	if err != nil {
		return nil, err
	}

	notify := make(chan string)
	go func() {
		defer close(notify)
		for event := range notifyBuild {
			notify <- event.Name
		}
	}()

	go func() {
		for event := range notifyHeal {
			if event.Op.Has(fsnotify.Remove) || event.Op.Has(fsnotify.Rename) {
				if err := autoheal(event.Name); err != nil {
					panic(err)
				}
			}
		}
	}()

	return notify, nil
}

func useSSE(buildCh <-chan []error) {
	var hub shared.SSEHandler

	go func() {
		for errs := range buildCh {
			switch nerrs := len(errs); nerrs {
			case 0:
				hub.Broadcast("event: build\ndata: success")
			default:
				hub.Broadcast(fmt.Sprint("event: build-errors\ndata: ", nerrs))
			}
		}
	}()

	http.HandleFunc(settings.LiveServer, hub.Handler(&shared.SSEHandlerOpt{
		CrossOriginHeader: "*",
		HandleErr:         func(e error) { hint("[Live Server]", e) },
	}))
}

func each(reload time.Duration, rootdir, builddir string, stop <-chan bool) (<-chan string, error) {
	var notify = make(chan string)
	var tick = time.NewTicker(reload)
	assets := defAssets()

	go func() {
		if v := <-stop; v {
			tick.Stop()
			close(notify)
		}
	}()

	go func() {
		latest := time.Now().Unix()

		for range tick.C {
			err := filepath.Walk(rootdir, func(path string, info os.FileInfo, e error) error {
				if e != nil {
					return e
				}

				if !info.IsDir() {
					if tfile := info.ModTime().Unix(); tfile > latest {
						latest = tfile
						notify <- path
						return filepath.SkipAll
					}
				} else if path == builddir {
					return filepath.SkipDir
				}
				return nil
			})
			if err != nil {
				tick.Stop()
				close(notify)
				return
			}

			for name := range assets {
				if _, err := os.Stat(name); err != nil {
					if err = autoheal(name); err != nil {
						panic(err)
					}
				}
			}
		}
	}()

	return notify, nil
}

func build() chan error {
	errch := make(chan error)
	go func() {
		defer close(errch)
		engine.Build(settings.FileSettings.Settings, errch)
	}()
	return errch
}

func liveReload(stop <-chan bool) chan []error {
	var (
		notify <-chan string
		err    error
	)
	if *settings.reload == 0 {
		notify, err = watch(settings.InputDir, settings.OutputDir, stop)
	} else {
		notify, err = each(*settings.reload, settings.InputDir, settings.OutputDir, stop)
	}

	errch := make(chan []error)
	if err != nil {
		errch <- []error{err}
		close(errch)
		return errch
	}

	ssech := make(chan []error)
	useSSE(ssech)

	go func() {
		defer close(errch)
		defer close(ssech)

		prev := ""
		for range notify {
			errs, serr := []error{}, ""

			for err := range build() {
				errs = append(errs, err)
				serr += err.Error()
			}

			if serr != prev {
				errch <- errs
				ssech <- errs
				prev = serr
			} else if serr == "" {
				ssech <- errs
			}
		}
	}()

	return errch
}

func defaultAppComponent() []byte {
	var buf = bytes.NewBuffer([]byte{})
	t, err := template.New("temp").Delims("{!default{", "}!}").Parse(string(appTemplate))
	if err != nil {
		panic(err)
	}

	if err = t.Execute(buf, settings); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

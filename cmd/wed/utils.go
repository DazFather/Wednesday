package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/DazFather/Wednesday/pkg/engine"

	"github.com/fsnotify/fsnotify"
)

func cutExt(s string) string {
	return s[:len(s)-len(filepath.Ext(s))]
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

func watch(rootdir, builddir string, notify func() error) error {
	var watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = filepath.Walk(rootdir, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}

		if info.IsDir() {
			if path == builddir {
				return filepath.SkipDir
			}
			watcher.Add(path)
		}
		return nil
	})

	for err == nil {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				continue
			}

			if event.Op.Has(fsnotify.Create) {
				var info os.FileInfo
				if info, err = os.Stat(event.Name); err == nil && info.IsDir() && event.Name != builddir {
					watcher.Add(event.Name)
				}
			}
			err = notify()
		case e, ok := <-watcher.Errors:
			if !ok {
				continue
			}
			if e != nil {
				err = e
			}
		}
	}

	return err
}

func sse(ready <-chan int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// You may need this locally for CORS requests
		w.Header().Set("Access-Control-Allow-Origin", "*")

		connection, cancel := context.WithCancel(r.Context())

		rc := http.NewResponseController(w)

		for {
			select {
			case <-connection.Done():
				hint("Live server: Client disconnected\n")
				return
			case nerr := <-ready:
				var value = "error"
				if nerr == 0 {
					cancel()
					value = "refresh"
				}
				if _, err := fmt.Fprintf(w, "data: %s\n\n", value); err != nil {
					hint("Live server: error while sending data over SSE", err)
					return
				}
				if err := rc.Flush(); err != nil {
					hint("Live server: error while flushing data over SSE", err)
					return
				}
				if nerr == 0 {
					return
				}
			}
		}
	}
}

func each(reload time.Duration, notify func() error) error {
	var tick = time.NewTicker(reload)
	defer tick.Stop()

	for range tick.C {
		if err := notify(); err != nil {
			return err
		}
	}
	return nil
}

func build() chan error {
	return engine.Build(settings.FileSettings.Settings)
}

func liveReload() chan []error {
	var ssech chan int
	errch, prev := make(chan []error), ""

	var reload = func() error {
		var (
			errs []error
			serr string
		)
		for err := range build() {
			errs = append(errs, err)
			serr += err.Error()
		}

		if serr != prev {
			errch <- errs
			prev = serr
		}
		if ssech != nil {
			ssech <- len(errs)
		}
		return nil
	}

	go func() {
		defer close(errch)

		var err error
		if *settings.reload == 0 {
			ssech = make(chan int)
			http.HandleFunc("/wed-live", sse(ssech))
			if err = watch(settings.InputDir, settings.OutputDir, reload); err != nil {
				err = fmt.Errorf("Live server stopped working cause %w", err)
			}
		} else if err = each(*settings.reload, reload); err != nil {
			err = fmt.Errorf("Live server stopped working cause %w", err)
		}
		if err != nil {
			errch <- []error{err}
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

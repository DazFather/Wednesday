package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
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

func watch(ctx context.Context, rootdir, builddir string) (<-chan string, error) {

	isInsideDir := func(path string) bool {
		rel, err := filepath.Rel(builddir, path)
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}

	isNotAssetDir := func(path string) bool {
		prefix := builddir + string(filepath.Separator)
		return path != builddir && path != prefix+"style" && path != prefix+"script"
	}

	notifyHeal, err := shared.WatchDir(builddir, isNotAssetDir)
	if err != nil {
		return nil, err
	}

	notifyBuild, err := shared.WatchDir(rootdir, isInsideDir)
	if err != nil {
		return nil, err
	}

	notify := make(chan string)
	go func() {
		defer close(notify)
		for event := range notifyBuild.Iter(ctx) {
			ctxsend(ctx, notify, event.Name)
		}
		if err := notifyBuild.Err(); err != nil {
			hint(red.Paint(" ! "), " Autodetect changes stopped working: ", err)
		}
	}()

	go func() {
		for event := range notifyHeal.Iter(ctx) {
			if event.Op.Has(fsnotify.Remove) || event.Op.Has(fsnotify.Rename) {
				if err := autoheal(event.Name); err != nil {
					hint(red.Paint(" ! "), " Unable to heal ", gray.Paint(event.Name), ", error: ", err)
				}
			}
		}
		if err := notifyHeal.Err(); err != nil {
			hint(red.Paint(" ! "), " Autoheal stopped working: ", err)
		}
	}()

	return notify, nil
}

func useSSE(srv *http.ServeMux, buildCh <-chan []error) {
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

	srv.HandleFunc("GET "+settings.LiveServer, hub.Handler(&shared.SSEHandlerOpt{
		CrossOriginHeader: "*",
		HandleErr:         func(e error) { hint("[Live Server]", e) },
	}))

}

func each(ctx context.Context, reload time.Duration, rootdir, builddir string) (<-chan string, error) {
	var notify = make(chan string)

	go func() {
		var (
			assets = defAssets()
			tick   = time.NewTicker(reload)
			latest = time.Now().Unix()
		)
		defer tick.Stop()
		defer close(notify)

		for loop := true; loop; {
			select {
			case <-tick.C:
			case <-ctx.Done():
				return
			}

			for name := range assets {
				if _, err := os.Stat(name); err != nil {
					if err = autoheal(name); err != nil {
						hint(red.Paint(" ! "), " Unable to heal ", gray.Paint(name), ", error: ", err)
					}
				}
			}

			err := filepath.Walk(rootdir, func(path string, info os.FileInfo, e error) error {
				if e != nil {
					return e
				}

				if !info.IsDir() {
					if tfile := info.ModTime().Unix(); tfile > latest {
						latest = tfile
						select {
						case notify <- path:
						case <-ctx.Done():
							loop = false
							return filepath.SkipAll
						}
						return filepath.SkipAll
					}
				} else if path == builddir {
					return filepath.SkipDir
				}
				return nil
			})
			if err != nil {
				return
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

func ctxsend[T any](ctx context.Context, ch chan T, val T) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- val:
		return true
	}
}

func liveReload(ctx context.Context, ssesrv *http.ServeMux) chan []error {
	var (
		notify <-chan string
		err    error
	)
	if *settings.reload == 0 {
		notify, err = watch(ctx, settings.InputDir, settings.OutputDir)
	} else {
		notify, err = each(ctx, *settings.reload, settings.InputDir, settings.OutputDir)
	}

	if err != nil {
		errch := make(chan []error, 1)
		errch <- []error{err}
		close(errch)
		return errch
	}

	var (
		errch = make(chan []error)
		ssech chan []error
	)
	if ssesrv != nil {
		ssech = make(chan []error)
		useSSE(ssesrv, ssech)
	}

	go func() {
		defer close(errch)
		if ssech != nil {
			defer close(ssech)
		}

		prev := ""
		for range notify {
			errs, serr := []error{}, ""

			for err := range build() {
				errs = append(errs, err)
				serr += err.Error()
			}

			if serr != prev {
				ctxsend(ctx, errch, errs)
				if ssech != nil {
					ctxsend(ctx, ssech, errs)
				}
				prev = serr
			} else if serr == "" && ssech != nil {
				ctxsend(ctx, ssech, errs)
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

func useStopper(c *context.Context, onStopFns ...func(context.Context) error) (context.Context, context.CancelFunc) {
	if c == nil {
		bg := context.Background()
		c = &bg
	}
	ctx, stop := signal.NotifyContext(*c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		for _, fn := range onStopFns {
			if fn != nil {
				if err := fn(ctx); err != nil {
					hint(err)
				}
			}
		}

	}()
	return ctx, stop
}

func serveDir(port, outputdir string) (srv *http.Server, han *http.ServeMux) {
	if han == nil {
		if han = http.DefaultServeMux; han == nil {
			han = http.NewServeMux()
		}
	}
	han.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./"+outputdir))))
	srv = &http.Server{Addr: settings.port, Handler: han}
	return
}

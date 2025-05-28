package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type smap[K comparable, V any] sync.Map

func (s *smap[K, V]) Load(key K) (V, bool) {
	v, ok := (*sync.Map)(s).Load(key)
	return v.(V), ok
}

func (s *smap[K, V]) Store(key K, val V) {
	(*sync.Map)(s).Store(key, val)
}

func (s *smap[K, V]) Delete(key K) {
	(*sync.Map)(s).Delete(key)
}

func (s *smap[K, V]) Range(fn func(K, V) bool) {
	(*sync.Map)(s).Range(func(rk, rv any) bool {
		return fn(rk.(K), rv.(V))
	})
}

func cutExt(s string) string {
	return s[:len(s)-len(filepath.Ext(s))]
}

func getBody(link string) (content []byte, err error) {
	res, err := http.Get(link)
	if err != nil {
		return
	}

	if 200 <= res.StatusCode && res.StatusCode < 300 {
		return nil, errors.New("invalid status code '" + res.Status + "'")
	}

	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func cleanURL(link string) (string, error) {
	u, err := url.ParseRequestURI(link)
	if err != nil {
		return "", err
	}
	return u.RequestURI(), nil
}

func getContent(link string) (content []byte, err error) {
	if url, e := cleanURL(link); e == nil {
		return getBody(url)
	}
	return os.ReadFile(link)
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

func splitExt(name string) (base, ext string) {
	ext = strings.ToLower(filepath.Ext(name))
	base = name[:len(name)-len(ext)]
	if ext == ".html" {
		ext = strings.ToLower(filepath.Ext(base)) + ext
		base = name[:len(name)-len(ext)]
	}
	return
}

func validHTML(content string) error {
	var dec = xml.NewDecoder(strings.NewReader(content))
	dec.Strict = false
	return dec.Decode(new(interface{}))
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

func liveReload(success func(), fail func(string)) error {
	errch, prev := make(chan error), ""
	defer close(errch)

	var fn = func() error {
		serr := ""
		for err := range build() {
			serr += fmt.Sprintln(err)
		}
		if serr != prev {
			if serr == "" {
				success()
			} else {
				fail(serr)
			}
			prev = serr
		}
		return nil
	}

	if *settings.reload == 0 {
		return watch(settings.InputDir, settings.OutputDir, fn)
	}
	return each(*settings.reload, fn)
}

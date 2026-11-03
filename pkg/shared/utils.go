package shared

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"

	"github.com/fsnotify/fsnotify"
)

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

func CleanURL(link string) (string, error) {
	u, err := url.ParseRequestURI(link)
	if err != nil {
		return "", err
	}
	return u.RequestURI(), nil
}

func FetchContent(link string) (content []byte, err error) {
	if url, e := CleanURL(link); e == nil {
		return getBody(url)
	}
	return os.ReadFile(link)
}

func Compact[T comparable](list []T) []T {
	var res []T
	for _, item := range list {
		if !slices.Contains(res, item) {
			res = append(res, item)
		}
	}
	return res
}

func WatchDir(dir string, skip func(string) bool, stop <-chan bool) (<-chan fsnotify.Event, error) {
	var watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, e error) error {
		if e == nil && info.IsDir() {
			if skip != nil && skip(path) {
				return filepath.SkipDir
			}
			watcher.Add(path)
		}
		return e
	})
	if err != nil {
		return nil, err
	}

	notify := make(chan fsnotify.Event)
	go func() {
		defer close(notify)
		defer watcher.Close()
		for err == nil {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					continue
				}
				if event.Op.Has(fsnotify.Create) {
					info, statErr := os.Stat(event.Name)
					if statErr == nil && info.IsDir() && !skip(event.Name) {
						watcher.Add(event.Name)
					}
				}
				notify <- event
			case event, ok := <-watcher.Errors:
				if !ok {
					continue
				}
				err = event
			case event, ok := <-stop:
				if !ok {
					continue
				}
				if event {
					return
				}
			}
		}
	}()

	return notify, nil
}

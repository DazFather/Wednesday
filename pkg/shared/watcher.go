package shared

import (
	"context"
	"iter"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type DirWatcher struct {
	watcher chan fsnotify.Event
	stopper chan struct{}
}

func (fw *DirWatcher) Close() {
	fw.stopper <- struct{}{}
}

func (fw *DirWatcher) Observe() <-chan fsnotify.Event {
	return fw.watcher
}

func (fw *DirWatcher) emit(event fsnotify.Event) bool {
	select {
	case fw.watcher <- event:
	case <-fw.stopper:
		return true
	}
	return false
}

func (fw *DirWatcher) Iter(ctx context.Context) iter.Seq[fsnotify.Event] {
	return func(yield func(fsnotify.Event) bool) {
		defer fw.Close()
		for {
			select {
			case event, ok := <-fw.watcher:
				if !ok || !yield(event) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

func WatchDir(dir string, skip func(string) bool) (*DirWatcher, error) {
	var fsnw, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, e error) error {
		if e == nil && info.IsDir() {
			if skip != nil && skip(path) {
				return filepath.SkipDir
			}
			fsnw.Add(path)
		}
		return e
	})
	if err != nil {
		fsnw.Close()
		return nil, err
	}

	var watcher = DirWatcher{
		watcher: make(chan fsnotify.Event),
		stopper: make(chan struct{}),
	}

	go func() {
		defer fsnw.Close()
		for err == nil {
			select {
			case event, ok := <-fsnw.Events:
				if !ok {
					continue
				}
				if event.Op.Has(fsnotify.Create) {
					info, statErr := os.Stat(event.Name)
					if statErr == nil && info.IsDir() && (skip != nil || !skip(event.Name)) {
						fsnw.Add(event.Name)
					}
				}
				if watcher.emit(event) {
					return
				}
			case event, ok := <-fsnw.Errors:
				if !ok {
					continue
				}
				err = event
			case <-watcher.stopper:
				return
			}
		}
	}()

	return &watcher, nil
}

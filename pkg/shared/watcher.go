package shared

import (
	"context"
	"errors"
	"iter"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type DirWatcher struct {
	watcher chan fsnotify.Event
	stopper chan struct{}
	errch   chan error
	err     error
	closefn func()
}

func (fw DirWatcher) Err() error {
	return fw.err
}

func (fw *DirWatcher) Close() {
	fw.closefn()
}

func (fw *DirWatcher) emit(event fsnotify.Event) bool {
	select {
	case fw.watcher <- event:
	case <-fw.stopper:
		return true
	}
	return false
}

func (fw DirWatcher) Observe() <-chan fsnotify.Event {
	return fw.watcher
}

func (fw DirWatcher) Errors() <-chan error {
	return fw.errch
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
			case <-fw.errch:
				return
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

	err = filepath.WalkDir(dir, func(path string, info os.DirEntry, e error) error {
		if e == nil && info.IsDir() {
			if skip != nil && skip(path) {
				return filepath.SkipDir
			}
			e = fsnw.Add(path)
		}
		return e
	})
	if err != nil {
		_ = fsnw.Close()
		return nil, err
	}

	var watcher = &DirWatcher{
		watcher: make(chan fsnotify.Event),
		stopper: make(chan struct{}, 1),
		errch:   make(chan error),
	}
	watcher.closefn = sync.OnceFunc(func() {
		close(watcher.watcher)
		close(watcher.errch)
		watcher.stopper <- struct{}{}
		close(watcher.stopper)
	})

	go watcher.scan(fsnw, skip)

	return watcher, nil
}

func (fw *DirWatcher) scan(fsnw *fsnotify.Watcher, skip func(string) bool) {
	defer fsnw.Close()

	handleErr := func(err error) {
		if err == nil {
			return
		}
		fw.err = errors.Join(fw.err, err)
		fw.errch <- err
	}

	for {
		select {
		case event, ok := <-fsnw.Events:
			if !ok {
				return
			}
			if event.Op.Has(fsnotify.Create) {
				info, statErr := os.Stat(event.Name)
				if statErr == nil && info.IsDir() && (skip == nil || !skip(event.Name)) {
					handleErr(fsnw.Add(event.Name))
				}
			}
			if fw.emit(event) {
				return
			}
		case event, ok := <-fsnw.Errors:
			if !ok {
				continue
			}
			handleErr(event)
		case <-fw.stopper:
			return
		}
	}
}

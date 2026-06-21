// Package sync — watcher.go wires fsnotify events to the Service.
package sync

import (
	"log/slog"

	"github.com/fsnotify/fsnotify"
)

// Watcher wraps fsnotify and routes events to the Service.
type Watcher struct {
	service *Service
	logger  *slog.Logger
	inner   *fsnotify.Watcher
}

// NewWatcher creates a Watcher but does not start it yet.
func NewWatcher(service *Service, logger *slog.Logger) (*Watcher, error) {
	inner, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{service: service, logger: logger, inner: inner}, nil
}

// Add registers a directory (recursively) with the underlying fsnotify watcher.
func (w *Watcher) Add(dir string) error {
	return w.inner.Add(dir)
}

// Run blocks, dispatching events until the done channel is closed.
func (w *Watcher) Run(done <-chan struct{}) {
	for {
		select {
		case event, ok := <-w.inner.Events:
			if !ok {
				return
			}
			w.handle(event)

		case err, ok := <-w.inner.Errors:
			if !ok {
				return
			}
			w.logger.Error("watcher error", "err", err)

		case <-done:
			_ = w.inner.Close()
			return
		}
	}
}

func (w *Watcher) handle(event fsnotify.Event) {
	switch {
	case event.Has(fsnotify.Create) || event.Has(fsnotify.Write):
		w.service.SyncFile(event.Name)

	case event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename):
		w.service.RemoveFile(event.Name)
	}
}

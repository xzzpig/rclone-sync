// Package watcher provides file system watching functionality for realtime sync.
package watcher

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"go.uber.org/zap"
)

// RecursiveWatcher wraps fsnotify.Watcher to provide recursive directory watching
// and reference counting for shared paths.
type RecursiveWatcher struct {
	fsWatcher *fsnotify.Watcher
	logger    *zap.Logger
	mu        sync.Mutex

	// watchedDirs tracks usage count for each directory path
	// path -> count
	watchedDirs map[string]int

	// Events and Errors channels forward events from fsnotify
	events chan fsnotify.Event
	errors chan error

	done chan struct{}
}

// NewRecursiveWatcher creates a new RecursiveWatcher instance.
func NewRecursiveWatcher() (*RecursiveWatcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	rw := &RecursiveWatcher{
		fsWatcher:   fsWatcher,
		logger:      logger.L.Named("recursive_watcher"),
		watchedDirs: make(map[string]int),
		events:      make(chan fsnotify.Event),
		errors:      make(chan error),
		done:        make(chan struct{}),
	}

	go rw.loop()

	return rw, nil
}

// Events returns the channel for receiving file system events.
func (rw *RecursiveWatcher) Events() chan fsnotify.Event {
	return rw.events
}

// Errors returns the channel for receiving watcher errors.
func (rw *RecursiveWatcher) Errors() chan error {
	return rw.errors
}

// Close stops the watcher and releases resources.
func (rw *RecursiveWatcher) Close() error {
	close(rw.done)
	return rw.fsWatcher.Close()
}

// Add recursively watches a directory and its subdirectories
func (rw *RecursiveWatcher) Add(root string) error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Verify root exists
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errs.ConstError("path is not a directory")
	}

	// Walk and add all subdirectories
	return rw.addRecursiveLocked(root)
}

// Remove stops watching a directory (decrementing reference count)
// It recursively removes watches for subdirectories based on internal state.
func (rw *RecursiveWatcher) Remove(root string) error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// This handles cases where the directory was deleted before we could walk it.
	root = filepath.Clean(root)
	for path := range rw.watchedDirs {
		if path == root || strings.HasPrefix(path, root+string(os.PathSeparator)) {
			rw.removeDirLocked(path)
		}
	}

	return nil
}

// addRecursiveLocked recursively adds a directory and its subdirectories.
// Caller must hold rw.mu.
func (rw *RecursiveWatcher) addRecursiveLocked(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip errors for individual files/dirs but log them
			rw.logger.Warn("Error walking path", zap.String("path", path), zap.Error(err))
			return nil
		}

		if d.IsDir() {
			if err := rw.addDirLocked(path); err != nil {
				return err
			}
		}
		return nil
	})
}

// addDirLocked adds a directory to fsnotify and increments count
func (rw *RecursiveWatcher) addDirLocked(path string) error {
	// Check exclusion patterns (simple implementation for now)
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") && base != "." && base != ".." {
		// Ignore hidden directories like .git
		return filepath.SkipDir
	}

	count := rw.watchedDirs[path]
	rw.watchedDirs[path] = count + 1

	if count == 0 {
		if err := rw.fsWatcher.Add(path); err != nil {
			// If we fail to add, rollback count
			delete(rw.watchedDirs, path)
			return err
		}
		rw.logger.Debug("Added watch", zap.String("path", path))
	}
	return nil
}

// removeDirLocked decrements count and removes from fsnotify if 0
func (rw *RecursiveWatcher) removeDirLocked(path string) {
	count, ok := rw.watchedDirs[path]
	if !ok {
		return
	}

	if count <= 1 {
		_ = rw.fsWatcher.Remove(path)
		delete(rw.watchedDirs, path)
		rw.logger.Debug("Removed watch", zap.String("path", path))
	} else {
		rw.watchedDirs[path] = count - 1
	}
}

func (rw *RecursiveWatcher) loop() {
	defer close(rw.events)
	defer close(rw.errors)

	for {
		select {
		case <-rw.done:
			return
		case event, ok := <-rw.fsWatcher.Events:
			if !ok {
				return
			}

			// Filter logic
			if rw.shouldIgnore(event.Name) {
				continue
			}

			// Handle dynamic directory creation/removal
			if event.Op&fsnotify.Create == fsnotify.Create {
				// If it's a directory, add it recursively
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					rw.mu.Lock()
					_ = rw.addRecursiveLocked(event.Name)
					rw.mu.Unlock()
				}
			}
			// Note: fsnotify automatically removes watches for deleted directories
			// But we should update our map
			if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
				// We don't know if it was a directory easily without stat (which will fail)
				// But we can check our map
				rw.mu.Lock()
				if _, ok := rw.watchedDirs[event.Name]; ok {
					delete(rw.watchedDirs, event.Name)
					// Also, should we remove subdirectories from our map?
					// If a parent is removed, all children are removed effectively.
					// We should iterate map and remove keys with this prefix.
					for path := range rw.watchedDirs {
						if strings.HasPrefix(path, event.Name+string(os.PathSeparator)) {
							delete(rw.watchedDirs, path)
						}
					}
				}
				rw.mu.Unlock()
			}

			// Forward event
			rw.events <- event

		case err, ok := <-rw.fsWatcher.Errors:
			if !ok {
				return
			}
			rw.errors <- err
		}
	}
}

func (rw *RecursiveWatcher) shouldIgnore(path string) bool {
	// Basic filtering
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") && base != "." && base != ".." {
		return true
	}
	// TODO: Add more ignore patterns (tmp files etc)
	return false
}

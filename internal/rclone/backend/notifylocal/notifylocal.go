//go:build !plan9 && !js

// Package notifylocal provides a local filesystem backend with ChangeNotify support.
// It wraps rclone's local backend and uses RecursiveWatcher to detect file changes,
// making it suitable for testing metacache's ChangeNotify integration.
package notifylocal

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rclone/rclone/backend/local"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/hash"
	"github.com/xzzpig/rclone-sync/internal/core/watcher"
)

func init() {
	localRegInfo := fs.MustFind("local")
	fs.Register(&fs.RegInfo{
		Name:         "notifylocal",
		Description:  "Local filesystem with ChangeNotify support (for testing)",
		NewFs:        NewFs,
		Options:      localRegInfo.Options,
		MetadataInfo: localRegInfo.MetadataInfo,
	})
}

// Fs wraps rclone's local.Fs to add ChangeNotify support using RecursiveWatcher.
type Fs struct {
	fs.Fs
	name         string
	root         string
	features     *fs.Features
	watcher      *watcher.RecursiveWatcher
	notifyMu     sync.Mutex
	notifyFuncs  []func(string, fs.EntryType)
	watcherDone  chan struct{}
	shutdownOnce sync.Once
}

var (
	_ fs.Fs             = (*Fs)(nil)
	_ fs.ChangeNotifier = (*Fs)(nil)
	_ fs.UnWrapper      = (*Fs)(nil)
	_ fs.Wrapper        = (*Fs)(nil)
	_ fs.Shutdowner     = (*Fs)(nil)
)

// NewFs creates a new Fs wrapping a local filesystem with ChangeNotify support.
func NewFs(ctx context.Context, name, rootPath string, m configmap.Mapper) (fs.Fs, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	wrappedFs, wrapErr := local.NewFs(ctx, name, rootPath, m)
	if wrapErr != nil && !errors.Is(wrapErr, fs.ErrorIsFile) {
		return nil, wrapErr
	}

	rw, err := watcher.NewRecursiveWatcher()
	if err != nil {
		return nil, err
	}

	var fsErr error
	if errors.Is(wrapErr, fs.ErrorIsFile) {
		fsErr = fs.ErrorIsFile
		absRoot = filepath.Dir(absRoot)
	}

	f := &Fs{
		Fs:          wrappedFs,
		name:        name,
		root:        absRoot,
		watcher:     rw,
		watcherDone: make(chan struct{}),
	}

	f.features = (&fs.Features{
		CanHaveEmptyDirectories: true,
		BucketBased:             true,
	}).Fill(ctx, f).Mask(ctx, wrappedFs).WrapsFs(f, wrappedFs)

	f.features.ChangeNotify = f.ChangeNotify
	f.features.Shutdown = f.Shutdown
	f.features.BucketBased = true

	if _, statErr := os.Stat(absRoot); statErr == nil {
		if addErr := rw.Add(absRoot); addErr != nil {
			_ = rw.Close()
			return nil, addErr
		}
	}

	go f.watchLoop()

	fs.Debugf(f, "Created notifylocal Fs with ChangeNotify support at %s", absRoot)

	return f, fsErr
}

// Name of the remote (as passed into NewFs)
func (f *Fs) Name() string {
	return f.name
}

// Root of the remote (as passed into NewFs)
func (f *Fs) Root() string {
	return f.Fs.Root()
}

// String returns a description of the FS
func (f *Fs) String() string {
	return f.Fs.String()
}

// Features returns the optional features of this Fs
func (f *Fs) Features() *fs.Features {
	return f.features
}

// Precision returns the precision of this Fs
func (f *Fs) Precision() time.Duration {
	return f.Fs.Precision()
}

// Hashes returns the supported hash sets of this Fs
func (f *Fs) Hashes() hash.Set {
	return f.Fs.Hashes()
}

// UnWrap returns the Fs that this Fs is wrapping
func (f *Fs) UnWrap() fs.Fs {
	return f.Fs
}

// WrapFs returns the Fs that this Fs is wrapping
func (f *Fs) WrapFs() fs.Fs {
	return nil
}

// SetWrapper is called by the features manager
func (f *Fs) SetWrapper(_ fs.Fs) {}

// Mkdir creates the container if it doesn't exist
func (f *Fs) Mkdir(ctx context.Context, dir string) error {
	err := f.Fs.Mkdir(ctx, dir)
	if err != nil {
		return err
	}

	absDir := filepath.Join(f.root, dir)
	if addErr := f.watcher.Add(absDir); addErr != nil {
		fs.Debugf(f, "Failed to add watcher for %s: %v", absDir, addErr)
	}

	return nil
}

// ChangeNotify implements fs.ChangeNotifier interface.
func (f *Fs) ChangeNotify(_ context.Context, notifyFunc func(string, fs.EntryType), pollIntervalChan <-chan time.Duration) {
	f.notifyMu.Lock()
	f.notifyFuncs = append(f.notifyFuncs, notifyFunc)
	f.notifyMu.Unlock()

	fs.Debugf(f, "Registered ChangeNotify callback")

	go func() {
		for {
			select {
			case <-f.watcherDone:
				return
			case _, ok := <-pollIntervalChan:
				if !ok {
					return
				}
			}
		}
	}()
}

func (f *Fs) watchLoop() {
	defer close(f.watcherDone)

	events := f.watcher.Events()
	errs := f.watcher.Errors()

	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			f.handleEvent(event)

		case err, ok := <-errs:
			if !ok {
				return
			}
			fs.Errorf(f, "Watcher error: %v", err)
		}
	}
}

func (f *Fs) handleEvent(event fsnotify.Event) {
	if event.Op&fsnotify.Chmod == fsnotify.Chmod {
		return
	}

	relPath, err := filepath.Rel(f.root, event.Name)
	if err != nil {
		fs.Debugf(f, "Failed to get relative path for %s: %v", event.Name, err)
		return
	}

	relPath = filepath.ToSlash(relPath)
	if relPath == "." {
		relPath = ""
	}

	entryType := fs.EntryObject
	if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
		info, statErr := os.Stat(event.Name)
		if statErr == nil && info.IsDir() {
			entryType = fs.EntryDirectory
		}
	}

	fs.Debugf(f, "ChangeNotify: %s (%v) -> %s", event.Name, event.Op, relPath)

	f.notifyMu.Lock()
	notifyFuncs := f.notifyFuncs
	f.notifyMu.Unlock()

	for _, fn := range notifyFuncs {
		fn(relPath, entryType)
	}
}

// Shutdown implements fs.Shutdowner interface.
func (f *Fs) Shutdown(ctx context.Context) error {
	var err error
	f.shutdownOnce.Do(func() {
		fs.Debugf(f, "Shutting down notifylocal")

		if f.watcher != nil {
			if closeErr := f.watcher.Close(); closeErr != nil {
				fs.Errorf(f, "Error closing watcher: %v", closeErr)
			}
		}

		select {
		case <-f.watcherDone:
		case <-time.After(5 * time.Second):
			fs.Errorf(f, "Timeout waiting for watcher loop to finish")
		case <-ctx.Done():
			err = ctx.Err()
			return
		}

		if shutdowner, ok := f.Fs.(fs.Shutdowner); ok {
			err = shutdowner.Shutdown(ctx)
		}
	})
	return err
}

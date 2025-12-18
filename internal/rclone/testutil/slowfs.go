// Package testutil provides a slowfs backend for testing purposes.
// slowfs wraps a local filesystem and adds controllable delays to operations.
package testutil

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/cache"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/configstruct"
	"github.com/rclone/rclone/fs/fspath"
	"github.com/rclone/rclone/fs/hash"
)

// slowFsController provides global control channels for slowfs instances
var slowFsController = struct {
	mu        sync.RWMutex
	startedCh chan struct{} // Notifies when an operation starts
	blockCh   chan struct{} // Controls when to unblock operations
}{
	startedCh: nil,
	blockCh:   nil,
}

// SetSlowFsController sets the control channels for slowfs testing
func SetSlowFsController(startedCh chan struct{}, blockCh chan struct{}) {
	slowFsController.mu.Lock()
	defer slowFsController.mu.Unlock()
	slowFsController.startedCh = startedCh
	slowFsController.blockCh = blockCh
}

// ClearSlowFsController clears the control channels
func ClearSlowFsController() {
	slowFsController.mu.Lock()
	defer slowFsController.mu.Unlock()
	slowFsController.startedCh = nil
	slowFsController.blockCh = nil
}

// notifyStarted notifies that an operation has started (non-blocking)
func notifyStarted() {
	slowFsController.mu.RLock()
	ch := slowFsController.startedCh
	slowFsController.mu.RUnlock()

	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// waitForUnblock waits for permission to continue or context cancellation
func waitForUnblock(ctx context.Context) error {
	slowFsController.mu.RLock()
	ch := slowFsController.blockCh
	slowFsController.mu.RUnlock()

	if ch == nil {
		return nil // No blocking configured
	}

	select {
	case <-ch:
		return nil // Unblocked
	case <-ctx.Done():
		return ctx.Err() // Cancelled
	}
}

// Options defines the configuration for slowfs backend
type slowFsOptions struct {
	Remote string `config:"remote"`
}

// SlowFs wraps a real Fs and adds controllable delays
type SlowFs struct {
	name     string
	root     string
	wrapped  fs.Fs
	features *fs.Features
}

// Name of the remote (as passed into NewFs)
func (f *SlowFs) Name() string { return f.name }

// Root of the remote (as passed into NewFs)
func (f *SlowFs) Root() string { return f.root }

// String returns a description of the FS
func (f *SlowFs) String() string { return f.name + ":" + f.root }

// Precision of the ModTimes in this Fs
func (f *SlowFs) Precision() time.Duration { return f.wrapped.Precision() }

// Hashes returns the supported hash types of the filesystem
func (f *SlowFs) Hashes() hash.Set { return f.wrapped.Hashes() }

// Features returns the optional features of this Fs
func (f *SlowFs) Features() *fs.Features { return f.features }

// List the objects and directories in dir into entries
func (f *SlowFs) List(ctx context.Context, dir string) (fs.DirEntries, error) {
	return f.wrapped.List(ctx, dir)
}

// NewObject finds the Object at remote
func (f *SlowFs) NewObject(ctx context.Context, remote string) (fs.Object, error) {
	o, err := f.wrapped.NewObject(ctx, remote)
	if err != nil {
		return nil, err
	}
	return &SlowObject{Object: o, fs: f}, nil
}

// Put in to the remote path with the modTime given of the given size
func (f *SlowFs) Put(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	// Notify that we've started
	notifyStarted()

	// Wait for permission to continue or cancellation
	if err := waitForUnblock(ctx); err != nil {
		return nil, err
	}

	o, err := f.wrapped.Put(ctx, in, src, options...)
	if err != nil {
		return nil, err
	}
	return &SlowObject{Object: o, fs: f}, nil
}

// Mkdir makes the directory (container, bucket)
func (f *SlowFs) Mkdir(ctx context.Context, dir string) error {
	return f.wrapped.Mkdir(ctx, dir)
}

// Rmdir removes the directory (container, bucket) if empty
func (f *SlowFs) Rmdir(ctx context.Context, dir string) error {
	return f.wrapped.Rmdir(ctx, dir)
}

// SlowObject wraps an Object and adds controllable delays
type SlowObject struct {
	fs.Object
	fs *SlowFs
}

// Fs returns the parent Fs
func (o *SlowObject) Fs() fs.Info { return o.fs }

// Update in to the object with the modTime given of the given size
func (o *SlowObject) Update(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) error {
	// Notify that we've started
	notifyStarted()

	// Wait for permission to continue or cancellation
	if err := waitForUnblock(ctx); err != nil {
		return err
	}

	return o.Object.Update(ctx, in, src, options...)
}

// Remove this object
func (o *SlowObject) Remove(ctx context.Context) error {
	return o.Object.Remove(ctx)
}

// Open opens the file for read
func (o *SlowObject) Open(ctx context.Context, options ...fs.OpenOption) (io.ReadCloser, error) {
	return o.Object.Open(ctx, options...)
}

// NewSlowFs constructs a SlowFs from the path
func NewSlowFs(ctx context.Context, name, root string, m configmap.Mapper) (fs.Fs, error) {
	opt := new(slowFsOptions)
	err := configstruct.Set(m, opt)
	if err != nil {
		return nil, err
	}

	remotePath := fspath.JoinRootPath(opt.Remote, root)
	wrappedFs, err := cache.Get(ctx, remotePath)
	if err != nil {
		return nil, err
	}

	f := &SlowFs{
		name:    name,
		root:    root,
		wrapped: wrappedFs,
	}

	// Copy features from wrapped fs, but use our own implementation
	f.features = &fs.Features{
		CaseInsensitive:          wrappedFs.Features().CaseInsensitive,
		DuplicateFiles:           wrappedFs.Features().DuplicateFiles,
		ReadMimeType:             wrappedFs.Features().ReadMimeType,
		WriteMimeType:            wrappedFs.Features().WriteMimeType,
		CanHaveEmptyDirectories:  wrappedFs.Features().CanHaveEmptyDirectories,
		BucketBased:              wrappedFs.Features().BucketBased,
		BucketBasedRootOK:        wrappedFs.Features().BucketBasedRootOK,
		SetTier:                  wrappedFs.Features().SetTier,
		GetTier:                  wrappedFs.Features().GetTier,
		ServerSideAcrossConfigs:  wrappedFs.Features().ServerSideAcrossConfigs,
		IsLocal:                  wrappedFs.Features().IsLocal,
		SlowModTime:              wrappedFs.Features().SlowModTime,
		SlowHash:                 wrappedFs.Features().SlowHash,
		ReadMetadata:             wrappedFs.Features().ReadMetadata,
		WriteMetadata:            wrappedFs.Features().WriteMetadata,
		UserMetadata:             wrappedFs.Features().UserMetadata,
		ReadDirMetadata:          wrappedFs.Features().ReadDirMetadata,
		WriteDirMetadata:         wrappedFs.Features().WriteDirMetadata,
		WriteDirSetModTime:       wrappedFs.Features().WriteDirSetModTime,
		UserDirMetadata:          wrappedFs.Features().UserDirMetadata,
		DirModTimeUpdatesOnWrite: wrappedFs.Features().DirModTimeUpdatesOnWrite,
		FilterAware:              wrappedFs.Features().FilterAware,
		PartialUploads:           wrappedFs.Features().PartialUploads,
		NoMultiThreading:         wrappedFs.Features().NoMultiThreading,
		Overlay:                  wrappedFs.Features().Overlay,
	}

	return f, nil
}

// registerSlowFs registers the slowfs backend with rclone
func init() {
	fsi := &fs.RegInfo{
		Name:        "slowfs",
		Description: "Slow filesystem for testing (wraps another remote with controllable delays)",
		NewFs:       NewSlowFs,
		Options: []fs.Option{{
			Name:     "remote",
			Help:     "Remote to wrap (e.g., '/local/path' or 'myremote:path').",
			Required: true,
		}},
	}
	fs.Register(fsi)
}

var _ fs.Fs = (*SlowFs)(nil)
var _ fs.Object = (*SlowObject)(nil)

package watcher

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"go.uber.org/zap"
)

// FileWatcher defines the interface for watching files.
type FileWatcher interface {
	Add(string) error
	Remove(string) error
	Close() error
	Events() chan fsnotify.Event
	Errors() chan error
}

type Watcher struct {
	recWatcher FileWatcher
	taskSvc    ports.TaskService
	runner     ports.Runner
	logger     *zap.Logger
	mu         sync.Mutex
	watchMap   map[string]string // Maps task ID to source path
	debounce   map[string]*time.Timer
	running    bool
}

func NewWatcher(taskSvc ports.TaskService, runner ports.Runner) (*Watcher, error) {
	recWatcher, err := NewRecursiveWatcher()
	if err != nil {
		return nil, err
	}
	return newWatcher(taskSvc, runner, recWatcher), nil
}

func newWatcher(taskSvc ports.TaskService, runner ports.Runner, fw FileWatcher) *Watcher {
	return &Watcher{
		recWatcher: fw,
		taskSvc:    taskSvc,
		runner:     runner,
		logger:     logger.L.Named("watcher"),
		watchMap:   make(map[string]string),
		debounce:   make(map[string]*time.Timer),
	}
}

// Start begins the file watching process. It is idempotent and will do nothing
// if the watcher is already running. Once the watcher is stopped with Stop(),
// it cannot be restarted.
func (w *Watcher) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.recWatcher == nil {
		w.logger.Warn("Watcher has been stopped and cannot be restarted")
		return
	}
	if w.running {
		w.logger.Info("Watcher is already running")
		return
	}

	w.logger.Info("Starting file watcher")
	w.loadWatchTasks()
	go w.watchLoop()
	w.running = true
}

// Stop halts the file watching process. It is idempotent. Once stopped, the
// watcher cannot be restarted.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.recWatcher == nil {
		return // Already stopped
	}
	if !w.running {
		w.logger.Info("Watcher is not running")
		return
	}
	w.logger.Info("Stopping file watcher")
	w.recWatcher.Close()
	w.recWatcher = nil
	w.running = false
}

func (w *Watcher) loadWatchTasks() {
	w.logger.Info("Loading realtime tasks from database")
	tasks, err := w.taskSvc.ListAllTasks(context.Background())
	if err != nil {
		w.logger.Error("Failed to load tasks for watcher", zap.Error(err))
		return
	}

	for _, task := range tasks {
		if task.Realtime {
			if err := w.addWatch(task); err != nil {
				w.logger.Error("Failed to add path to watcher on load",
					zap.String("task_id", task.ID.String()),
					zap.String("path", task.SourcePath),
					zap.Error(err),
				)
			}
		}
	}
	w.logger.Info("Finished loading realtime tasks", zap.Int("count", len(w.watchMap)))
}

func (w *Watcher) AddTask(task *ent.Task) error {
	if !task.Realtime {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.addWatch(task)
}

func (w *Watcher) RemoveTask(task *ent.Task) error {
	if !task.Realtime {
		// Even if not realtime now, it might have been before?
		// We probably need to check if we are watching it.
		// For now assume caller knows current state or check watchMap
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.removeWatch(task.ID.String())
	return nil
}

func (w *Watcher) addWatch(task *ent.Task) error {
	taskIDStr := task.ID.String()

	// If already watching, remove first (handle updates)
	if _, ok := w.watchMap[taskIDStr]; ok {
		w.removeWatch(taskIDStr)
	}

	err := w.recWatcher.Add(task.SourcePath)
	if err != nil {
		return err
	}

	w.watchMap[taskIDStr] = task.SourcePath
	w.logger.Info("Added path to watcher", zap.String("task", task.Name), zap.String("path", task.SourcePath))
	return nil
}

func (w *Watcher) removeWatch(taskID string) {
	if path, ok := w.watchMap[taskID]; ok {
		w.recWatcher.Remove(path)
		delete(w.watchMap, taskID)
		w.logger.Info("Removed path from watcher", zap.String("task_id", taskID), zap.String("path", path))
	}
}

func (w *Watcher) watchLoop() {
	// Capture the watcher to avoid race condition when Stop() sets w.recWatcher to nil
	rw := w.recWatcher
	if rw == nil {
		return
	}

	for {
		select {
		case event, ok := <-rw.Events():
			if !ok {
				return
			}
			// Ignore CHMOD events to reduce noise
			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			}
			w.handleEvent(event)
		case err, ok := <-rw.Errors():
			if !ok {
				return
			}
			w.logger.Error("Watcher error", zap.Error(err))
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Find which task(s) match this event path
	// This is a simple linear search. For many tasks, a reverse map or trie would be better.
	// TODO: Optimize task lookup for better performance with many tasks
	for taskID, sourcePath := range w.watchMap {
		// Check if event path is within source path
		// We use filepath.Rel to check if event.Name is relative to sourcePath
		// If Rel returns error or starts with "..", it's not inside.
		rel, err := filepath.Rel(sourcePath, event.Name)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(rel, "..") {
			w.triggerSync(taskID)
		}
	}
}

func (w *Watcher) triggerSync(taskID string) {
	// Debounce logic
	if timer, ok := w.debounce[taskID]; ok {
		timer.Stop()
	}

	// Wait 2 seconds after last event before syncing
	w.debounce[taskID] = time.AfterFunc(2*time.Second, func() {
		w.logger.Info("Triggering realtime sync", zap.String("task_id", taskID))

		ctx := context.Background()
		id, err := uuid.Parse(taskID)
		if err != nil {
			w.logger.Error("Failed to parse task ID", zap.String("task_id", taskID), zap.Error(err))
			return
		}

		task, err := w.taskSvc.GetTask(ctx, id)
		if err != nil {
			w.logger.Error("Failed to get task for sync", zap.String("task_id", taskID), zap.Error(err))
			return
		}

		w.runner.StartTask(task, "realtime")
	})
}

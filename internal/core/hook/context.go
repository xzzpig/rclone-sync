package hook

import (
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TaskInfo contains basic information about a sync task.
type TaskInfo struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	SourcePath string    `json:"sourcePath"`
	RemotePath string    `json:"remotePath"`
	Direction  string    `json:"direction"`
}

// JobInfo contains basic information about a sync job.
type JobInfo struct {
	ID        uuid.UUID `json:"id"`
	Status    string    `json:"status"`
	Trigger   string    `json:"trigger"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// TransferStats contains statistics about files and bytes transferred.
type TransferStats struct {
	FilesTransferred int64 `json:"filesTransferred"`
	BytesTransferred int64 `json:"bytesTransferred"`
	FilesDeleted     int64 `json:"filesDeleted"`
	ErrorCount       int64 `json:"errorCount"`
}

// Context provides context information for hook execution and template rendering.
type Context struct {
	Task     TaskInfo          `json:"task"`
	Job      JobInfo           `json:"job"`
	Event    string            `json:"event"`
	Error    string            `json:"error"`
	Duration time.Duration     `json:"duration"`
	Stats    TransferStats     `json:"stats"`
	Env      map[string]string `json:"env"`
}

// BuildContext creates a new Context from individual fields.
func BuildContext(
	taskID uuid.UUID, taskName, sourcePath, remotePath, direction string,
	jobID uuid.UUID, jobStatus, trigger string, startTime, endTime time.Time,
	event string, errMsg string, duration time.Duration,
	filesTransferred, bytesTransferred, filesDeleted, errorCount int64,
) *Context {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	return &Context{
		Task: TaskInfo{
			ID:         taskID,
			Name:       taskName,
			SourcePath: sourcePath,
			RemotePath: remotePath,
			Direction:  direction,
		},
		Job: JobInfo{
			ID:        jobID,
			Status:    jobStatus,
			Trigger:   trigger,
			StartTime: startTime,
			EndTime:   endTime,
		},
		Event:    event,
		Error:    errMsg,
		Duration: duration,
		Stats: TransferStats{
			FilesTransferred: filesTransferred,
			BytesTransferred: bytesTransferred,
			FilesDeleted:     filesDeleted,
			ErrorCount:       errorCount,
		},
		Env: env,
	}
}

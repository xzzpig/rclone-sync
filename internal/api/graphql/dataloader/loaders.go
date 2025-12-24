// Package dataloader provides dataloaders for efficient batch loading of data.
package dataloader

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
)

type ctxKey string

const loadersKey ctxKey = "dataloaders"

// Loaders holds all dataloaders for a request.
type Loaders struct {
	ConnectionLoader *ConnectionLoader
	TaskLoader       *TaskLoader
	JobLoader        *JobLoader
}

// NewLoaders creates a new Loaders instance for the request.
func NewLoaders(client *ent.Client) *Loaders {
	return &Loaders{
		ConnectionLoader: NewConnectionLoader(client),
		TaskLoader:       NewTaskLoader(client),
		JobLoader:        NewJobLoader(client),
	}
}

// Middleware injects dataloaders into the request context.
func Middleware(client *ent.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		loaders := NewLoaders(client)
		ctx := context.WithValue(c.Request.Context(), loadersKey, loaders)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// For retrieves the dataloaders from context.
func For(ctx context.Context) *Loaders {
	loaders, ok := ctx.Value(loadersKey).(*Loaders)
	if !ok {
		panic("dataloader: loaders not found in context - did you forget to add dataloader.Middleware?")
	}
	return loaders
}

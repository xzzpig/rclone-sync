// Package resolver provides GraphQL resolver implementations.
package resolver

//go:generate go run github.com/99designs/gqlgen generate
//go:generate node ../../../../scripts/merge-schema.js

import (
	"github.com/xzzpig/rclone-sync/internal/api/graphql/generated"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/subscription"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Dependencies holds all dependencies required by resolvers.
type Dependencies struct {
	SyncEngine          *rclone.SyncEngine
	Runner              ports.Runner
	Watcher             ports.Watcher
	Scheduler           ports.Scheduler
	Encryptor           *crypto.Encryptor
	JobProgressBus      *subscription.JobProgressBus
	TransferProgressBus *subscription.TransferProgressBus
	ConnectionService   *services.ConnectionService
	TaskService         *services.TaskService
	JobService          *services.JobService
}

// Resolver is the root resolver that holds all dependencies.
type Resolver struct {
	deps *Dependencies
}

// New creates a new Resolver with the given dependencies.
func New(deps *Dependencies) *Resolver {
	return &Resolver{deps: deps}
}

// Ensure Resolver implements the generated ResolverRoot interface.
var _ generated.ResolverRoot = (*Resolver)(nil)

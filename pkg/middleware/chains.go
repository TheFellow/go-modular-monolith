package middleware

import (
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

type PipelineConfig struct {
	Store            *store.Store
	Dispatcher       EventDispatcher
	Metrics          telemetry.Metrics
	ActivityRecorder ActivityRecorder
}

type Pipeline struct {
	store   *store.Store
	query   *Chain
	command *Chain
}

func NewPipeline(config PipelineConfig) *Pipeline {
	return &Pipeline{
		store: config.Store,
		query: NewChain(
			Logging(),
			Metrics(config.Metrics),
			Authorize(),
		),
		command: NewChain(
			Logging(),
			Metrics(config.Metrics),
			TrackActivity(config.Store, config.ActivityRecorder),
			UnitOfWork(config.Store),
			DispatchEvents(config.Dispatcher),
		),
	}
}

var defaultPipeline = NewPipeline(PipelineConfig{})

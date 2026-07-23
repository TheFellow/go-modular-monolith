package middleware

import (
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
)

type PipelineConfig struct {
	Store          *store.Store
	Dispatcher     EventDispatcher
	Metrics        telemetry.Metrics
	RecordActivity func(*Context, middlewareevents.Activity) error
}

type Pipeline struct {
	query   *Chain
	command *Chain
}

func NewPipeline(config PipelineConfig) *Pipeline {
	return &Pipeline{
		query: NewChain(
			SerializeTransaction(),
			Logging(),
			Metrics(config.Metrics),
		),
		command: NewChain(
			SerializeTransaction(),
			Logging(),
			Metrics(config.Metrics),
			TrackActivity(config.Store, config.RecordActivity),
			UnitOfWork(config.Store),
			recordSuccessfulActivity(config.RecordActivity),
			DispatchEvents(config.Dispatcher),
		),
	}
}

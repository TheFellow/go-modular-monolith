package dao

import (
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"time"

	cedar "github.com/cedar-policy/cedar-go"
)

type AuditEntryRow struct {
	ID           string
	WorkflowID   string
	Effects      []middlewareevents.Effect
	Participants []cedar.EntityUID
	Revision     uint64 `json:"-" store:"revision"`

	Action string `store:"index"`

	ResourceType string `store:"index"`
	ResourceID   string `store:"index"`

	PrincipalType string `store:"index"`
	PrincipalID   string `store:"index"`

	Touches []cedar.EntityUID

	StartedAt   time.Time `store:"index"`
	CompletedAt time.Time

	Success bool `store:"index"`
	Error   string
}

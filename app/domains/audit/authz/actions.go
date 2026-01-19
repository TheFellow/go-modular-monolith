package authz

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	cedar "github.com/cedar-policy/cedar-go"
)

const AuditAction cedar.EntityType = entity.TypeAuditEntry + "::Action"

var (
	ActionList = cedar.NewEntityUID(AuditAction, "list")
	ActionGet  = cedar.NewEntityUID(AuditAction, "get")
)

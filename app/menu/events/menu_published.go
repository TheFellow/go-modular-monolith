package events

import (
	"time"

	cedar "github.com/cedar-policy/cedar-go"
)

type MenuPublished struct {
	MenuID      cedar.EntityUID
	PublishedAt time.Time
}

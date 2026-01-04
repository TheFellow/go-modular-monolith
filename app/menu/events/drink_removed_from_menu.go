package events

import cedar "github.com/cedar-policy/cedar-go"

type DrinkRemovedFromMenu struct {
	MenuID  cedar.EntityUID
	DrinkID cedar.EntityUID
}

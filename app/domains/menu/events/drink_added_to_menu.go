package events

import cedar "github.com/cedar-policy/cedar-go"

type DrinkAddedToMenu struct {
	MenuID  cedar.EntityUID
	DrinkID cedar.EntityUID
}

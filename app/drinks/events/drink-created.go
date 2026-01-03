package events

import cedar "github.com/cedar-policy/cedar-go"

type DrinkCreated struct {
	DrinkID cedar.EntityUID
	Name    string
}

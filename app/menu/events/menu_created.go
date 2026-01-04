package events

import cedar "github.com/cedar-policy/cedar-go"

type MenuCreated struct {
	MenuID cedar.EntityUID
	Name   string
}

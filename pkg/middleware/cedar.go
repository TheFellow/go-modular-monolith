package middleware

import cedar "github.com/cedar-policy/cedar-go"

type CedarEntity interface {
	CedarEntity() cedar.Entity
}

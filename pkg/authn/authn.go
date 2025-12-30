package authn

import (
	"fmt"
	"strings"

	cedar "github.com/cedar-policy/cedar-go"
)

func Anonymous() cedar.EntityUID {
	return cedar.NewEntityUID(cedar.EntityType("Mixology::Actor"), cedar.String("anonymous"))
}

func Owner() cedar.EntityUID {
	return cedar.NewEntityUID(cedar.EntityType("Mixology::Actor"), cedar.String("owner"))
}

func ParseActor(s string) (cedar.EntityUID, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "owner":
		return Owner(), nil
	case "anonymous", "anon":
		return Anonymous(), nil
	default:
		return cedar.EntityUID{}, fmt.Errorf("unknown actor: %q", s)
	}
}

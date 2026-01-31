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

func Manager() cedar.EntityUID {
	return cedar.NewEntityUID(cedar.EntityType("Mixology::Actor"), cedar.String("manager"))
}

func Sommelier() cedar.EntityUID {
	return cedar.NewEntityUID(cedar.EntityType("Mixology::Actor"), cedar.String("sommelier"))
}

func Bartender() cedar.EntityUID {
	return cedar.NewEntityUID(cedar.EntityType("Mixology::Actor"), cedar.String("bartender"))
}

func ParseActor(s string) (cedar.EntityUID, error) {
	actor := strings.ToLower(strings.TrimSpace(s))
	switch actor {
	case "", "owner":
		return Owner(), nil
	case "manager":
		return Manager(), nil
	case "anonymous", "anon":
		return Anonymous(), nil
	case "sommelier":
		return Sommelier(), nil
	case "bartender":
		return Bartender(), nil
	default:
		return cedar.EntityUID{}, fmt.Errorf("unknown actor: %q", s)
	}
}

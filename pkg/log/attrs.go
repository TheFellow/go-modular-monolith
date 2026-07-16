package log

import (
	"log/slog"

	"github.com/cedar-policy/cedar-go"
)

func Actor(p cedar.EntityUID) slog.Attr {
	return slog.String("actor", p.String())
}

func Action(a cedar.EntityUID) slog.Attr {
	return slog.String("action", a.String())
}

func Resource(r cedar.EntityUID) slog.Attr {
	return slog.String("resource", r.String())
}

func EventType(name string) slog.Attr {
	return slog.String("event_type", name)
}

func Err(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "")
	}
	return slog.String("error", err.Error())
}

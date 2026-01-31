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

func Domain(name string) slog.Attr {
	return slog.String("domain", name)
}

func EventType(name string) slog.Attr {
	return slog.String("event_type", name)
}

func RequestID(id string) slog.Attr {
	return slog.String("request_id", id)
}

func Allowed(v bool) slog.Attr {
	return slog.Bool("allowed", v)
}

func Err(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.Any("error", err)
}

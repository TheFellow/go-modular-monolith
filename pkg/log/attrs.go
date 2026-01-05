package log

import (
	"log/slog"
	"time"

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

func Duration(d time.Duration) slog.Attr {
	return slog.Duration("duration", d)
}

func Allowed(v bool) slog.Attr {
	return slog.Bool("allowed", v)
}

func Err(err error) slog.Attr {
	return slog.Any("err", err)
}

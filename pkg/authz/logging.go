package authz

import (
	"log/slog"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/log"
)

// logDecision logs an authorization decision.
// The logger is expected to already include action/resource from upstream middleware.
func logDecision(logger *slog.Logger, allowed bool, duration time.Duration, err error) {
	if logger == nil {
		logger = slog.Default()
	}

	if err != nil {
		logger.Warn("authorization error",
			log.Allowed(allowed),
			slog.Duration("duration", duration),
			log.Err(err),
		)
		return
	}

	if !allowed {
		logger.Info("authorization denied",
			log.Allowed(allowed),
			slog.Duration("duration", duration),
		)
		return
	}

	logger.Debug("authorization allowed",
		log.Allowed(allowed),
		slog.Duration("duration", duration),
	)
}

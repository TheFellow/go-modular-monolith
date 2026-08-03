// Package runtimeconfig defines the process-level settings shared by every
// Mixology entry point. Presentation-specific flag libraries remain at the
// executable edge; their names, environment variables, and defaults do not.
package runtimeconfig

const (
	DefaultDatabasePath = "data/mixology.db"
	DefaultActor        = "owner"
	DefaultLogLevel     = "info"
	DefaultLogFormat    = "text"
	DefaultMetricsAddr  = ":9090"

	EnvDatabasePath = "MIXOLOGY_DB"
	EnvActor        = "MIXOLOGY_ACTOR"
	EnvDataDir      = "MIXOLOGY_DATA_DIR"
	EnvLogLevel     = "MIXOLOGY_LOG_LEVEL"
	EnvLogFormat    = "MIXOLOGY_LOG_FORMAT"
	EnvLogFile      = "MIXOLOGY_LOG_FILE"
	EnvMetrics      = "MIXOLOGY_METRICS"
)

// Config is the common runtime contract. An executable may choose not to
// expose every setting, but shared settings keep the same meaning everywhere.
type Config struct {
	DatabasePath  string
	Actor         string
	LogLevel      string
	LogFormat     string
	LogFile       string
	EnableMetrics bool
	MetricsAddr   string
}

func Default() Config {
	return Config{
		DatabasePath: DefaultDatabasePath,
		Actor:        DefaultActor,
		LogLevel:     DefaultLogLevel,
		LogFormat:    DefaultLogFormat,
		MetricsAddr:  DefaultMetricsAddr,
	}
}

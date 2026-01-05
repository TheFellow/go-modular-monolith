package telemetry

const (
	MetricCommandTotal    = "mixology_command_total"
	MetricCommandDuration = "mixology_command_duration_seconds"
	MetricCommandErrors   = "mixology_command_errors_total"

	MetricQueryTotal    = "mixology_query_total"
	MetricQueryDuration = "mixology_query_duration_seconds"
	MetricQueryErrors   = "mixology_query_errors_total"

	MetricAuthZTotal   = "mixology_authz_decisions_total"
	MetricAuthZDenied  = "mixology_authz_denied_total"
	MetricAuthZLatency = "mixology_authz_duration_seconds"

	MetricEventsDispatched = "mixology_events_dispatched_total"
	MetricEventsDuration   = "mixology_events_duration_seconds"
	MetricEventsErrors     = "mixology_events_errors_total"

	MetricStoreReadDuration  = "mixology_store_read_duration_seconds"
	MetricStoreWriteDuration = "mixology_store_write_duration_seconds"
)

const (
	LabelDomain    = "domain"
	LabelAction    = "action"
	LabelEventType = "event_type"
	LabelResult    = "result"
	LabelDecision  = "decision"
)

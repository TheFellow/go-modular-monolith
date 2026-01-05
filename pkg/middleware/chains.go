package middleware

var (
	Query = NewQueryChain(
		QueryLogging(),
		QueryMetrics(),
		QueryAuthorize(
			AuthZLogging(),
			AuthZMetrics(),
		),
	)
	QueryWithResource = NewQueryWithResourceChain(
		QueryWithResourceLogging(),
		QueryWithResourceMetrics(),
		QueryWithResourceAuthorize(
			AuthZLogging(),
			AuthZMetrics(),
		),
	)
	Command = NewCommandChain(
		CommandLogging(),
		CommandMetrics(),
		CommandAuthorize(
			AuthZLogging(),
			AuthZMetrics(),
		),
		UnitOfWork(),
		DispatchEvents(
			EventMetrics(),
		),
	)
)

package middleware

var (
	Query = NewQueryChain(
		QueryLogging(),
		QueryMetrics(),
		QueryAuthorize(),
	)
	QueryWithResource = NewQueryWithResourceChain(
		QueryWithResourceLogging(),
		QueryWithResourceMetrics(),
		QueryWithResourceAuthorize(),
	)
	Command = NewCommandChain(
		CommandLogging(),
		CommandMetrics(),
		CommandAuthorize(),
		UnitOfWork(),
		DispatchEvents(),
	)
)

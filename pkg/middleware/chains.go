package middleware

var (
	Query = NewQueryChain(
		QueryLogging(),
		QueryMetrics(),
		QueryAuthZ(),
	)
	QueryWithResource = NewQueryWithResourceChain(
		QueryWithResourceLogging(),
		QueryWithResourceMetrics(),
		QueryAuthZWithResource(),
	)
	Command = NewCommandChain(
		CommandLogging(),
		CommandMetrics(),
		CommandAuthZ(),
		UnitOfWork(),
		DispatchEvents(),
	)
)

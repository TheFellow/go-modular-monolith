package middleware

var (
	Query = NewChain(
		Logging(),
		Metrics(),
		Authorize(),
	)
	Command = NewChain(
		Logging(),
		Metrics(),
		TrackActivity(),
		UnitOfWork(),
		DispatchEvents(),
	)
)

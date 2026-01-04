package middleware

var (
	Query             = NewQueryChain(QueryAuthZ())
	QueryWithResource = NewQueryWithResourceChain(QueryAuthZWithResource())
	Command           = NewCommandChain(CommandAuthZ(), UnitOfWork())
)

package events

// OrdersAmended is the complete result of one selected-batch amendment command.
type OrdersAmended struct{ Changes []OrderAmended }

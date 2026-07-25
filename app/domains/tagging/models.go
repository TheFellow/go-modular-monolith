package tagging

type entityTagRow struct {
	ID         uint64
	EntityType string `bstore:"unique EntityType+EntityID+Key,index EntityType+EntityID"`
	EntityID   string
	Key        string
	Value      string
}

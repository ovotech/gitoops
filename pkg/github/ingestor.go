package github

// Interface for all ingestors
type Ingestor interface {
	FetchData()
	Sync()
}

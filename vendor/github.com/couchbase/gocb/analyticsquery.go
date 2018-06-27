package gocb

// AnalyticsQuery represents a pending Analytics query.
type AnalyticsQuery struct {
	options map[string]interface{}
}

// NewAnalyticsQuery creates a new N1qlQuery object from a query string.
func NewAnalyticsQuery(statement string) *AnalyticsQuery {
	nq := &AnalyticsQuery{
		options: make(map[string]interface{}),
	}
	nq.options["statement"] = statement
	return nq
}

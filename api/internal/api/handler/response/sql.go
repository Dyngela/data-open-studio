package response

type GuessQueryResponse struct {
	Query string `json:"query"`
}

type OptimizeQueryResponse struct {
	OptimizedQuery string `json:"optimizedQuery"`
	Explanation    string `json:"explanation"`
}

package request

type GuessQueryRequest struct {
	Prompt                  string   `json:"prompt" validate:"required"`
	ConnectionID            int      `json:"connectionId" validate:"required"`
	SchemaOptimizationNeeded bool    `json:"schemaOptimizationNeeded"`
	PreviousMessages        []string `json:"previousMessages"`
}

type OptimizeQueryRequest struct {
	Query        string `json:"query" validate:"required"`
	ConnectionID int    `json:"connectionId" validate:"required"`
}

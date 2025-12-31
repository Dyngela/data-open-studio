package response

type APIError struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type Page[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalPages int   `json:"totalPages"`
}

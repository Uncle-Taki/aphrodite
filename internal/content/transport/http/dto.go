package http

// ContentListResponse is the paginated envelope for content list endpoints.
type ContentListResponse struct {
	Items     []interface{} `json:"items"`
	Total     int           `json:"total" example:"42"`
	Limit     int           `json:"limit" example:"15"`
	Page      int           `json:"page" example:"1"`
	PageCount int           `json:"page_count" example:"3"`
}

// ErrorResponse is the generic error body for content endpoints.
type ErrorResponse struct {
	Error string `json:"error" example:"content not found"`
}

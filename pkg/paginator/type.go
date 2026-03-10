package paginator

// PaginateQuery contains pagination parameters for a request.
type PaginateQuery struct {
	Page  int   `json:"page" form:"page"`   // Page number (1-indexed)
	Limit int64 `json:"limit" form:"limit"` // Number of items per page
}

// Paginator contains pagination metadata for a query result.
type Paginator struct {
	Total       int64 `json:"total"`        // Total number of items across all pages
	Count       int64 `json:"count"`        // Number of items in current page
	PerPage     int64 `json:"per_page"`     // Number of items per page
	CurrentPage int   `json:"current_page"` // Current page number (1-indexed)
}

// PaginatorResponse is the response format for pagination metadata.
type PaginatorResponse struct {
	Total       int64 `json:"total"`        // Total number of items across all pages
	Count       int64 `json:"count"`        // Number of items in current page
	PerPage     int64 `json:"per_page"`     // Number of items per page
	CurrentPage int   `json:"current_page"` // Current page number (1-indexed)
	TotalPages  int   `json:"total_pages"`  // Total number of pages
	HasNext     bool  `json:"has_next"`     // Whether there is a next page
	HasPrev     bool  `json:"has_prev"`     // Whether there is a previous page
}

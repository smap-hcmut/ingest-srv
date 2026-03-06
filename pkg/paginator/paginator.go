package paginator

import "math"

// Adjust normalizes the pagination parameters to valid values.
// Sets defaults if values are invalid and enforces maximum limit.
func (p *PaginateQuery) Adjust() {
	if p.Page < 1 {
		p.Page = DefaultPage
	}

	if p.Limit < 1 {
		p.Limit = DefaultLimit
	} else if p.Limit > MaxLimit {
		p.Limit = MaxLimit
	}
}

// Offset calculates the database offset for the current page.
func (p *PaginateQuery) Offset() int64 {
	return int64((p.Page - 1)) * p.Limit
}

// TotalPages calculates the total number of pages based on total items and items per page.
func (p Paginator) TotalPages() int {
	if p.Total == 0 || p.PerPage == 0 {
		return 0
	}
	return int(math.Ceil(float64(p.Total) / float64(p.PerPage)))
}

// HasNextPage checks if there is a next page available.
func (p Paginator) HasNextPage() bool {
	return p.CurrentPage < p.TotalPages()
}

// HasPreviousPage checks if there is a previous page available.
func (p Paginator) HasPreviousPage() bool {
	return p.CurrentPage > 1
}

// ToResponse converts the paginator to a response format with additional calculated fields.
func (p Paginator) ToResponse() PaginatorResponse {
	return PaginatorResponse{
		Total:       p.Total,
		Count:       p.Count,
		PerPage:     p.PerPage,
		CurrentPage: p.CurrentPage,
		TotalPages:  p.TotalPages(),
		HasNext:     p.HasNextPage(),
		HasPrev:     p.HasPreviousPage(),
	}
}

// ToPaginator converts a response back to a paginator (e.g. for deserialization).
func (p PaginatorResponse) ToPaginator() Paginator {
	return Paginator{
		Total:       p.Total,
		Count:       p.Count,
		PerPage:     p.PerPage,
		CurrentPage: p.CurrentPage,
	}
}

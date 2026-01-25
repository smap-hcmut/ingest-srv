package paginator

// PaginateSlice applies pagination to a slice of any type.
// It returns a new slice containing only the items for the requested page.
func PaginateSlice[T any](slice []T, query PaginateQuery) ([]T, Paginator) {
	// Adjust pagination parameters if needed
	query.Adjust()

	// Calculate total items
	total := int64(len(slice))

	// Calculate start and end indices
	startIndex := query.Offset()
	endIndex := startIndex + query.Limit

	// Ensure endIndex doesn't exceed slice length
	if endIndex > total {
		endIndex = total
	}

	// Handle empty slice or out of range page
	if startIndex >= total {
		// Return empty slice with pagination info
		return []T{}, Paginator{
			Total:       total,
			Count:       0,
			PerPage:     query.Limit,
			CurrentPage: query.Page,
		}
	}

	// Extract the page items
	pageItems := slice[startIndex:endIndex]

	// Create pagination response
	paginationResponse := Paginator{
		Total:       total,
		Count:       int64(len(pageItems)),
		PerPage:     query.Limit,
		CurrentPage: query.Page,
	}

	return pageItems, paginationResponse
}

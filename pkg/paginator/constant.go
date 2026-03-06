package paginator

const (
	// DefaultPage is the default page number when invalid page is provided.
	DefaultPage = 1
	// DefaultLimit is the default number of items per page when invalid limit is provided.
	DefaultLimit = 15
	// MaxLimit is the maximum number of items per page to prevent excessive queries.
	MaxLimit = 100
)

package microservice

type ProjectStatus string

const (
	ProjectStatusDraft    ProjectStatus = "DRAFT"
	ProjectStatusActive   ProjectStatus = "ACTIVE"
	ProjectStatusPaused   ProjectStatus = "PAUSED"
	ProjectStatusArchived ProjectStatus = "ARCHIVED"
)

type ProjectDetail struct {
	ID     string
	Status ProjectStatus
}

package microservice

type ProjectStatus string

const (
	ProjectStatusPending  ProjectStatus = "PENDING"
	ProjectStatusActive   ProjectStatus = "ACTIVE"
	ProjectStatusPaused   ProjectStatus = "PAUSED"
	ProjectStatusArchived ProjectStatus = "ARCHIVED"
)

type ProjectDetail struct {
	ID             string
	Status         ProjectStatus
	DomainTypeCode string
}

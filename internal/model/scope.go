package model

const (
	ScopeTypeAccess = "access"
	SMAPAPI         = "ingest-srv"
)

const (
	RoleAdmin   = "ADMIN"
	RoleAnalyst = "ANALYST"
	RoleViewer  = "VIEWER"
)

type Scope struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"` // ADMIN, ANALYST, or VIEWER
	JTI      string `json:"jti"`
}

// IsAdmin checks if the scope has admin role
func (s Scope) IsAdmin() bool {
	return s.Role == RoleAdmin
}

// IsAnalyst checks if the scope has analyst role
func (s Scope) IsAnalyst() bool {
	return s.Role == RoleAnalyst
}

// IsViewer checks if the scope has viewer role
func (s Scope) IsViewer() bool {
	return s.Role == RoleViewer
}

package postgre

import (
	"database/sql"

	repo "ingest-srv/internal/datasource/repository"
	"ingest-srv/pkg/log"
)

type implRepository struct {
	db *sql.DB
	l  log.Logger
}

// New creates a new PostgreSQL data source repository.
func New(l log.Logger, db *sql.DB) repo.Repository {
	return &implRepository{db: db, l: l}
}

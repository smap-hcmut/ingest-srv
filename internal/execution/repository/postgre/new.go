package postgre

import (
	"database/sql"

	"ingest-srv/internal/execution/repository"
	"ingest-srv/pkg/log"
)

type implRepository struct {
	l  log.Logger
	db *sql.DB
}

// New creates a PostgreSQL-backed execution repository.
func New(l log.Logger, db *sql.DB) repository.Repository {
	return &implRepository{
		l:  l,
		db: db,
	}
}

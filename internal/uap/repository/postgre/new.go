package postgre

import (
	"database/sql"

	repo "ingest-srv/internal/uap/repository"
	"ingest-srv/pkg/log"
)

type implRepository struct {
	l  log.Logger
	db *sql.DB
}

// New creates a new UAP repository.
func New(l log.Logger, db *sql.DB) repo.Repository {
	return &implRepository{
		l:  l,
		db: db,
	}
}

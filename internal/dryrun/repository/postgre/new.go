package postgre

import (
	"database/sql"

	dryrunRepo "ingest-srv/internal/dryrun/repository"
	"ingest-srv/pkg/log"
)

type implRepository struct {
	db *sql.DB
	l  log.Logger
}

// New creates a PostgreSQL dryrun repository.
func New(l log.Logger, db *sql.DB) dryrunRepo.Repository {
	return &implRepository{db: db, l: l}
}

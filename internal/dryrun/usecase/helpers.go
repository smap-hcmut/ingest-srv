package usecase

import (
	"strings"

	"ingest-srv/internal/dryrun"
)

func validTriggerInput(input dryrun.TriggerInput) error {
	if strings.TrimSpace(input.SourceID) == "" {
		return dryrun.ErrSourceNotFound
	}
	if input.SampleLimit != nil && *input.SampleLimit <= 0 {
		return dryrun.ErrInvalidSampleLimit
	}
	return nil
}

func validGetLatestInput(input dryrun.GetLatestInput) error {
	if strings.TrimSpace(input.SourceID) == "" {
		return dryrun.ErrSourceNotFound
	}
	return nil
}

func validListHistoryInput(input dryrun.ListHistoryInput) error {
	if strings.TrimSpace(input.SourceID) == "" {
		return dryrun.ErrSourceNotFound
	}
	return nil
}

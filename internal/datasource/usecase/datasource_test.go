package usecase

import (
	"errors"
	"testing"

	"ingest-srv/internal/datasource"
)

func TestValidUpdateInputAcceptsPositiveCrawlInterval(t *testing.T) {
	interval := 17
	uc := &implUseCase{}

	err := uc.validUpdateInput(datasource.UpdateInput{
		ID:                   "datasource-id",
		CrawlIntervalMinutes: &interval,
	})
	if err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}
}

func TestValidUpdateInputRejectsNonPositiveCrawlInterval(t *testing.T) {
	uc := &implUseCase{}

	for _, interval := range []int{0, -1} {
		err := uc.validUpdateInput(datasource.UpdateInput{
			ID:                   "datasource-id",
			CrawlIntervalMinutes: &interval,
		})
		if !errors.Is(err, datasource.ErrInvalidCrawlInterval) {
			t.Fatalf("expected ErrInvalidCrawlInterval for %d, got %v", interval, err)
		}
	}
}

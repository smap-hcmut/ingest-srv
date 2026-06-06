package http

import "testing"

func TestUpdateReqToInputIncludesCrawlInterval(t *testing.T) {
	interval := 17
	req := updateReq{
		ID:                   "550e8400-e29b-41d4-a716-446655440000",
		Name:                 "TikTok crawler",
		CrawlIntervalMinutes: &interval,
	}

	input := req.toInput()
	if input.CrawlIntervalMinutes == nil {
		t.Fatal("expected crawl_interval_minutes to be forwarded")
	}
	if got := *input.CrawlIntervalMinutes; got != interval {
		t.Fatalf("unexpected crawl_interval_minutes: got %d want %d", got, interval)
	}
}

func TestUpdateReqValidateRejectsNonPositiveCrawlInterval(t *testing.T) {
	for _, interval := range []int{0, -1} {
		req := updateReq{CrawlIntervalMinutes: &interval}
		if err := req.validate(); err != errInvalidCrawlInterval {
			t.Fatalf("expected errInvalidCrawlInterval for %d, got %v", interval, err)
		}
	}
}

package model

import (
	"time"

	"ingest-srv/internal/sqlboiler"
)

// CrawlModeDefault stores system fallback interval by crawl mode.
// This corresponds to ingest.crawl_mode_defaults.
type CrawlModeDefault struct {
	Mode            CrawlMode `json:"mode"`                     // Mode hệ thống cần apply default interval.
	IntervalMinutes int       `json:"interval_minutes"`         // Giá trị fallback để derive interval cho source khi đổi mode.
	ModeMultiplier  string    `json:"mode_multiplier"`          // Hệ số nhân interval khi scheduler tính effective interval per-target.
	Description     string    `json:"description,omitempty"`    // Ghi chú để vận hành hiểu ý nghĩa business của default này.
	CreatedAt       time.Time `json:"created_at"`               // Mốc tạo cấu hình default.
	UpdatedAt       time.Time `json:"updated_at"`               // Mốc chỉnh sửa gần nhất của default config.
}

func NewCrawlModeDefaultFromDB(db *sqlboiler.CrawlModeDefault) *CrawlModeDefault {
	if db == nil {
		return nil
	}

	return &CrawlModeDefault{
		Mode:            CrawlMode(db.Mode),
		IntervalMinutes: db.IntervalMinutes,
		ModeMultiplier:  db.ModeMultiplier.String(),
		Description:     stringFromNull(db.Description),
		CreatedAt:       db.CreatedAt,
		UpdatedAt:       db.UpdatedAt,
	}
}

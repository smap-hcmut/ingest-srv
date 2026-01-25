package repository

import (
	"context"
	"time"

	"smap-collector/internal/models"
)

//go:generate mockery --name=StateRepository
type StateRepository interface {
	// InitState khởi tạo state mới trong Redis với TTL.
	// Sử dụng HSET để set multiple fields cùng lúc.
	InitState(ctx context.Context, key string, state models.ProjectState, ttl time.Duration) error

	// GetState lấy state từ Redis.
	// Trả về nil nếu key không tồn tại.
	GetState(ctx context.Context, key string) (*models.ProjectState, error)

	// SetField set một field trong hash.
	// Dùng cho status updates.
	SetField(ctx context.Context, key, field string, value any) error

	// SetFields set multiple fields trong hash.
	// Dùng cho batch updates.
	SetFields(ctx context.Context, key string, fields map[string]any) error

	// IncrementField tăng giá trị của field lên delta.
	// Dùng cho done/errors counters.
	IncrementField(ctx context.Context, key, field string, delta int64) (int64, error)

	// Exists kiểm tra key có tồn tại không.
	Exists(ctx context.Context, key string) (bool, error)

	// SetTTL set TTL cho key.
	SetTTL(ctx context.Context, key string, ttl time.Duration) error

	// Delete xóa key.
	Delete(ctx context.Context, key string) error

	// SetString set một string value.
	// Dùng cho user mapping.
	SetString(ctx context.Context, key, value string, ttl time.Duration) error

	// GetString lấy string value.
	// Dùng cho user mapping.
	GetString(ctx context.Context, key string) (string, error)
}

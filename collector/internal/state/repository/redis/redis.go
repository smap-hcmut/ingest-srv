package redis

import (
	"context"
	"strconv"
	"time"

	"smap-collector/internal/models"
	"smap-collector/internal/state"
)

// InitState khởi tạo state mới trong Redis với two-phase counters.
func (r *redisRepository) InitState(ctx context.Context, key string, s models.ProjectState, ttl time.Duration) error {
	// Set status
	if err := r.client.HSet(ctx, key, state.FieldStatus, string(s.Status)); err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.InitState: HSet status error: %v", err)
		return ErrHSetFailed
	}

	// Set crawl phase fields
	if err := r.client.HSet(ctx, key, state.FieldCrawlTotal, s.CrawlTotal); err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.InitState: HSet crawl_total error: %v", err)
		return ErrHSetFailed
	}
	if err := r.client.HSet(ctx, key, state.FieldCrawlDone, s.CrawlDone); err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.InitState: HSet crawl_done error: %v", err)
		return ErrHSetFailed
	}
	if err := r.client.HSet(ctx, key, state.FieldCrawlErrors, s.CrawlErrors); err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.InitState: HSet crawl_errors error: %v", err)
		return ErrHSetFailed
	}

	// Set analyze phase fields
	if err := r.client.HSet(ctx, key, state.FieldAnalyzeTotal, s.AnalyzeTotal); err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.InitState: HSet analyze_total error: %v", err)
		return ErrHSetFailed
	}
	if err := r.client.HSet(ctx, key, state.FieldAnalyzeDone, s.AnalyzeDone); err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.InitState: HSet analyze_done error: %v", err)
		return ErrHSetFailed
	}
	if err := r.client.HSet(ctx, key, state.FieldAnalyzeErrors, s.AnalyzeErrors); err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.InitState: HSet analyze_errors error: %v", err)
		return ErrHSetFailed
	}

	if err := r.client.Expire(ctx, key, int(ttl.Seconds())); err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.InitState: Expire error: %v", err)
		return ErrExpireFailed
	}

	return nil
}

// GetState lấy state từ Redis với two-phase counters.
func (r *redisRepository) GetState(ctx context.Context, key string) (*models.ProjectState, error) {
	data, err := r.client.HGetAll(ctx, key)
	if err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.GetState: HGetAll error: %v", err)
		return nil, ErrHGetAllFailed
	}

	if len(data) == 0 {
		return nil, nil
	}

	s := &models.ProjectState{
		Status: models.ProjectStatus(data[state.FieldStatus]),
	}

	// Parse crawl phase fields
	if crawlTotal, err := strconv.ParseInt(data[state.FieldCrawlTotal], 10, 64); err == nil {
		s.CrawlTotal = crawlTotal
	}
	if crawlDone, err := strconv.ParseInt(data[state.FieldCrawlDone], 10, 64); err == nil {
		s.CrawlDone = crawlDone
	}
	if crawlErrors, err := strconv.ParseInt(data[state.FieldCrawlErrors], 10, 64); err == nil {
		s.CrawlErrors = crawlErrors
	}

	// Parse analyze phase fields
	if analyzeTotal, err := strconv.ParseInt(data[state.FieldAnalyzeTotal], 10, 64); err == nil {
		s.AnalyzeTotal = analyzeTotal
	}
	if analyzeDone, err := strconv.ParseInt(data[state.FieldAnalyzeDone], 10, 64); err == nil {
		s.AnalyzeDone = analyzeDone
	}
	if analyzeErrors, err := strconv.ParseInt(data[state.FieldAnalyzeErrors], 10, 64); err == nil {
		s.AnalyzeErrors = analyzeErrors
	}

	return s, nil
}

// SetField set một field trong hash.
func (r *redisRepository) SetField(ctx context.Context, key, field string, value any) error {
	if err := r.client.HSet(ctx, key, field, value); err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.SetField: HSet error: %v", err)
		return ErrHSetFailed
	}
	return nil
}

// SetFields set multiple fields trong hash.
func (r *redisRepository) SetFields(ctx context.Context, key string, fields map[string]any) error {
	for field, value := range fields {
		if err := r.client.HSet(ctx, key, field, value); err != nil {
			r.l.Errorf(ctx, "internal.state.repository.redis.SetFields: HSet error: %v", err)
			return ErrHSetFailed
		}
	}
	return nil
}

// IncrementField tăng giá trị của field.
func (r *redisRepository) IncrementField(ctx context.Context, key, field string, delta int64) (int64, error) {
	result, err := r.client.HIncrBy(ctx, key, field, delta)
	if err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.IncrementField: HIncrBy error: %v", err)
		return 0, ErrHIncrByFailed
	}
	return result, nil
}

// Exists kiểm tra key có tồn tại không.
func (r *redisRepository) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := r.client.Exists(ctx, key)
	if err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.Exists: error: %v", err)
		return false, ErrExistsFailed
	}
	return exists, nil
}

// SetTTL set TTL cho key.
func (r *redisRepository) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	if err := r.client.Expire(ctx, key, int(ttl.Seconds())); err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.SetTTL: Expire error: %v", err)
		return ErrExpireFailed
	}
	return nil
}

// Delete xóa key.
func (r *redisRepository) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key); err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.Delete: Del error: %v", err)
		return ErrDelFailed
	}
	return nil
}

// SetString set một string value.
func (r *redisRepository) SetString(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, value, int(ttl.Seconds())); err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.SetString: Set error: %v", err)
		return ErrSetFailed
	}
	return nil
}

// GetString lấy string value.
func (r *redisRepository) GetString(ctx context.Context, key string) (string, error) {
	value, err := r.client.Get(ctx, key)
	if err != nil {
		r.l.Errorf(ctx, "internal.state.repository.redis.GetString: Get error: %v", err)
		return "", ErrGetFailed
	}
	return string(value), nil
}

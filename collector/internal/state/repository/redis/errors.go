package redis

import "errors"

var (
	ErrHSetFailed    = errors.New("redis HSet failed")
	ErrHGetAllFailed = errors.New("redis HGetAll failed")
	ErrHIncrByFailed = errors.New("redis HIncrBy failed")
	ErrExpireFailed  = errors.New("redis Expire failed")
	ErrExistsFailed  = errors.New("redis Exists failed")
	ErrDelFailed     = errors.New("redis Del failed")
	ErrSetFailed     = errors.New("redis Set failed")
	ErrGetFailed     = errors.New("redis Get failed")
)

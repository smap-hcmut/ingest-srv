package util

import "errors"

var (
	ErrInvalidEmail     = errors.New("invalid email")
	ErrInvalidPhone     = errors.New("invalid phone")
	ErrInvalidPhoneCode = errors.New("invalid phone code")
	ErrInvalidPassword  = errors.New("invalid password")
)

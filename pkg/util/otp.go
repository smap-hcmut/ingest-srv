package util

import (
	"math/rand"
	"strconv"
	"time"
)

const (
	length = 6
	expire = time.Hour * 24
)

func GenerateOTP() (string, time.Time) {
	otp := ""
	for range length {
		otp += strconv.Itoa(rand.Intn(10))
	}
	return otp, time.Now().Add(expire)
}

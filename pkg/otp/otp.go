package otp

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	OTP_LENGTH = 6
	ONE_HOUR   = 3600
)

func GenerateOTP(current time.Time) (string, time.Time) {
	otp := fmt.Sprintf("%06d", rand.Intn(1000000))
	return otp, current.Add(time.Second * ONE_HOUR)
}

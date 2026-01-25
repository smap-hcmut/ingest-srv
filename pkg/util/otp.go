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

func GenerateOTP() (string, error) {
	otp := ""
	for i := 0; i < length; i++ {
		otp += strconv.Itoa(rand.Intn(10))
	}
	return otp, nil
}

func GenerateOTPExpireAt() (time.Time, error) {
	return time.Now().Add(expire), nil
}

func GenerateOTPExpireAtStr() (string, error) {
	otpExpireAt, err := GenerateOTPExpireAt()
	if err != nil {
		return "", err
	}
	return DateTimeToStr(otpExpireAt), nil
}

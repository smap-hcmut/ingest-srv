package util

import (
	"math/rand"
	"strings"
	"unicode"
)

// random string
func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func AlphabetString(s string) bool {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}

// remove special character
func RemoveSpecialCharacter(s string) string {
	s = strings.TrimSpace(s)
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func ConvertToInterface(strs []string) []interface{} {
	interfaces := make([]interface{}, len(strs))
	for i, v := range strs {
		interfaces[i] = v
	}
	return interfaces
}

func SliceToArray(slice []string) []interface{} {
	interfaces := make([]interface{}, len(slice))
	for i, v := range slice {
		interfaces[i] = v
	}
	return interfaces
}

package util

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	DateTimeFormat = "2006-01-02 15:04:05"
	DateFormat     = "2006-01-02"
)

func MillisecondsToTime(ms int64) time.Time {
	seconds := ms / 1000
	nanoseconds := (ms % 1000) * 1000000
	return time.Unix(seconds, nanoseconds)
}

func MicrosecondsToTime(ms int64) time.Time {
	seconds := ms / 1000000
	nanoseconds := (ms % 1000000) * 1000
	return time.Unix(seconds, nanoseconds)
}

func ConvertTimeZone(t time.Time, fromLoc *time.Location, toLoc *time.Location) (time.Time, error) {
	tInFromLoc := t.In(fromLoc)
	tInToLoc := tInFromLoc.In(toLoc)
	return tInToLoc, nil
}

func GetTimeZone(t time.Time) *time.Location {
	return t.Location()
}

func StrToDateTime(str string) (time.Time, error) {
	t, err := time.ParseInLocation(DateTimeFormat, str, GetDefaultTimezone())
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

func StrToDate(str string) (time.Time, error) {
	t, err := time.ParseInLocation(DateFormat, str, GetDefaultTimezone())
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

func DateToStr(dt time.Time) string {
	return dt.Format(DateFormat)
}

func DateTimeToStr(dt time.Time) string {
	return dt.Format(DateTimeFormat)
}

func Now() time.Time {
	return time.Now().In(GetDefaultTimezone())
}

func GetDefaultTimezone() *time.Location {
	localTimeZone, _ := time.LoadLocation("Local")
	return localTimeZone
}

func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, GetDefaultTimezone())
}

func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, GetDefaultTimezone())
}

func SetHour(t time.Time, hour int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), hour, t.Minute(), t.Second(), 0, GetDefaultTimezone())
}

func SetMinute(t time.Time, minute int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, t.Second(), 0, GetDefaultTimezone())
}

func DateTimeToInt(dt time.Time) int {
	return int(dt.Unix())
}

func FormatTime(t time.Time, format string) string {
	goFormat := convertPHPToGoTimeFormat(format)
	return t.Format(goFormat)
}

func convertPHPToGoTimeFormat(format string) string {
	replacements := map[string]string{
		"Y": "2006",
		"m": "01",
		"d": "02",
		"H": "15",
		"i": "04",
		"s": "05",
	}

	for php, goTime := range replacements {
		format = strings.Replace(format, php, goTime, -1)
	}

	return format
}

func BuildDateTimeStrFromDateStrAndHourMinute(date string, hour int, minute int) (string, error) {
	dateTime, err := StrToDateTime(fmt.Sprintf("%s 00:00:00", date))
	if err != nil {
		return "", err
	}
	if hour > 23 {
		dateTime = dateTime.AddDate(0, 0, 1)
		hour = hour - 24
	}

	if hour < 0 {
		dateTime = dateTime.AddDate(0, 0, -1)
		hour = hour + 24
	}

	dateTime = SetHour(dateTime, hour)
	dateTime = SetMinute(dateTime, minute)

	return DateTimeToStr(dateTime), nil
}

func AddMonths(date time.Time, months int) time.Time {
	year, month, day := date.Date()
	month = time.Month(int(month) + months)
	newDate := time.Date(year, month, 1, 0, 0, 0, 0, GetDefaultTimezone())
	newDate = newDate.AddDate(0, 1, -1)
	if day > newDate.Day() {
		day = newDate.Day()
	}
	return time.Date(newDate.Year(), newDate.Month(), day, date.Hour(), date.Minute(), date.Second(), date.Nanosecond(), date.Location())
}

func GetPeriodAndYear(t time.Time) (int32, int32) {
	p := int32(math.Ceil(float64(t.Month()) / 3))
	return p, int32(t.Year())
}

func GetPeriodAndYearRange(start_time time.Time, end_time time.Time) []time.Time {
	var result []time.Time
	result = append(result, start_time)
	for {
		start_time = start_time.AddDate(0, 3, 0)
		result = append(result, start_time)
		if !start_time.Before(end_time) {
			break
		}
	}
	pS, yS := GetPeriodAndYear(start_time)
	pE, yE := GetPeriodAndYear(end_time)
	if pS != pE || yS != yE {
		result = result[:len(result)-1]
	}
	return result
}

package time

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"
)

// CountDigitsInNumber returns the number of digits in an integer.
// Handles zero and negative numbers by absolute value conversion.
func CountDigitsInNumber(number int64) int {
	if number == 0 {
		return 1
	}

	count := 0
	number = int64(math.Abs(float64(number)))

	for number != 0 {
		number /= 10
		count++
	}
	return count
}

// UnixToTime converts Unix timestamp to time.Time with automatic scaling.
// Supports seconds, milliseconds, microseconds and nanoseconds.
func UnixToTime(unixTime int64) time.Time {
	digits := CountDigitsInNumber(unixTime)

	switch {
	case digits <= 10:
		return time.Unix(unixTime, 0)
	case digits <= 13:
		return time.UnixMilli(unixTime)
	case digits <= 16:
		return time.UnixMicro(unixTime)
	default:
		return time.Unix(0, unixTime)
	}
}

// SecondsSince returns seconds between now and given timestamp (past-aware).
func SecondsSince(unixTime int64) int {
	t := UnixToTime(unixTime)
	diff := time.Since(t)
	return int(math.Round(diff.Seconds()))
}

// SecondsUntil returns seconds between given timestamp and now (future-aware).
func SecondsUntil(unixTime int64) int {
	t := UnixToTime(unixTime)
	diff := t.Sub(time.Now())
	return int(math.Round(diff.Seconds()))
}

// TimeTrack measures execution time and logs with customizable level.
// Usage: defer TimeTrack(time.Now(), "operation", "DEBUG")
func TimeTrack(start time.Time, name string, level ...string) {
	elapsed := time.Since(start)
	logLevel := "INFO"
	if len(level) > 0 {
		logLevel = level[0]
	}

	log.Printf("[%s] %s took %s", logLevel, name, elapsed)
}

// FormatDuration converts duration to human-readable string.
// Supports nanoseconds to hours with appropriate units.
func FormatDuration(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return strconv.FormatInt(d.Nanoseconds(), 10) + "ns"
	case d < time.Millisecond:
		return strconv.FormatInt(d.Microseconds(), 10) + "µs"
	case d < time.Second:
		return strconv.FormatInt(d.Milliseconds(), 10) + "ms"
	case d < time.Minute:
		return strconv.FormatFloat(d.Seconds(), 'f', 2, 64) + "s"
	default:
		return d.Round(time.Second).String()
	}
}

// UnixMilli returns current timestamp in milliseconds (Go 1.17+ compatible).
func UnixMilli() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// ConvertToTimezone converts a time.Time to a specified timezone.
// Example: ConvertToTimezone(UTC时间, "Asia/Tokyo") -> Время в Токио.
func ConvertToTimezone(t time.Time, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("timeutil: invalid timezone %q", timezone)
	}
	return t.In(loc), nil
}

// NowInTimezone returns the current time in the specified timezone.
// Example: NowInTimezone("Europe/Moscow") -> Текущее время в Москве.
func NowInTimezone(timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("timeutil: invalid timezone %q", timezone)
	}
	return time.Now().In(loc), nil
}

// ParseInTimezone parses a time string in the specified timezone and layout.
// Example: ParseInTimezone("2023-10-01 12:00", "2006-01-02 15:04", "America/Chicago")
func ParseInTimezone(timeStr, layout, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("timeutil: invalid timezone %q", timezone)
	}
	return time.ParseInLocation(layout, timeStr, loc)
}

// LocationByOffset creates a fixed timezone by UTC offset (e.g., 3 for UTC+3).
// Example: LocationByOffset(3) -> UTC+3.
func LocationByOffset(offsetHours int) (*time.Location, error) {
	if offsetHours < -12 || offsetHours > 14 {
		return nil, fmt.Errorf("timeutil: offset %d is out of range [-12, 14]", offsetHours)
	}
	offset := time.Duration(offsetHours) * time.Hour
	return time.FixedZone(fmt.Sprintf("UTC%+d", offsetHours), int(offset.Seconds())), nil
}

// FormatWithTimezone returns a formatted string with timezone info.
// Example: FormatWithTimezone(time.Now(), time.RFC3339, "Australia/Sydney")
func FormatWithTimezone(t time.Time, layout, timezone string) (string, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return "", fmt.Errorf("timeutil: invalid timezone %q", timezone)
	}
	return t.In(loc).Format(layout), nil
}

package core

import (
	"strconv"
	"strings"
	"time"
)

func ParseMinute(value string) (int, error) {
	const width, partsCount, hourLimit, minuteLimit = 5, 2, 24, 60
	const delimiter, zero, hourIndex, minuteIndex = ":", 0, 0, 1
	parts := strings.Split(value, delimiter)
	if len(value) != width || len(parts) != partsCount {
		return zero, ErrInvalid
	}
	for _, part := range parts {
		for _, char := range part {
			const low, high = '0', '9'
			if char < low || char > high {
				return zero, ErrInvalid
			}
		}
	}
	hour, errHour := strconv.Atoi(parts[hourIndex])
	minute, errMinute := strconv.Atoi(parts[minuteIndex])
	if errHour != nil || errMinute != nil || hour >= hourLimit || minute >= minuteLimit {
		return zero, ErrInvalid
	}
	return hour*minuteLimit + minute, nil
}

func ParseWindow(input InputWindow) (Window, error) {
	start, err := ParseMinute(input.Start)
	if err != nil {
		return Window{}, err
	}
	end, err := ParseMinute(input.End)
	if err != nil {
		return Window{}, err
	}
	window := Window{Start: start, End: end}
	return window, ValidateWindows([]Window{window})
}

func (w Window) Contains(minute int) bool {
	if w.Start < w.End {
		return minute >= w.Start && minute < w.End
	}
	return minute >= w.Start || minute < w.End
}

func ValidateWindows(windows []Window) error {
	const dayMinutes, zero = 1440, 0
	var occupied [dayMinutes]bool
	for _, w := range windows {
		if w.Start < zero || w.Start >= dayMinutes || w.End < zero || w.End >= dayMinutes || w.Start == w.End {
			return ErrInvalid
		}
		for minute := zero; minute < dayMinutes; minute++ {
			if !w.Contains(minute) {
				continue
			}
			if occupied[minute] {
				return ErrOverlap
			}
			occupied[minute] = true
		}
	}
	return nil
}

func Duration(value string) (time.Duration, error) {
	const day = 24 * time.Hour
	const halfDay, one, three, seven, fourteen, thirty = "12h", "1d", "3d", "7d", "14d", "30d"
	const halfHours, threeDays, sevenDays, fortnightDays, monthDays, zero = 12, 3, 7, 14, 30, 0
	switch value {
	case halfDay:
		return halfHours * time.Hour, nil
	case one:
		return day, nil
	case three:
		return threeDays * day, nil
	case seven:
		return sevenDays * day, nil
	case fourteen:
		return fortnightDays * day, nil
	case thirty:
		return monthDays * day, nil
	default:
		return zero, ErrInvalid
	}
}

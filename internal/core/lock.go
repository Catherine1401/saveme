package core

import "time"

func (l *Lock) Active(now time.Time) bool {
	const minutesPerHour = 60
	if l == nil {
		return false
	}
	if l.Kind == Range {
		return now.Unix() >= l.StartAt && now.Unix() < l.EndAt
	}
	zone, err := time.LoadLocation(l.Zone)
	if err != nil {
		return true
	}
	local := now.In(zone)
	minute := local.Hour()*minutesPerHour + local.Minute()
	for _, window := range l.Windows {
		if window.Contains(minute) {
			return true
		}
	}
	return false
}

func (l *Lock) Transition(now time.Time) int64 {
	const zero, dayMinutes, horizonMinutes = 0, 1440, 8 * 1440
	if l == nil {
		return zero
	}
	if l.Kind == Range {
		if now.Unix() < l.StartAt {
			return l.StartAt
		}
		if now.Unix() < l.EndAt {
			return l.EndAt
		}
		return zero
	}
	var covered [dayMinutes]bool
	for _, window := range l.Windows {
		for minute := zero; minute < dayMinutes; minute++ {
			covered[minute] = covered[minute] || window.Contains(minute)
		}
	}
	all := true
	for _, enabled := range covered {
		all = all && enabled
	}
	if all {
		return zero
	}
	active := l.Active(now)
	cursor := now.Truncate(time.Minute).Add(time.Minute)
	for checked := zero; checked < horizonMinutes; checked++ {
		if l.Active(cursor) != active {
			return cursor.Unix()
		}
		cursor = cursor.Add(time.Minute)
	}
	return zero
}

func (s State) Validate() error {
	const zero = 0
	if s.Version != StateVersion || s.MaxSeen < zero {
		return ErrInvalid
	}
	for index, entry := range s.Entries {
		if ValidateDomain(entry.Domain) != nil {
			return ErrInvalid
		}
		for _, previous := range s.Entries[:index] {
			if previous.Domain == entry.Domain {
				return ErrInvalid
			}
		}
		if entry.Lock == nil {
			continue
		}
		if err := entry.Lock.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (l *Lock) Validate() error {
	const zero, maxRangeSeconds = 0, 30 * 24 * 60 * 60
	if _, err := time.LoadLocation(l.Zone); err != nil {
		return ErrInvalid
	}
	switch l.Kind {
	case Daily:
		if len(l.Windows) == zero || l.StartAt != zero || l.EndAt != zero {
			return ErrInvalid
		}
		return ValidateWindows(l.Windows)
	case Range:
		span := l.EndAt - l.StartAt
		if l.StartAt < zero || span <= zero || span > maxRangeSeconds || len(l.Windows) != zero {
			return ErrInvalid
		}
		return nil
	default:
		return ErrInvalid
	}
}

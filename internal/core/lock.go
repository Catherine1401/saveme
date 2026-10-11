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
	const version, zero, minimumDomains, maxRangeSeconds = 1, 0, 1, 30 * 24 * 60 * 60
	if s.Version != version || s.MaxSeen < zero {
		return ErrInvalid
	}
	for index, domain := range s.Domains {
		if ValidateDomain(domain) != nil {
			return ErrInvalid
		}
		for _, previous := range s.Domains[:index] {
			if previous == domain {
				return ErrInvalid
			}
		}
	}
	if s.Lock == nil {
		return nil
	}
	if len(s.Domains) < minimumDomains {
		return ErrInvalid
	}
	if _, err := time.LoadLocation(s.Lock.Zone); err != nil {
		return ErrInvalid
	}
	switch s.Lock.Kind {
	case Daily:
		if len(s.Lock.Windows) == zero || s.Lock.StartAt != zero || s.Lock.EndAt != zero {
			return ErrInvalid
		}
		return ValidateWindows(s.Lock.Windows)
	case Range:
		span := s.Lock.EndAt - s.Lock.StartAt
		if s.Lock.StartAt < zero || span <= zero || span > maxRangeSeconds || len(s.Lock.Windows) != zero {
			return ErrInvalid
		}
		return nil
	default:
		return ErrInvalid
	}
}

package config

import (
	"strconv"
	"strings"
)

// statusNumber returns the integer that follows the first colon on a w32tm
// status line. Labels are localized on some Windows installations, so callers
// rely on the fixed line order instead of English text.
func statusNumber(line string) (int, bool) {
	_, value, found := strings.Cut(line, ":")
	if !found {
		return 0, false
	}
	value = strings.TrimSpace(value)
	end := 0
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	number, err := strconv.Atoi(value[:end])
	return number, err == nil
}

// trustedWindowsStatus reports whether w32tm's status output shows a completed
// network synchronization. A configured source alone is not enough: w32time
// keeps reporting it while the clock is unsynchronized (leap indicator 3,
// stratum 0). Unknown or malformed output is treated as untrusted so stored
// locks stay closed.
func trustedWindowsStatus(text string) bool {
	const local, free = "Local CMOS Clock", "Free-running System Clock"
	if strings.Contains(text, local) || strings.Contains(text, free) {
		return false
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) < 2 {
		return false
	}
	leap, ok := statusNumber(lines[0])
	if !ok || leap > 2 {
		return false
	}
	stratum, ok := statusNumber(lines[1])
	return ok && stratum >= 1 && stratum <= 15
}

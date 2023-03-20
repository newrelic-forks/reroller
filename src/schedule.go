package reroller

import (
	"time"
)

type Schedule struct {
	After  time.Time
	Before time.Time
}

func (s *Schedule) ShouldRunNow() bool {
	return s.ShouldRun(time.Now())
}

func (s *Schedule) ShouldRun(target time.Time) bool {
	if s.After == s.Before {
		return true
	}

	return target.After(projectTime(s.After, target)) && target.Before(projectTime(s.Before, target))
}

// projectTime is a helper that returns a time with the date and location from `date` and the clock of `t`
func projectTime(t time.Time, date time.Time) time.Time {
	y, m, d := date.Date()
	h, mi, s := t.Clock()

	return time.Date(y, m, d, h, mi, s, t.Nanosecond(), date.Location())
}

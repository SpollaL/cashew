package domain

import (
	"fmt"
	"time"
)

type Granularity int

const (
	AllTime Granularity = iota
	Yearly
	Monthly
	Weekly
	Daily
)

func (g Granularity) String() string {
	switch g {
	case AllTime:
		return "All Time"
	case Yearly:
		return "Yearly"
	case Monthly:
		return "Monthly"
	case Weekly:
		return "Weekly"
	case Daily:
		return "Daily"
	default:
		return "Unknown"
	}
}

type Period struct {
	Start time.Time
	End   time.Time
	Label string
}

// Bucket returns the Period that t belongs to for granularity g.
// All transactions in the same period share the same Label.
func Bucket(t time.Time, g Granularity) Period {
	switch g {
	case Daily:
		d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		return Period{Start: d, End: d.AddDate(0, 0, 1), Label: d.Format("2006-01-02")}
	case Weekly:
		// ISO week: Monday as first day
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7
		} // Sunday = 7
		monday := t.AddDate(0, 0, -(weekday - 1))
		monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, t.Location())
		year, week := t.ISOWeek()
		return Period{
			Start: monday,
			End:   monday.AddDate(0, 0, 7),
			Label: fmt.Sprintf("%d-W%02d", year, week),
		}
	case Monthly:
		m := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
		return Period{Start: m, End: m.AddDate(0, 1, 0), Label: m.Format("2006-01")}
	case Yearly:
		y := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
		return Period{Start: y, End: y.AddDate(1, 0, 0), Label: fmt.Sprintf("%d", t.Year())}
	default: // AllTime
		return Period{Label: "All Time"}
	}
}

package domain_test

import (
	"github.com/spolla-l/cashew/internal/domain"
	"testing"
	"time"
)

func TestBucket(t *testing.T) {
	date := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC) // Sunday

	tests := []struct {
		granularity domain.Granularity
		wantLabel   string
	}{
		{domain.Daily, "2026-03-15"},
		{domain.Monthly, "2026-03"},
		{domain.Yearly, "2026"},
		{domain.AllTime, "All Time"},
	}

	for _, tt := range tests {
		t.Run(tt.wantLabel, func(t *testing.T) {
			p := domain.Bucket(date, tt.granularity)
			if p.Label != tt.wantLabel {
				t.Errorf("got label %q, want %q", p.Label, tt.wantLabel)
			}
		})
	}
}

func TestBucketWeekly(t *testing.T) {
	// 2026-03-15 is a Sunday — ISO week 11 of 2026.
	sunday := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	// 2026-03-09 is the Monday that starts ISO week 11.
	monday := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)

	ps := domain.Bucket(sunday, domain.Weekly)
	pm := domain.Bucket(monday, domain.Weekly)

	if ps.Label != pm.Label {
		t.Errorf("Sunday and Monday of same ISO week got different labels: %q vs %q", ps.Label, pm.Label)
	}
}

func TestBucketPeriodBounds(t *testing.T) {
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	p := domain.Bucket(date, domain.Monthly)

	wantStart := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	if !p.Start.Equal(wantStart) {
		t.Errorf("Start: got %v, want %v", p.Start, wantStart)
	}
	if !p.End.Equal(wantEnd) {
		t.Errorf("End: got %v, want %v", p.End, wantEnd)
	}
}

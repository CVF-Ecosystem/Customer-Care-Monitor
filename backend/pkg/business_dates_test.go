package pkg

import (
	"errors"
	"testing"
	"time"
)

// CCMAI-RUNTIME-027: the shared Vietnam business-date parser (UTC+7, inclusive dates, exclusive
// next-midnight upper bound, typed instants).

func TestParseBusinessRangeBoundaries(t *testing.T) {
	r, err := ParseBusinessRange("2026-10-02", "2026-10-02")
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2026, 10, 1, 17, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 10, 2, 17, 0, 0, 0, time.UTC)
	if r.From == nil || !r.From.Equal(wantFrom) || r.ToExclusive == nil || !r.ToExclusive.Equal(wantTo) || !r.Set() {
		t.Fatalf("got %+v", r)
	}
	// 23:59:59 VN is inside, 00:00:00 of the next VN day is outside.
	inside := time.Date(2026, 10, 2, 16, 59, 59, 0, time.UTC)
	outside := time.Date(2026, 10, 2, 17, 0, 0, 0, time.UTC)
	if inside.Before(*r.From) || !inside.Before(*r.ToExclusive) || outside.Before(*r.ToExclusive) {
		t.Fatalf("edge instants misclassified: %+v", r)
	}
}

func TestParseBusinessRangeOpenEndsAndEmpty(t *testing.T) {
	r, err := ParseBusinessRange("", "")
	if err != nil || r.Set() {
		t.Fatalf("empty: %+v %v", r, err)
	}
	r, err = ParseBusinessRange("2026-10-02", "")
	if err != nil || r.From == nil || r.ToExclusive != nil {
		t.Fatalf("from only: %+v %v", r, err)
	}
	r, err = ParseBusinessRange("", "2026-10-02")
	if err != nil || r.From != nil || r.ToExclusive == nil {
		t.Fatalf("to only: %+v %v", r, err)
	}
}

func TestParseBusinessRangeRejectsInvalid(t *testing.T) {
	for name, c := range map[string][2]string{
		"reversed":         {"2026-10-03", "2026-10-02"},
		"bad month":        {"2026-13-01", ""},
		"bad day":          {"", "2026-02-30"},
		"datetime":         {"2026-10-01T00:00:00Z", ""},
		"slashes":          {"01/10/2026", ""},
		"spaces":           {" 2026-10-01", ""},
		"before the floor": {"0000-01-01", ""},
		"after the cap":    {"", "9999-12-31"},
	} {
		if _, err := ParseBusinessRange(c[0], c[1]); !errors.Is(err, ErrInvalidDateRange) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

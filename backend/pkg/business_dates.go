package pkg

import (
	"errors"
	"time"
)

// CCMAI-RUNTIME-026 (F04) / CCMAI-RUNTIME-027 (F05): the Vietnam business-day date contract,
// shared by the report handlers and the analyzer so there is one parser. Date controls are
// date-only "YYYY-MM-DD" strings in Asia/Ho_Chi_Minh (UTC+07:00), independent of the browser, the
// server TZ, the UI language and the (unactivated) tenant timezone setting. `to` is an inclusive
// calendar date publicly and the next VN midnight, exclusively, internally. Bounds are typed
// instants, so the MySQL driver's own location stays the authority for how stored wall times are
// read and no hand-formatted UTC strings meet Local-storage timestamps.

// BusinessDateLayout is the only accepted date layout.
const BusinessDateLayout = "2006-01-02"

// ErrInvalidDateRange is the only error text a client sees for a bad date control.
var ErrInvalidDateRange = errors.New("invalid_date_range")

// DATETIME holds 1000-01-01 .. 9999-12-31; both bounds must be representable.
var (
	BusinessDateMin = time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC)
	BusinessDateMax = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
)

// BusinessRange is a half-open interval [From, ToExclusive); a nil side is open.
type BusinessRange struct {
	From        *time.Time
	ToExclusive *time.Time
}

// Set reports whether either side is bounded.
func (r BusinessRange) Set() bool { return r.From != nil || r.ToExclusive != nil }

// parseBusinessDate parses one strict calendar date. A missing value is not an error (ok=false).
func parseBusinessDate(s string) (day time.Time, ok bool, err error) {
	if s == "" {
		return time.Time{}, false, nil
	}
	t, perr := time.ParseInLocation(BusinessDateLayout, s, VNLocation)
	if perr != nil || t.Format(BusinessDateLayout) != s {
		return time.Time{}, false, ErrInvalidDateRange
	}
	utcDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	if utcDay.Before(BusinessDateMin) || utcDay.After(BusinessDateMax) {
		return time.Time{}, false, ErrInvalidDateRange
	}
	return t, true, nil
}

// ParseBusinessRange converts the public inclusive date pair to [fromStart, toExclusive).
// Empty strings leave that side open. Malformed, impossible, unrepresentable or reversed input
// is ErrInvalidDateRange; nothing falls back silently.
func ParseBusinessRange(fromStr, toStr string) (BusinessRange, error) {
	var r BusinessRange
	from, hasFrom, err := parseBusinessDate(fromStr)
	if err != nil {
		return BusinessRange{}, err
	}
	to, hasTo, err := parseBusinessDate(toStr)
	if err != nil {
		return BusinessRange{}, err
	}
	if hasFrom {
		// Calendar year 1000 is not enough: its VN midnight can serialize
		// as year 0999 on the supported UTC connection. Validate the actual
		// instant against the common UTC/VN storage range before any query.
		if from.UTC().Before(BusinessDateMin) {
			return BusinessRange{}, ErrInvalidDateRange
		}
		r.From = &from
	}
	if hasTo {
		next := to.AddDate(0, 0, 1)
		// Only a supplied inclusive `to` needs the next midnight; a
		// from-only final calendar day remains representable and open.
		if next.Year() > BusinessDateMax.Year() {
			return BusinessRange{}, ErrInvalidDateRange
		}
		r.ToExclusive = &next
	}
	// to < from (as calendar dates) means the exclusive bound is not after the start; an equal
	// date pair is a valid single day (To exclusive = From + 1 day).
	if hasFrom && hasTo && !r.ToExclusive.After(*r.From) {
		return BusinessRange{}, ErrInvalidDateRange
	}
	return r, nil
}

package handlers

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-026 (F04): Vietnam business-day read contract. Date controls are date-only
// "YYYY-MM-DD" strings in Asia/Ho_Chi_Minh (UTC+07:00), independent of the browser, the server
// TZ, the UI language and the (unactivated) tenant timezone setting. `to` is an inclusive
// calendar date publicly and the next VN midnight, exclusively, internally. Range parameters are
// typed instants, so the MySQL driver's own location stays the authority for how stored wall
// times are read and no hand-formatted UTC strings meet Local-storage timestamps.

const businessDateLayout = "2006-01-02"

// errInvalidDateRange is the only error text a client sees for a bad date control.
var errInvalidDateRange = errors.New("invalid_date_range")

// DATETIME holds 1000-01-01 .. 9999-12-31; both bounds must be representable.
var (
	businessDateMin = time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC)
	businessDateMax = time.Date(9999, 12, 30, 0, 0, 0, 0, time.UTC) // inclusive `to`; +1 day stays <= 9999-12-31
)

// businessClock is the single request clock; tests replace it through a private seam.
var businessClock = time.Now

// businessLocation returns the fixed business timezone.
func businessLocation() *time.Location { return pkg.VNLocation }

// businessRange is a half-open interval [From, ToExclusive); a nil side is open.
type businessRange struct {
	From        *time.Time
	ToExclusive *time.Time
}

// parseBusinessDate parses one strict calendar date. A missing value is not an error (ok=false).
func parseBusinessDate(s string) (day time.Time, ok bool, err error) {
	if s == "" {
		return time.Time{}, false, nil
	}
	t, perr := time.ParseInLocation(businessDateLayout, s, businessLocation())
	if perr != nil || t.Format(businessDateLayout) != s {
		return time.Time{}, false, errInvalidDateRange
	}
	utcDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	if utcDay.Before(businessDateMin) || utcDay.After(businessDateMax) {
		return time.Time{}, false, errInvalidDateRange
	}
	return t, true, nil
}

// parseBusinessRange converts the public inclusive date pair to [fromStart, toExclusive).
// Empty strings leave that side open. Malformed, impossible, unrepresentable or reversed input
// is errInvalidDateRange; nothing falls back silently.
func parseBusinessRange(fromStr, toStr string) (businessRange, error) {
	var r businessRange
	from, hasFrom, err := parseBusinessDate(fromStr)
	if err != nil {
		return businessRange{}, err
	}
	to, hasTo, err := parseBusinessDate(toStr)
	if err != nil {
		return businessRange{}, err
	}
	if hasFrom {
		r.From = &from
	}
	if hasTo {
		next := to.AddDate(0, 0, 1)
		r.ToExclusive = &next
	}
	// to < from (as calendar dates) means the exclusive bound is not after the start; an equal
	// date pair is a valid single day (To exclusive = From + 1 day).
	if hasFrom && hasTo && !r.ToExclusive.After(*r.From) {
		return businessRange{}, errInvalidDateRange
	}
	return r, nil
}

// requestBusinessRange reads the `from`/`to` query pair and answers 400 `invalid_date_range`
// (and returns ok=false) before any query or file output.
func requestBusinessRange(c *gin.Context) (businessRange, bool) {
	r, err := parseBusinessRange(c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid_date_range"})
		return businessRange{}, false
	}
	return r, true
}

// where applies `col >= From AND col < ToExclusive` for the sides that are set.
func (r businessRange) where(q *gorm.DB, col string) *gorm.DB {
	if r.From != nil {
		q = q.Where(col+" >= ?", *r.From)
	}
	if r.ToExclusive != nil {
		q = q.Where(col+" < ?", *r.ToExclusive)
	}
	return q
}

// businessDayStart is the VN midnight that starts the calendar day containing t.
func businessDayStart(t time.Time) time.Time {
	v := t.In(businessLocation())
	return time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, businessLocation())
}

// businessDayKey formats an instant as its VN calendar date.
func businessDayKey(t time.Time) string {
	return t.In(businessLocation()).Format(businessDateLayout)
}

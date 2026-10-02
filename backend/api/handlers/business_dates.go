package handlers

import (
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

const businessDateLayout = pkg.BusinessDateLayout

// errInvalidDateRange is the only error text a client sees for a bad date control (CCMAI-RUNTIME-027:
// the parser itself now lives in pkg so the analyzer shares it).
var errInvalidDateRange = pkg.ErrInvalidDateRange

// DATETIME limits, shared with the pkg parser (used by the report tests).
var (
	businessDateMin = pkg.BusinessDateMin
	businessDateMax = pkg.BusinessDateMax
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

// parseBusinessRange converts the public inclusive date pair to [fromStart, toExclusive) through
// the shared pkg parser (same semantics, same errors).
func parseBusinessRange(fromStr, toStr string) (businessRange, error) {
	r, err := pkg.ParseBusinessRange(fromStr, toStr)
	if err != nil {
		return businessRange{}, err
	}
	return businessRange{From: r.From, ToExclusive: r.ToExclusive}, nil
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

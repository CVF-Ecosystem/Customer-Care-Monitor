package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-026 (F04): shared fixtures and parser tests for the Vietnam business-day read
// contract. Handlers run against a disposable MySQL on two real storage encodings: a connection
// whose driver location is UTC and one whose location is Asia/Ho_Chi_Minh. Fixtures are written
// through the same driver with typed instants, so the stored wall time differs between the two
// (asserted from the raw column text) while every handler answer must be identical.

// The four named boundary instants of the Vietnam day 2026-10-02 (UTC+07:00).
var (
	bizBefore = time.Date(2026, 10, 1, 16, 59, 59, 999_000_000, time.UTC) // 2026-10-01 23:59:59.999 VN: previous day
	bizStart  = time.Date(2026, 10, 1, 17, 0, 0, 0, time.UTC)             // 2026-10-02 00:00:00.000 VN: first instant
	bizLast   = time.Date(2026, 10, 2, 16, 59, 59, 999_000_000, time.UTC) // 2026-10-02 23:59:59.999 VN: last instant
	bizAfter  = time.Date(2026, 10, 2, 17, 0, 0, 0, time.UTC)             // 2026-10-03 00:00:00.000 VN: next day
)

const bizDay = "2026-10-02"

// Reviewer regression: a valid calendar label can still yield a typed lower
// bound outside DATETIME under the supported UTC driver location.
func TestBusinessReviewRejectsUnrepresentableUTCStart(t *testing.T) {
	for _, to := range []string{"", "1000-01-01", "1000-01-02"} {
		r, err := parseBusinessRange("1000-01-01", to)
		if err != errInvalidDateRange {
			t.Errorf("from=1000-01-01 to=%q admitted lower bound %v: error=%v", to, r.From, err)
		}
	}
	// Adjacent-day and to-only requests have representable actual bounds.
	for _, pair := range [][2]string{{"1000-01-02", "1000-01-02"}, {"", "1000-01-01"}} {
		r, err := parseBusinessRange(pair[0], pair[1])
		if err != nil {
			t.Fatalf("representable bounds rejected: %v: %v", pair, err)
		}
		for _, bound := range []*time.Time{r.From, r.ToExclusive} {
			if bound != nil && bound.UTC().Before(businessDateMin) {
				t.Fatalf("out-of-range UTC bound: %v", bound.UTC())
			}
		}
	}
}

func TestBusinessReviewInvalidStartReturns400BeforeQueries(t *testing.T) {
	// This test needs no database; all paths must reject before reading one.
	for name, handler := range map[string]gin.HandlerFunc{
		"dashboard": GetDashboard, "results": ListResults,
		"results_export": ExportResults, "cost_logs": ListCostLogs,
		"messages_export": ExportMessages,
	} {
		t.Run(name, func(t *testing.T) {
			rec := bizGet(handler, "unused", "/", "from=1000-01-01&to=1000-01-02&format=csv")
			if rec.Code != http.StatusBadRequest || rec.Body.String() != `{"error":"invalid_date_range"}` {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestBusinessReviewValidatesOnlySuppliedUpperBoundary(t *testing.T) {
	r, err := parseBusinessRange("9999-12-31", "")
	if err != nil || r.From == nil || r.ToExclusive != nil {
		t.Fatalf("representable from-only final day rejected: range=%+v error=%v", r, err)
	}
	// An inclusive to on the same final day needs an unrepresentable next
	// midnight on the VN storage connection and must still be rejected.
	if _, err := parseBusinessRange("", "9999-12-31"); err != errInvalidDateRange {
		t.Fatalf("unrepresentable exclusive upper bound accepted: %v", err)
	}
}

var bizStorageLocations = []string{"UTC", "Asia/Ho_Chi_Minh"}

var bizLocRe = regexp.MustCompile(`loc=[^&]*`)

// bizConnect reconnects the global pool with the given driver location. The previous pool is
// closed first and the new one is closed when the test ends (registered before the fixture
// cleanups, which therefore still find an open pool).
func bizConnect(t *testing.T, loc string) {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("bo qua: TEST_DB_DSN chua duoc thiet lap")
	}
	repl := "loc=" + url.QueryEscape(loc)
	if bizLocRe.MatchString(dsn) {
		dsn = bizLocRe.ReplaceAllString(dsn, repl)
	} else {
		dsn += "&" + repl
	}
	db.Close()
	if err := db.Connect(dsn, false); err != nil {
		t.Skipf("bo qua: khong ket noi duoc DB test: %v", err)
	}
	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate loi: %v", err)
	}
	t.Cleanup(db.Close)
}

// forEachStorage runs body once per real storage encoding.
func forEachStorage(t *testing.T, body func(t *testing.T, loc string)) {
	for _, loc := range bizStorageLocations {
		loc := loc
		t.Run("loc="+strings.ReplaceAll(loc, "/", "_"), func(t *testing.T) {
			bizConnect(t, loc)
			body(t, loc)
		})
	}
}

func bizExec(t *testing.T, sql string, args ...interface{}) {
	t.Helper()
	if err := db.DB.Exec(sql, args...).Error; err != nil {
		t.Fatalf("fixture: %v", err)
	}
}

// bizRaw reads a stored column as text, bypassing the driver's time conversion.
func bizRaw(t *testing.T, table, col, id string) string {
	t.Helper()
	var s string
	if err := db.DB.Raw("SELECT CAST("+col+" AS CHAR) FROM "+table+" WHERE id = ?", id).Scan(&s).Error; err != nil {
		t.Fatal(err)
	}
	return s
}

// bizWantRaw is the stored wall time of an instant under a driver location.
func bizWantRaw(at time.Time, loc string) string {
	l, err := time.LoadLocation(loc)
	if err != nil {
		panic(err)
	}
	return at.In(l).Format("2006-01-02 15:04:05.000")
}

// bizSuffix gives each fixture unique IDs so reruns and parallel packages never collide.
func bizSuffix() string { return pkg.NewUUID()[:8] }

// bizGet runs a handler against a tenant with the given raw query and returns the recorder.
func bizGet(h gin.HandlerFunc, tenantID, path, rawQuery string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set("tenant_id", tenantID)
	target := path
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	h(c)
	return rec
}

// ---- parser / boundary unit tests (no database) ----

func TestParseBusinessRangeExclusiveNextMidnight(t *testing.T) {
	r, err := parseBusinessRange(bizDay, bizDay)
	if err != nil || r.From == nil || r.ToExclusive == nil {
		t.Fatalf("same-day range: %+v, %v", r, err)
	}
	if !r.From.Equal(bizStart) || !r.ToExclusive.Equal(bizAfter) {
		t.Fatalf("same-day range = [%v, %v), want [%v, %v)", r.From.UTC(), r.ToExclusive.UTC(), bizStart, bizAfter)
	}
	// The interval keeps the full day, including stored milliseconds.
	if bizBefore.Before(*r.From) == false || !bizLast.Before(*r.ToExclusive) || bizAfter.Before(*r.ToExclusive) {
		t.Fatal("named boundary instants do not fall on the expected sides")
	}
	one, err := parseBusinessRange("", bizDay)
	if err != nil || one.From != nil || one.ToExclusive == nil {
		t.Fatalf("to only: %+v, %v", one, err)
	}
	none, err := parseBusinessRange("", "")
	if err != nil || none.From != nil || none.ToExclusive != nil {
		t.Fatalf("no dates must be open: %+v, %v", none, err)
	}
}

func TestParseBusinessRangeRolloverAndLeapDays(t *testing.T) {
	cases := []struct {
		to        string
		wantExcl  time.Time
		wantValid bool
	}{
		{"2026-10-31", time.Date(2026, 10, 31, 17, 0, 0, 0, time.UTC), true}, // month rollover: Nov 1 00:00 VN
		{"2026-12-31", time.Date(2026, 12, 31, 17, 0, 0, 0, time.UTC), true}, // year rollover: Jan 1 00:00 VN
		{"2024-02-28", time.Date(2024, 2, 28, 17, 0, 0, 0, time.UTC), true},  // next day is the leap day
		{"2024-02-29", time.Date(2024, 2, 29, 17, 0, 0, 0, time.UTC), true},  // leap day itself
		{"2026-02-28", time.Date(2026, 2, 28, 17, 0, 0, 0, time.UTC), true},  // no leap day in 2026
		{"2026-02-29", time.Time{}, false},
		{"2100-02-29", time.Time{}, false}, // century is not a leap year
		{"2000-02-29", time.Date(2000, 2, 29, 17, 0, 0, 0, time.UTC), true},
	}
	for _, c := range cases {
		r, err := parseBusinessRange("", c.to)
		if c.wantValid != (err == nil) {
			t.Errorf("to=%s: err %v, want valid=%v", c.to, err, c.wantValid)
			continue
		}
		if c.wantValid && !r.ToExclusive.Equal(c.wantExcl) {
			t.Errorf("to=%s: exclusive bound %v, want %v", c.to, r.ToExclusive.UTC(), c.wantExcl)
		}
	}
}

func TestParseBusinessRangeRejectsBadInput(t *testing.T) {
	bad := []struct{ from, to string }{
		{"2026-13-01", ""}, {"2026-00-10", ""}, {"2026-02-30", ""}, {"20261002", ""}, {"2026-1-2", ""},
		{"2026-10-02x", ""}, {" 2026-10-02", ""}, {"2026-10-02 ", ""}, {"abc", ""}, {"2026/10/02", ""},
		{"", "2026-10-32"}, {"", "xx"}, {"2026-10-02T00:00:00Z", ""},
		{"2026-10-03", "2026-10-02"},           // reversed
		{"0999-12-31", ""}, {"", "0999-12-31"}, // before the DATETIME range
		{"", "9999-12-31"}, // only a supplied inclusive `to` needs the next midnight
	}
	for _, c := range bad {
		if _, err := parseBusinessRange(c.from, c.to); err != errInvalidDateRange {
			t.Errorf("from=%q to=%q: err %v, want invalid_date_range", c.from, c.to, err)
		}
	}
	good := []struct{ from, to string }{
		{"1000-01-02", ""}, {"9999-12-31", ""}, {"", "9999-12-30"}, {bizDay, bizDay}, {"2026-10-02", "2026-10-03"},
	}
	for _, c := range good {
		if _, err := parseBusinessRange(c.from, c.to); err != nil {
			t.Errorf("from=%q to=%q: unexpected %v", c.from, c.to, err)
		}
	}
}

// The business day follows Asia/Ho_Chi_Minh, not the server clock zone.
func TestBusinessDayStartIsIndependentOfServerZone(t *testing.T) {
	orig := time.Local
	t.Cleanup(func() { time.Local = orig })
	for _, zone := range []string{"UTC", "America/New_York", "Asia/Ho_Chi_Minh", "Pacific/Auckland"} {
		l, err := time.LoadLocation(zone)
		if err != nil {
			t.Fatal(err)
		}
		time.Local = l
		cases := []struct {
			at   time.Time
			want time.Time
		}{
			{time.Date(2026, 10, 1, 16, 59, 59, 999_000_000, time.UTC), time.Date(2026, 9, 30, 17, 0, 0, 0, time.UTC)},
			{time.Date(2026, 10, 1, 17, 0, 0, 0, time.UTC), bizStart},
			{time.Date(2026, 10, 2, 0, 30, 0, 0, time.UTC), bizStart}, // 07:30 VN
			{time.Date(2026, 10, 2, 16, 59, 59, 999_000_000, time.UTC), bizStart},
			{time.Date(2026, 10, 2, 17, 0, 0, 0, time.UTC), bizAfter},
		}
		for _, c := range cases {
			if got := businessDayStart(c.at); !got.Equal(c.want) {
				t.Errorf("zone %s: businessDayStart(%v) = %v, want %v", zone, c.at, got.UTC(), c.want)
			}
		}
		if got := businessDayKey(bizLast.In(l)); got != bizDay {
			t.Errorf("zone %s: key %s", zone, got)
		}
	}
}

func TestDashboardWindowBoundaries(t *testing.T) {
	// Early VN hours (00:00, 00:30, 06:59, 07:00) and a clock just before midnight.
	clocks := []struct {
		at        time.Time
		wantToday string
	}{
		{time.Date(2026, 10, 1, 17, 0, 0, 0, time.UTC), "2026-10-02"},             // 00:00:00.000 VN
		{time.Date(2026, 10, 1, 17, 30, 0, 0, time.UTC), "2026-10-02"},            // 00:30 VN
		{time.Date(2026, 10, 1, 23, 59, 0, 0, time.UTC), "2026-10-02"},            // 06:59 VN
		{time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), "2026-10-02"},              // 07:00 VN
		{time.Date(2026, 10, 2, 16, 59, 59, 999_000_000, time.UTC), "2026-10-02"}, // 23:59:59.999 VN
		{time.Date(2026, 10, 1, 16, 59, 59, 999_000_000, time.UTC), "2026-10-01"}, // 23:59:59.999 VN, day before
	}
	for _, c := range clocks {
		w := newDashboardWindow(c.at)
		if businessDayKey(w.todayStart) != c.wantToday {
			t.Errorf("clock %v: today %s, want %s", c.at, businessDayKey(w.todayStart), c.wantToday)
		}
		if !w.tomorrowStart.Equal(w.todayStart.AddDate(0, 0, 1)) {
			t.Errorf("clock %v: tomorrow bound wrong", c.at)
		}
	}
	// Month and year edges, February of a leap year, series horizon.
	w := newDashboardWindow(time.Date(2024, 2, 29, 5, 0, 0, 0, time.UTC))
	if businessDayKey(w.monthStart) != "2024-02-01" || businessDayKey(w.nextMonth) != "2024-03-01" {
		t.Errorf("leap month: %s .. %s", businessDayKey(w.monthStart), businessDayKey(w.nextMonth))
	}
	w = newDashboardWindow(time.Date(2026, 12, 31, 20, 0, 0, 0, time.UTC)) // 03:00 VN on 2027-01-01
	if businessDayKey(w.todayStart) != "2027-01-01" || businessDayKey(w.monthStart) != "2027-01-01" || businessDayKey(w.nextMonth) != "2027-02-01" {
		t.Errorf("year rollover: today %s month %s next %s", businessDayKey(w.todayStart), businessDayKey(w.monthStart), businessDayKey(w.nextMonth))
	}
	w = newDashboardWindow(time.Date(2026, 10, 1, 17, 30, 0, 0, time.UTC))
	if businessDayKey(w.seriesStart) != "2026-09-02" || len(w.seriesDayStarts()) != 31 {
		t.Errorf("series horizon %s with %d dates", businessDayKey(w.seriesStart), len(w.seriesDayStarts()))
	}
}

// The default dashboard interval is today; an explicit side leaves the other side open.
func TestDashboardRangeDefaultsAndOneSided(t *testing.T) {
	w := newDashboardWindow(time.Date(2026, 10, 1, 17, 30, 0, 0, time.UTC))
	r, err := dashboardRange(w, "", "")
	if err != nil || !r.From.Equal(bizStart) || !r.ToExclusive.Equal(bizAfter) {
		t.Fatalf("default range %+v, %v", r, err)
	}
	r, err = dashboardRange(w, bizDay, "")
	if err != nil || r.From == nil || r.ToExclusive != nil {
		t.Fatalf("from only: %+v, %v", r, err)
	}
	r, err = dashboardRange(w, "", bizDay)
	if err != nil || r.From != nil || r.ToExclusive == nil {
		t.Fatalf("to only: %+v, %v", r, err)
	}
	if _, err = dashboardRange(w, "nope", ""); err != errInvalidDateRange {
		t.Fatalf("invalid default case: %v", err)
	}
}

// The matrix is genuine: the same instant is stored with different wall times under the two
// driver locations (and read back as the same instant).
func TestStorageMatrixStoresDifferentWallTimesForTheSameInstants(t *testing.T) {
	forEachStorage(t, func(t *testing.T, loc string) {
		s := bizSuffix()
		tenant := "biz-matrix-" + s
		bizExec(t, `INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Biz', ?, '{}', NOW(), NOW())`, tenant, tenant)
		t.Cleanup(func() {
			db.DB.Exec("DELETE FROM ai_usage_logs WHERE tenant_id = ?", tenant)
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tenant)
		})
		for i, at := range []time.Time{bizBefore, bizStart, bizLast, bizAfter} {
			id := tenant + "-" + string(rune('a'+i))
			bizExec(t, `INSERT INTO ai_usage_logs (id, tenant_id, job_id, job_run_id, provider, model, input_tokens, output_tokens, cost_usd, created_at) VALUES (?, ?, '', '', 'p', 'm', 1, 1, 1, ?)`, id, tenant, at)
			if got, want := bizRaw(t, "ai_usage_logs", "created_at", id), bizWantRaw(at, loc); got != want {
				t.Fatalf("%s: stored %q, want %q under loc=%s", id, got, want, loc)
			}
		}
	})
}

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-026 (F04): the Dashboard on Vietnam business days. The real GetDashboard runs on
// both storage encodings (driver loc=UTC and loc=Asia/Ho_Chi_Minh) with the named boundary
// instants of 2026-10-02 VN; two tenants hold identical rows so any missing tenant predicate or
// upper bound changes an asserted number.

type dashBiz struct {
	tenant, other string
	chA, chA2     string
	label         map[string]string // row label -> id
}

// Month, series and cost fixtures (instants, cost). Each cost is a distinct power of two so any
// subset sum identifies exactly which rows were counted.
var dashBizCosts = []struct {
	label string
	at    time.Time
	cost  float64
}{
	{"before", bizBefore, 1},
	{"start", bizStart, 2},
	{"last", bizLast, 4},
	{"after", bizAfter, 8},
	{"future-same-month", time.Date(2026, 10, 5, 3, 0, 0, 0, time.UTC), 16},
	{"sep30-last", time.Date(2026, 9, 30, 16, 59, 59, 999_000_000, time.UTC), 32},     // Sep 30 23:59:59.999 VN
	{"oct1-first", time.Date(2026, 9, 30, 17, 0, 0, 0, time.UTC), 64},                 // Oct 1 00:00 VN
	{"oct31-last", time.Date(2026, 10, 31, 16, 59, 59, 999_000_000, time.UTC), 128},   // Oct 31 23:59:59.999 VN
	{"nov1-first", time.Date(2026, 10, 31, 17, 0, 0, 0, time.UTC), 256},               // Nov 1 00:00 VN
	{"horizon-before", time.Date(2026, 9, 1, 16, 59, 59, 999_000_000, time.UTC), 512}, // Sep 1 23:59:59.999 VN: before the series
	{"horizon-first", time.Date(2026, 9, 1, 17, 0, 0, 0, time.UTC), 1024},             // Sep 2 00:00 VN: first series instant
}

func setupDashBiz(t *testing.T) *dashBiz {
	t.Helper()
	s := bizSuffix()
	f := &dashBiz{tenant: "bizd-a-" + s, other: "bizd-b-" + s, chA: "ch-bizd-a-" + s, chA2: "ch-bizd-a2-" + s, label: map[string]string{}}
	t.Cleanup(func() {
		for _, tn := range []string{f.tenant, f.other} {
			for _, table := range []string{"job_results", "ai_usage_logs", "messages", "conversations", "channels"} {
				db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tn)
			}
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tn)
		}
	})
	for _, tn := range []string{f.tenant, f.other} {
		bizExec(t, `INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Biz', ?, '{}', NOW(), NOW())`, tn, tn)
	}
	chans := map[string][2]string{f.tenant: {f.chA, f.chA2}, f.other: {"ch-bizd-b-" + s, "ch-bizd-b2-" + s}}
	for tn, pair := range chans {
		bizExec(t, `INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'K1', ?, X'00', true, '{}', NOW(), NOW())`, pair[0], tn, "e1-"+pair[0])
		bizExec(t, `INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'zalo_oa', 'K2', ?, X'00', true, '{}', NOW(), NOW())`, pair[1], tn, "e2-"+pair[1])
	}
	// Four conversations per tenant at the four boundary instants: before/start on the first
	// channel type, last/after on the second.
	for tn, pair := range chans {
		tag := tn[5:6] + tn[len(tn)-8:] // tenant letter + suffix: IDs are globally unique keys
		for i, at := range []struct {
			label string
			t     time.Time
		}{{"before", bizBefore}, {"start", bizStart}, {"last", bizLast}, {"after", bizAfter}} {
			ch := pair[0]
			if i >= 2 {
				ch = pair[1]
			}
			convID := fmt.Sprintf("conv-%s-%s", at.label, tag)
			f.label[tn+"/conv/"+at.label] = convID
			bizExec(t, `INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 1, '{}', NOW(), NOW())`,
				convID, tn, ch, "ext-"+convID, "KH-"+at.label, at.t)
			// One result of each kind per boundary instant, on that conversation.
			for _, rt := range []string{"qc_violation", "classification_tag", "conversation_evaluation"} {
				id := fmt.Sprintf("res-%s-%s-%s", rt[:2], at.label, tag)
				f.label[tn+"/"+rt+"/"+at.label] = id
				bizExec(t, `INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, created_at) VALUES (?, 'run', ?, ?, ?, 'CAN_CAI_THIEN', 'R', 'E', '{}', ?)`, id, tn, convID, rt, at.t)
			}
		}
		for _, c := range dashBizCosts {
			id := fmt.Sprintf("use-%s-%s", c.label, tag)
			bizExec(t, `INSERT INTO ai_usage_logs (id, tenant_id, job_id, job_run_id, provider, model, input_tokens, output_tokens, cost_usd, created_at) VALUES (?, ?, '', '', 'p', 'm', ?, ?, ?, ?)`,
				id, tn, int(c.cost), int(c.cost)*2, c.cost, c.at)
		}
		// Messages: the day's first and last instants, one agent reply, the next day's first
		// instant, the previous day's last instant and both series-horizon edges.
		msgs := []struct {
			label  string
			at     time.Time
			sender string
			conv   string
		}{
			{"m-before", bizBefore, "customer", "before"},
			{"m-start", bizStart, "customer", "start"},
			{"m-reply", bizStart.Add(time.Hour), "agent", "start"},
			{"m-last", bizLast, "customer", "last"},
			{"m-after", bizAfter, "customer", "after"},
			{"m-horizon-before", time.Date(2026, 9, 1, 16, 59, 59, 999_000_000, time.UTC), "customer", "before"},
			{"m-horizon-first", time.Date(2026, 9, 1, 17, 0, 0, 0, time.UTC), "customer", "before"},
		}
		for _, m := range msgs {
			bizExec(t, `INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, ?, ?, 'N', ?, 'text', '[]', ?, NOW())`,
				pkg.NewUUID(), tn, f.label[tn+"/conv/"+m.conv], m.label+"-"+tag, m.sender, "noi dung "+m.label, m.at)
		}
	}
	return f
}

func (f *dashBiz) id(kind, label string) string { return f.label[f.tenant+"/"+kind+"/"+label] }

func (f *dashBiz) get(t *testing.T, clock time.Time, query string) (map[string]interface{}, int) {
	t.Helper()
	orig := businessClock
	businessClock = func() time.Time { return clock }
	t.Cleanup(func() { businessClock = orig })
	rec := bizGet(GetDashboard, f.tenant, "/api/v1/dashboard", query)
	var body map[string]interface{}
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("response: %v: %s", err, rec.Body.String())
		}
	}
	return body, rec.Code
}

func num(body map[string]interface{}, key string) float64 {
	v, _ := body[key].(float64)
	return v
}

func idsOf(t *testing.T, body map[string]interface{}, key string) []string {
	t.Helper()
	rows, _ := body[key].([]interface{})
	var ids []string
	for _, r := range rows {
		m := r.(map[string]interface{})
		ids = append(ids, m["id"].(string))
	}
	sort.Strings(ids)
	return ids
}

func sorted(ids ...string) []string {
	out := append([]string(nil), ids...)
	sort.Strings(out)
	return out
}

func sameIDs(a, b []string) bool { return strings.Join(a, ",") == strings.Join(b, ",") }

// Filtered cards, channel counts, recent QC/classification rows and cost_period share the same
// [00:00, next 00:00) interval; the named instants decide inclusion; other tenants never leak.
func TestDashboardFilterUsesVietnamDayBounds(t *testing.T) {
	forEachStorage(t, func(t *testing.T, loc string) {
		f := setupDashBiz(t)
		clock := time.Date(2026, 10, 1, 17, 30, 0, 0, time.UTC) // 00:30 VN on 2026-10-02
		body, code := f.get(t, clock, "from="+bizDay+"&to="+bizDay)
		if code != http.StatusOK {
			t.Fatalf("status %d", code)
		}
		if got := num(body, "total_conversations"); got != 2 {
			t.Fatalf("total_conversations %v, want 2 (start and last only)", got)
		}
		if got := num(body, "issues"); got != 6 {
			t.Fatalf("issues %v, want 6 (three result kinds on two conversations)", got)
		}
		if got := num(body, "qc_violation_count"); got != 2 {
			t.Fatalf("qc_violation_count %v, want 2 (violation rows only)", got)
		}
		channels := map[string]float64{}
		for _, r := range body["conversations_by_channel"].([]interface{}) {
			m := r.(map[string]interface{})
			channels[m["channel_type"].(string)] = m["count"].(float64)
		}
		if channels["pancake"] != 1 || channels["zalo_oa"] != 1 || len(channels) != 2 {
			t.Fatalf("conversations_by_channel %v, want one pancake (start) and one zalo_oa (last)", channels)
		}
		if got := idsOf(t, body, "qc_alerts"); !sameIDs(got, sorted(f.id("qc_violation", "start"), f.id("qc_violation", "last"))) {
			t.Fatalf("qc_alerts %v", got)
		}
		if got := idsOf(t, body, "classification_recent"); !sameIDs(got, sorted(f.id("classification_tag", "start"), f.id("classification_tag", "last"))) {
			t.Fatalf("classification_recent %v", got)
		}
		if got := num(body, "cost_period"); got != 6 {
			t.Fatalf("cost_period %v, want 6 (start 2 + last 4)", got)
		}
	})
}

// today and month are bounded above and independent of the selected interval; the series are
// bucketed by Vietnam date and bounded by the 31-date horizon and tomorrow's midnight.
func TestDashboardCalendarAggregatesAndSeries(t *testing.T) {
	forEachStorage(t, func(t *testing.T, loc string) {
		f := setupDashBiz(t)
		clock := time.Date(2026, 10, 1, 17, 30, 0, 0, time.UTC) // 00:30 VN on 2026-10-02
		// A wildly different selected interval must not change today/month/series.
		for _, query := range []string{"", "from=2026-01-01&to=2026-01-02", "from=" + bizDay + "&to=" + bizDay} {
			body, code := f.get(t, clock, query)
			if code != http.StatusOK {
				t.Fatalf("%q: status %d", query, code)
			}
			if got := num(body, "cost_today"); got != 6 { // start 2 + last 4; before/after/future excluded
				t.Fatalf("%q: cost_today %v, want 6", query, got)
			}
			// October VN: before 1 + start 2 + last 4 + after 8 + future 16 + oct1-first 64 + oct31-last 128.
			if got := num(body, "cost_this_month"); got != 223 {
				t.Fatalf("%q: cost_this_month %v, want 223", query, got)
			}
			costs := map[string][3]float64{}
			var dates []string
			for _, r := range body["cost_by_day"].([]interface{}) {
				m := r.(map[string]interface{})
				d := m["date"].(string)
				dates = append(dates, d)
				costs[d] = [3]float64{m["total_cost"].(float64), m["input_tokens"].(float64) + m["output_tokens"].(float64), m["call_count"].(float64)}
			}
			wantCosts := map[string][3]float64{
				"2026-10-02": {6, 18, 2},      // start 2 + last 4
				"2026-10-01": {65, 195, 2},    // before 1 + oct1-first 64
				"2026-09-30": {32, 96, 1},     // sep30-last
				"2026-09-02": {1024, 3072, 1}, // horizon-first
			}
			if len(costs) != len(wantCosts) {
				t.Fatalf("%q: cost_by_day dates %v, want %v", query, dates, wantCosts)
			}
			for d, w := range wantCosts {
				if costs[d] != w {
					t.Fatalf("%q: cost_by_day[%s] = %v, want %v", query, d, costs[d], w)
				}
			}
			if !sort.SliceIsSorted(dates, func(i, j int) bool { return dates[i] > dates[j] }) {
				t.Fatalf("%q: cost_by_day must be newest first: %v", query, dates)
			}
			type mrow struct{ count, chat, reply float64 }
			msgs := map[string]mrow{}
			var mdates []string
			for _, r := range body["messages_by_day"].([]interface{}) {
				m := r.(map[string]interface{})
				d := m["date"].(string)
				mdates = append(mdates, d)
				msgs[d] = mrow{m["count"].(float64), m["chat_count"].(float64), m["reply_count"].(float64)}
			}
			wantMsgs := map[string]mrow{
				"2026-10-02": {3, 2, 1}, // start + reply + last: two distinct customer conversations, one agent reply
				"2026-10-01": {1, 1, 0}, // m-before is 23:59:59.999 VN on Oct 1
				"2026-09-02": {1, 1, 0}, // m-horizon-first; m-horizon-before is outside the series
			}
			if len(msgs) != len(wantMsgs) {
				t.Fatalf("%q: messages_by_day dates %v, want %v", query, mdates, wantMsgs)
			}
			for d, w := range wantMsgs {
				if msgs[d] != w {
					t.Fatalf("%q: messages_by_day[%s] = %v, want %v", query, d, msgs[d], w)
				}
			}
			if !sort.StringsAreSorted(mdates) {
				t.Fatalf("%q: messages_by_day must be oldest first: %v", query, mdates)
			}
		}
	})
}

// No dates means today's Vietnam day at the captured clock instant, including the first and last
// instants of the day and the minute before midnight; one explicit side leaves the other open.
func TestDashboardDefaultAndOneSidedRanges(t *testing.T) {
	forEachStorage(t, func(t *testing.T, loc string) {
		f := setupDashBiz(t)
		cases := []struct {
			name  string
			clock time.Time
			query string
			want  float64
		}{
			{"default at 00:00:00.000 VN", bizStart, "", 2},
			{"default at 06:59 VN", time.Date(2026, 10, 1, 23, 59, 0, 0, time.UTC), "", 2},
			{"default at 23:59:59.999 VN", bizLast, "", 2},
			{"default at 00:00:00.000 VN next day", bizAfter, "", 1},         // only 'after'
			{"default one millisecond before midnight VN", bizBefore, "", 1}, // only 'before'
			{"from only is open above", bizStart, "from=" + bizDay, 3},       // start, last, after
			{"to only is open below", bizStart, "to=2026-10-01", 1},          // before only
			{"to only the day itself", bizStart, "to=" + bizDay, 3},          // before, start, last
		}
		for _, c := range cases {
			body, code := f.get(t, c.clock, c.query)
			if code != http.StatusOK {
				t.Fatalf("%s: status %d", c.name, code)
			}
			if got := num(body, "total_conversations"); got != c.want {
				t.Errorf("%s: total_conversations %v, want %v", c.name, got, c.want)
			}
		}
	})
}

func TestDashboardInvalidDatesAreBadRequests(t *testing.T) {
	forEachStorage(t, func(t *testing.T, loc string) {
		f := setupDashBiz(t)
		for _, q := range []string{"from=2026-02-30", "to=2026-13-01", "from=2026-10-03&to=2026-10-02", "from=0999-12-31", "to=9999-12-31", "from=abc"} {
			orig := businessClock
			rec := bizGet(GetDashboard, f.tenant, "/api/v1/dashboard", q)
			businessClock = orig
			if rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != `{"error":"invalid_date_range"}` {
				t.Errorf("%q: %d %s", q, rec.Code, rec.Body.String())
			}
		}
	})
}

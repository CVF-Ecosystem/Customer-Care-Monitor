package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
)

// CCMAI-RUNTIME-026 (F04): Cost Logs on Vietnam business days. The list and its total share the
// parsed [from, to+1 day) interval; the provider filter and pagination keep their contracts.

type costBiz struct {
	tenant, other string
	ids           map[string]string // label -> id
}

func setupCostBiz(t *testing.T) *costBiz {
	t.Helper()
	s := bizSuffix()
	f := &costBiz{tenant: "bizc-a-" + s, other: "bizc-b-" + s, ids: map[string]string{}}
	t.Cleanup(func() {
		for _, tn := range []string{f.tenant, f.other} {
			db.DB.Exec("DELETE FROM ai_usage_logs WHERE tenant_id = ?", tn)
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tn)
		}
	})
	rows := []struct {
		label, provider string
		at              time.Time
	}{
		{"before", "claude", bizBefore},
		{"start", "claude", bizStart},
		{"last", "openai", bizLast},
		{"after", "openai", bizAfter},
	}
	for _, tn := range []string{f.tenant, f.other} {
		bizExec(t, `INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Biz', ?, '{}', NOW(), NOW())`, tn, tn)
		for _, r := range rows {
			id := "use-bizc-" + r.label + "-" + tn[5:6] + s
			bizExec(t, `INSERT INTO ai_usage_logs (id, tenant_id, job_id, job_run_id, provider, model, input_tokens, output_tokens, cost_usd, created_at) VALUES (?, ?, '', '', ?, 'm', 1, 1, 1, ?)`, id, tn, r.provider, r.at)
			if tn == f.tenant {
				f.ids[r.label] = id
			}
		}
	}
	return f
}

func (f *costBiz) list(t *testing.T, query string) ([]string, float64, int) {
	t.Helper()
	rec := bizGet(ListCostLogs, f.tenant, "/api/v1/cost-logs", query)
	if rec.Code != http.StatusOK {
		return nil, 0, rec.Code
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Total float64 `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	var ids []string
	for _, d := range body.Data {
		ids = append(ids, d.ID)
	}
	sort.Strings(ids)
	return ids, body.Total, rec.Code
}

func (f *costBiz) want(labels ...string) []string {
	var out []string
	for _, l := range labels {
		out = append(out, f.ids[l])
	}
	sort.Strings(out)
	return out
}

func TestCostLogsUseVietnamDayBounds(t *testing.T) {
	forEachStorage(t, func(t *testing.T, loc string) {
		f := setupCostBiz(t)
		cases := []struct {
			name  string
			query string
			want  []string
		}{
			{"same day", "from=" + bizDay + "&to=" + bizDay, []string{"start", "last"}},
			{"from only is open above", "from=" + bizDay, []string{"start", "last", "after"}},
			{"to only is open below", "to=2026-10-01", []string{"before"}},
			{"no dates is unrestricted", "", []string{"before", "start", "last", "after"}},
			{"provider filter keeps working inside the day", "from=" + bizDay + "&to=" + bizDay + "&provider=openai", []string{"last"}},
			{"two-day window", "from=2026-10-01&to=" + bizDay, []string{"before", "start", "last"}},
		}
		for _, c := range cases {
			ids, total, code := f.list(t, c.query)
			want := f.want(c.want...)
			if code != http.StatusOK || !sameIDs(ids, want) || int(total) != len(want) {
				t.Errorf("%s: status %d ids %v total %v, want %v", c.name, code, ids, total, want)
			}
		}
		// Pagination still counts the whole filtered set.
		rec := bizGet(ListCostLogs, f.tenant, "/api/v1/cost-logs", "from="+bizDay+"&to="+bizDay+"&per_page=1&page=2")
		var page struct {
			Data  []map[string]interface{} `json:"data"`
			Total float64                  `json:"total"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &page)
		if rec.Code != http.StatusOK || len(page.Data) != 1 || page.Total != 2 {
			t.Errorf("pagination: %d %d rows total %v", rec.Code, len(page.Data), page.Total)
		}
	})
}

func TestCostLogsInvalidDatesAreBadRequests(t *testing.T) {
	forEachStorage(t, func(t *testing.T, loc string) {
		f := setupCostBiz(t)
		for _, q := range []string{"from=2026-02-30", "to=2026-13-01", "from=2026-10-03&to=2026-10-02", "from=0999-12-31", "to=9999-12-31", "from=x", "to=2026-10-02x"} {
			rec := bizGet(ListCostLogs, f.tenant, "/api/v1/cost-logs", q)
			if rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != `{"error":"invalid_date_range"}` {
				t.Errorf("%q: %d %s", q, rec.Code, rec.Body.String())
			}
		}
	})
}

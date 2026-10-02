package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
)

// CCMAI-RUNTIME-026 (F04): Results list/export on Vietnam business days. Four conversations sit
// at the named boundary instants while their evaluations sit on deliberately different instants,
// so `date_field=conv` and `date_field=eval` select different sets. The same expected sets are
// asserted from the list response, the CSV and the XLSX; a second tenant holds the same rows.

type resBiz struct {
	tenant, other string
	names         map[string]string // label -> customer name (unique, tenant-bound)
	convs         map[string]string // label -> conversation id
}

// conversation instant, evaluation instant (the labels are the conversation's boundary names).
var resBizRows = []struct {
	label string
	conv  time.Time
	eval  time.Time
}{
	{"before", bizBefore, bizStart}, // conversation outside, evaluation inside
	{"start", bizStart, bizBefore},  // conversation inside, evaluation outside
	{"last", bizLast, bizAfter},     // conversation inside, evaluation outside
	{"after", bizAfter, bizLast},    // conversation outside, evaluation inside
}

func setupResBiz(t *testing.T) *resBiz {
	t.Helper()
	s := bizSuffix()
	f := &resBiz{tenant: "bizr-a-" + s, other: "bizr-b-" + s, names: map[string]string{}, convs: map[string]string{}}
	t.Cleanup(func() {
		for _, tn := range []string{f.tenant, f.other} {
			for _, table := range []string{"job_results", "job_runs", "conversations", "jobs", "channels"} {
				db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tn)
			}
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tn)
		}
	})
	for _, tn := range []string{f.tenant, f.other} {
		tag := tn[5:6] + s
		ch, job, run := "ch-bizr-"+tag, "job-bizr-"+tag, "run-bizr-"+tag
		bizExec(t, `INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Biz', ?, '{}', NOW(), NOW())`, tn, tn)
		bizExec(t, `INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'Kenh', ?, X'00', true, '{}', NOW(), NOW())`, ch, tn, "e-"+ch)
		bizExec(t, `INSERT INTO jobs (id, tenant_id, name, job_type, input_channel_ids, rules_content, rules_config, schedule_type, is_active, outputs, output_schedule, created_at, updated_at) VALUES (?, ?, 'QC', 'qc_analysis', '[]', '', '[]', 'manual', true, '[]', 'none', NOW(), NOW())`, job, tn)
		bizExec(t, `INSERT INTO job_runs (id, job_id, tenant_id, started_at, status, summary, created_at) VALUES (?, ?, ?, NOW(), 'success', '{}', NOW())`, run, job, tn)
		for _, r := range resBizRows {
			conv := "conv-bizr-" + r.label + "-" + tag
			name := "KH-" + r.label + "-" + tag
			bizExec(t, `INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 1, '{}', NOW(), NOW())`, conv, tn, ch, "ext-"+conv, name, r.conv)
			bizExec(t, `INSERT INTO job_results (id, job_run_id, tenant_id, conversation_id, result_type, severity, rule_name, evidence, detail, created_at) VALUES (?, ?, ?, ?, 'conversation_evaluation', 'PASS', '', 'ok', '{"score":90}', ?)`, "ev-bizr-"+r.label+"-"+tag, run, tn, conv, r.eval)
			if tn == f.tenant {
				f.names[r.label] = name
				f.convs[r.label] = conv
			}
		}
	}
	return f
}

type resBizList struct {
	ids   []string
	total float64
	all   float64
}

func (f *resBiz) list(t *testing.T, query string) (resBizList, int) {
	t.Helper()
	rec := bizGet(ListResults, f.tenant, "/api/v1/results", query)
	var out resBizList
	if rec.Code != http.StatusOK {
		return out, rec.Code
	}
	var body struct {
		Items []struct {
			ConversationID string `json:"conversation_id"`
		} `json:"items"`
		Total  float64            `json:"total"`
		Counts map[string]float64 `json:"counts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("list body: %v: %s", err, rec.Body.String())
	}
	for _, it := range body.Items {
		out.ids = append(out.ids, it.ConversationID)
	}
	sort.Strings(out.ids)
	out.total, out.all = body.Total, body.Counts["all"]
	return out, rec.Code
}

func (f *resBiz) wantIDs(labels ...string) []string {
	var out []string
	for _, l := range labels {
		out = append(out, f.convs[l])
	}
	sort.Strings(out)
	return out
}

func (f *resBiz) csvNames(t *testing.T, query string) []string {
	t.Helper()
	rec := bizGet(ExportResults, f.tenant, "/api/v1/results/export", query+"&format=csv")
	if rec.Code != http.StatusOK {
		t.Fatalf("csv export %q: %d %s", query, rec.Code, rec.Body.String())
	}
	var out []string
	for _, line := range strings.Split(strings.TrimPrefix(rec.Body.String(), "\xEF\xBB\xBF"), "\n")[1:] {
		if line == "" {
			continue
		}
		out = append(out, strings.Split(strings.Trim(line, `"`), `","`)[0])
	}
	sort.Strings(out)
	return out
}

func (f *resBiz) xlsxNames(t *testing.T, query string) []string {
	t.Helper()
	rec := bizGet(ExportResults, f.tenant, "/api/v1/results/export", query+"&format=xlsx")
	if rec.Code != http.StatusOK {
		t.Fatalf("xlsx export %q: %d %s", query, rec.Code, rec.Body.String())
	}
	book, err := excelize.OpenReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("xlsx open: %v", err)
	}
	rows, err := book.GetRows("Results")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range rows[1:] {
		out = append(out, r[0])
	}
	sort.Strings(out)
	return out
}

func (f *resBiz) wantNames(labels ...string) []string {
	var out []string
	for _, l := range labels {
		out = append(out, f.names[l])
	}
	sort.Strings(out)
	return out
}

// The same boundary cases drive the list, the CSV and the XLSX: list totals, counts and exported
// customers must be identical for each date_field.
func TestResultsListAndExportsShareVietnamDayBounds(t *testing.T) {
	forEachStorage(t, func(t *testing.T, loc string) {
		f := setupResBiz(t)
		cases := []struct {
			name  string
			query string
			want  []string
		}{
			{"conv mode, same day", "from=" + bizDay + "&to=" + bizDay + "&date_field=conv", []string{"start", "last"}},
			{"eval mode, same day", "from=" + bizDay + "&to=" + bizDay + "&date_field=eval", []string{"before", "after"}},
			{"default date_field is conv", "from=" + bizDay + "&to=" + bizDay, []string{"start", "last"}},
			{"conv from only", "from=" + bizDay + "&date_field=conv", []string{"start", "last", "after"}},
			{"conv to only (day before)", "to=2026-10-01&date_field=conv", []string{"before"}},
			// Evaluations of the before/start/after rows are earlier than 2026-10-03 00:00 VN.
			{"eval to only (the day itself)", "to=" + bizDay + "&date_field=eval", []string{"before", "start", "after"}},
			{"no dates is unrestricted", "date_field=conv", []string{"before", "start", "last", "after"}},
		}
		for _, c := range cases {
			wantIDs := f.wantIDs(c.want...)
			got, code := f.list(t, c.query)
			if code != http.StatusOK {
				t.Fatalf("%s: status %d", c.name, code)
			}
			if !sameIDs(got.ids, wantIDs) || int(got.total) != len(wantIDs) || int(got.all) != len(wantIDs) {
				t.Errorf("%s: list ids %v total %v counts.all %v, want %v", c.name, got.ids, got.total, got.all, wantIDs)
			}
			wantNames := f.wantNames(c.want...)
			if got := f.csvNames(t, c.query); !sameIDs(got, wantNames) {
				t.Errorf("%s: csv customers %v, want %v", c.name, got, wantNames)
			}
			if got := f.xlsxNames(t, c.query); !sameIDs(got, wantNames) {
				t.Errorf("%s: xlsx customers %v, want %v", c.name, got, wantNames)
			}
		}
	})
}

// Exported times are printed on the Vietnam calendar (the one the filter uses), whatever the
// driver decoded them as.
func TestResultsExportPrintsVietnamTimes(t *testing.T) {
	forEachStorage(t, func(t *testing.T, loc string) {
		f := setupResBiz(t)
		rec := bizGet(ExportResults, f.tenant, "/api/v1/results/export", "from="+bizDay+"&to="+bizDay+"&format=csv")
		body := rec.Body.String()
		for _, want := range []string{
			f.names["start"],   // sanity: the in-range rows are present
			"2026-10-02 00:00", // conversation 'start' at 00:00 VN
			"2026-10-02 23:59", // conversation 'last' at 23:59 VN
		} {
			if !strings.Contains(body, want) {
				t.Errorf("csv lacks %q:\n%s", want, body)
			}
		}
		if strings.Contains(body, "2026-10-01 17:00") || strings.Contains(body, "2026-10-02 16:59") {
			t.Errorf("csv prints UTC wall times:\n%s", body)
		}
	})
}

func TestResultsInvalidDatesAreBadRequestsWithoutFileBytes(t *testing.T) {
	forEachStorage(t, func(t *testing.T, loc string) {
		f := setupResBiz(t)
		bad := []string{
			"from=2026-02-30", "to=2026-13-01", "from=2026-10-03&to=2026-10-02", "from=0999-12-31",
			"to=9999-12-31", "from=abc", "from=2026-1-2", "to=2026-10-02x", "from=" + url.QueryEscape("2026-10-02 "),
		}
		for _, q := range bad {
			rec := bizGet(ListResults, f.tenant, "/api/v1/results", q)
			if rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != `{"error":"invalid_date_range"}` {
				t.Errorf("list %q: %d %s", q, rec.Code, rec.Body.String())
			}
			for _, format := range []string{"csv", "xlsx"} {
				rec := bizGet(ExportResults, f.tenant, "/api/v1/results/export", q+"&format="+format)
				ct := rec.Header().Get("Content-Type")
				if rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != `{"error":"invalid_date_range"}` || strings.Contains(ct, "csv") || strings.Contains(ct, "spreadsheet") {
					t.Errorf("export %s %q: %d %q %s", format, q, rec.Code, ct, rec.Body.String())
				}
			}
		}
		// Positive control for the same harness: a valid pair returns a file.
		rec := bizGet(ExportResults, f.tenant, "/api/v1/results/export", "from="+bizDay+"&to="+bizDay+"&format=csv")
		if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "csv") {
			t.Fatalf("valid export: %d %q", rec.Code, rec.Header().Get("Content-Type"))
		}
	})
}

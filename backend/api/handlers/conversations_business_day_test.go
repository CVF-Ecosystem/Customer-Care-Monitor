package handlers

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
)

// CCMAI-RUNTIME-026 (F04): message export selects conversations by last_message_at on Vietnam
// business days and still exports every message of each selected conversation. Both dates stay
// required; invalid input is a bounded 400 with no file bytes.

type msgBiz struct {
	tenant, other string
	chPancake     string
	chZalo        string
}

type exportResult struct {
	code        int
	body        string
	contentType string
}

func setupMsgBiz(t *testing.T) *msgBiz {
	t.Helper()
	s := bizSuffix()
	f := &msgBiz{tenant: "bizm-a-" + s, other: "bizm-b-" + s}
	t.Cleanup(func() {
		for _, tn := range []string{f.tenant, f.other} {
			for _, table := range []string{"messages", "conversations", "channels"} {
				db.DB.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tn)
			}
			db.DB.Exec("DELETE FROM tenants WHERE id = ?", tn)
		}
	})
	convs := []struct {
		label string
		at    time.Time
		zalo  bool
	}{
		{"before", bizBefore, false},
		{"start", bizStart, false},
		{"last", bizLast, true},
		{"after", bizAfter, true},
	}
	for _, tn := range []string{f.tenant, f.other} {
		tag := tn[5:6] + s
		bizExec(t, `INSERT INTO tenants (id, name, slug, settings, created_at, updated_at) VALUES (?, 'Biz', ?, '{}', NOW(), NOW())`, tn, tn)
		pan, zal := "ch-bizm-p-"+tag, "ch-bizm-z-"+tag
		bizExec(t, `INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'pancake', 'P', ?, X'00', true, '{}', NOW(), NOW())`, pan, tn, "e-"+pan)
		bizExec(t, `INSERT INTO channels (id, tenant_id, channel_type, name, external_id, credentials_encrypted, is_active, metadata, created_at, updated_at) VALUES (?, ?, 'zalo_oa', 'Z', ?, X'00', true, '{}', NOW(), NOW())`, zal, tn, "e-"+zal)
		if tn == f.tenant {
			f.chPancake, f.chZalo = pan, zal
		}
		for _, c := range convs {
			ch := pan
			if c.zalo {
				ch = zal
			}
			id := "conv-bizm-" + c.label + "-" + tag
			bizExec(t, `INSERT INTO conversations (id, tenant_id, channel_id, external_conversation_id, customer_name, last_message_at, message_count, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 2, '{}', NOW(), NOW())`, id, tn, ch, "ext-"+id, "KH-"+c.label+"-"+tag, c.at)
			// An early message (three days before) and the last message: the whole conversation
			// is exported when the conversation is selected, whatever each message's own date.
			bizExec(t, `INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, ?, 'customer', 'Khach', ?, 'text', '[]', ?, NOW())`,
				"msg-early-"+c.label+"-"+tag, tn, id, "x1-"+c.label+"-"+tag, "EARLY-"+c.label+"-"+tag, c.at.Add(-72*time.Hour))
			bizExec(t, `INSERT INTO messages (id, tenant_id, conversation_id, external_message_id, sender_type, sender_name, content, content_type, attachments, sent_at, created_at) VALUES (?, ?, ?, ?, 'agent', 'NV', ?, 'text', '[]', ?, NOW())`,
				"msg-late-"+c.label+"-"+tag, tn, id, "x2-"+c.label+"-"+tag, "LATE-"+c.label+"-"+tag, c.at)
		}
	}
	return f
}

func (f *msgBiz) export(query string) exportResult {
	rec := bizGet(ExportMessages, f.tenant, "/api/v1/conversations/export", query)
	return exportResult{code: rec.Code, body: rec.Body.String(), contentType: rec.Header().Get("Content-Type")}
}

func (f *msgBiz) tag() string { return f.tenant[5:6] + f.tenant[len(f.tenant)-8:] }

// has reports which conversation labels have BOTH their early and late message in the body, and
// flags a conversation exported with only one of them.
func (f *msgBiz) has(body string) map[string]bool {
	out := map[string]bool{}
	for _, l := range []string{"before", "start", "last", "after"} {
		early, late := strings.Contains(body, "EARLY-"+l+"-"+f.tag()), strings.Contains(body, "LATE-"+l+"-"+f.tag())
		if early != late {
			out["PARTIAL-"+l] = true
		}
		if early && late {
			out[l] = true
		}
	}
	return out
}

func TestMessageExportUsesVietnamDayBoundsAndExportsWholeConversations(t *testing.T) {
	forEachStorage(t, func(t *testing.T, loc string) {
		f := setupMsgBiz(t)
		cases := []struct {
			name  string
			query string
			want  []string
		}{
			{"same day", "from=" + bizDay + "&to=" + bizDay, []string{"start", "last"}},
			{"two days", "from=2026-10-01&to=" + bizDay, []string{"before", "start", "last"}},
			{"channel_id filter", "from=" + bizDay + "&to=" + bizDay + "&channel_id=" + f.chPancake, []string{"start"}},
			{"channel_type filter", "from=" + bizDay + "&to=" + bizDay + "&channel_type=zalo_oa", []string{"last"}},
		}
		for _, format := range []string{"txt", "csv"} {
			for _, c := range cases {
				got := f.export(c.query + "&format=" + format)
				if got.code != http.StatusOK {
					t.Fatalf("%s/%s: status %d %s", format, c.name, got.code, got.body)
				}
				have := f.has(got.body)
				for k := range have {
					if strings.HasPrefix(k, "PARTIAL-") {
						t.Errorf("%s/%s: %s exported only partially (all messages must be exported)", format, c.name, k)
					}
				}
				for _, l := range []string{"before", "start", "last", "after"} {
					wantIt := false
					for _, w := range c.want {
						if w == l {
							wantIt = true
						}
					}
					if have[l] != wantIt {
						t.Errorf("%s/%s: conversation %s exported=%v, want %v", format, c.name, l, have[l], wantIt)
					}
				}
				// The other tenant's identical rows never appear.
				if strings.Contains(got.body, "-b"+f.tenant[len(f.tenant)-8:]) {
					t.Errorf("%s/%s: another tenant's rows leaked", format, c.name)
				}
			}
		}
	})
}

func TestMessageExportDateValidation(t *testing.T) {
	forEachStorage(t, func(t *testing.T, loc string) {
		f := setupMsgBiz(t)
		bad := []string{
			"from=2026-02-30&to=2026-10-02", "from=2026-10-02&to=2026-13-01", "from=2026-10-03&to=2026-10-02",
			"from=0999-12-31&to=2026-10-02", "from=2026-10-02&to=9999-12-31", "from=abc&to=2026-10-02", "from=2026-10-02&to=2026-10-02x",
		}
		for _, q := range bad {
			for _, format := range []string{"txt", "csv"} {
				got := f.export(q + "&format=" + format)
				if got.code != http.StatusBadRequest || strings.TrimSpace(got.body) != `{"error":"invalid_date_range"}` || strings.Contains(got.contentType, "csv") || strings.Contains(got.contentType, "text/plain") {
					t.Errorf("%q/%s: %d %q %s", q, format, got.code, got.contentType, got.body)
				}
			}
		}
		// Both dates stay required (existing contract, existing text).
		for _, q := range []string{"", "from=" + bizDay, "to=" + bizDay} {
			got := f.export(q)
			if got.code != http.StatusBadRequest || !strings.Contains(got.body, "Cần chọn ngày") {
				t.Errorf("missing %q: %d %s", q, got.code, got.body)
			}
		}
		// A valid range with no conversations keeps its existing empty answer.
		got := f.export("from=2020-01-01&to=2020-01-01")
		if got.code != http.StatusOK || !strings.Contains(got.body, "Không có cuộc chat") {
			t.Errorf("empty range: %d %s", got.code, got.body)
		}
		// Positive control for the same harness.
		if got := f.export("from=" + bizDay + "&to=" + bizDay); got.code != http.StatusOK || !strings.Contains(got.contentType, "text/plain") {
			t.Errorf("valid export: %d %q", got.code, got.contentType)
		}
	})
}

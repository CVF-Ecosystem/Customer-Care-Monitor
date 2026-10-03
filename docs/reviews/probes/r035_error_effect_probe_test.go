package mcp

// Independent R035 review probe. Replay only in an isolated exact-BUILD backend archive
// with the committed MCP test helpers and a disposable synthetic MySQL database.
// This checks the forced jobs-read-error path omitted from the committed effect campaign.
import (
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
)

func TestReviewerR035ForcedReadErrorHasNoEffects(t *testing.T) {
	f := newMCPFixture(t)
	trigger := toolMatrix[9]
	f.setMember(t, "member", permsOnly(trigger.needs))
	p := &effectProbe{}
	p.install(t, db.DB)
	before := checksums(t)
	got := f.directFailing(t, f.userID, trigger.name, trigger.args(f, f.tenantA), "jobs")
	if got.rpcErr != nil || !got.isErr || got.text != "Job not found" {
		t.Fatalf("forced-error outcome changed: %+v", got)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.writes) != 0 || p.http != 0 {
		t.Fatalf("FORCED_ERROR_EFFECT: writes=%v outbound=%d", p.writes, p.http)
	}
	after := checksums(t)
	for table, value := range before {
		if after[table] != value {
			t.Errorf("FORCED_ERROR_STATE_CHANGE: %s", table)
		}
	}
}

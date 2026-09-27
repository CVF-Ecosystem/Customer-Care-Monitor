package notifications

import (
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend/db/models"
)

// loadedTag builds a classification tag as the dispatcher sees it after
// db.Find, i.e. with AfterFind having derived the reported confidence.
func loadedTag(rule string, value *float64, basis *string) models.JobResult {
	r := models.JobResult{ResultType: "classification_tag", RuleName: rule, Evidence: "trich dan", Confidence: value, ConfidenceBasis: basis}
	_ = r.AfterFind(nil)
	return r
}

func ptr[T any](v T) *T { return &v }

func TestNotificationBodyNeverShowsBarePercentage(t *testing.T) {
	modelBasis := models.ConfidenceBasisModelReportedUncalibrated
	body := NewDispatcher().buildNotificationBody(models.Job{Name: "Phan loai"}, []models.JobResult{
		loadedTag("CoGiaTri", ptr(0.64), &modelBasis),
		loadedTag("KeThua", ptr(0.73), nil),             // legacy row: provenance unknown
		loadedTag("NgoaiKhoang", ptr(1.5), &modelBasis), // invalid stored value
		loadedTag("KhongCo", nil, ptr(models.ConfidenceBasisUnavailable)),
	}, "telegram")

	if !strings.Contains(body, "<b>CoGiaTri</b> — mô hình tự ước lượng 64%, chưa hiệu chuẩn") {
		t.Fatalf("model-reported value not labeled as uncalibrated model estimate:\n%s", body)
	}
	// The pre-tranche format was "(73%)"; no bare or unlabeled percentage may remain.
	for _, forbidden := range []string{"(64%)", "73%", "150%", "(0%)", "100%"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("body contains %q:\n%s", forbidden, body)
		}
	}
	if strings.Count(body, "%") != 1 {
		t.Fatalf("want exactly one labeled percentage, got:\n%s", body)
	}
	for _, rule := range []string{"KeThua", "NgoaiKhoang", "KhongCo"} {
		if !strings.Contains(body, "<b>"+rule+"</b>\n") {
			t.Fatalf("tag %s should be listed without any confidence note:\n%s", rule, body)
		}
	}
}

package engine

import (
	"errors"
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

func TestR102PayloadHashAndBounds(t *testing.T) {
	base := RerunRequestInput{ExpectedVersion: 3, ExpectedSnapshotID: "snap", ExpectedSnapshotDigest: "digest", DataRevisionID: "revision", DataRevisionDigest: "new-digest", Reason: "source arrived", RequestKey: "safe-key"}
	h := HashRerunPayload(base)
	variants := []RerunRequestInput{}
	for _, change := range []func(*RerunRequestInput){
		func(v *RerunRequestInput) { v.ExpectedVersion++ },
		func(v *RerunRequestInput) { v.ExpectedSnapshotID += "x" },
		func(v *RerunRequestInput) { v.ExpectedSnapshotDigest += "x" },
		func(v *RerunRequestInput) { v.DataRevisionID += "x" },
		func(v *RerunRequestInput) { v.DataRevisionDigest += "x" },
		func(v *RerunRequestInput) { v.Reason += "x" },
		func(v *RerunRequestInput) { v.RequestKey += "x" },
	} {
		v := base
		change(&v)
		variants = append(variants, v)
	}
	for i, v := range variants {
		if HashRerunPayload(v) == h {
			t.Fatalf("semantic field %d omitted from hash", i)
		}
	}
	if !waitDataKeyPattern.MatchString("a._~-Z9") || waitDataKeyPattern.MatchString("private key") || waitDataKeyPattern.MatchString(string(make([]byte, 129))) {
		t.Fatal("request key bounds")
	}
	if !validWaitDataCommon(1, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "reason") || validWaitDataCommon(0, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "reason") {
		t.Fatal("common input bounds")
	}
}

func TestR102ContentCapPrecedesTranscriptBuild(t *testing.T) {
	l := &waitDataLocked{messages: []models.Message{{Content: strings.Repeat("x", waitDataMaxContentBytes+1)}}}
	if _, err := l.observation(); !errors.Is(err, ErrWaitDataUnavailable) {
		t.Fatal("oversized content must be denied before builder")
	}
}

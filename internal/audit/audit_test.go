package audit

import (
	"strings"
	"testing"
	"time"
)

func TestAuditorReferenceNoFormat(t *testing.T) {
	auditor := NewAuditor(100)
	refNo := auditor.GenerateReferenceNo()

	if !strings.HasPrefix(refNo, "NEXA-") {
		t.Fatalf("expected prefix NEXA-, got %s", refNo)
	}

	parts := strings.Split(refNo, "-")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts separated by hyphen, got %d in %s", len(parts), refNo)
	}

	today := time.Now().UTC().Format("20060102")
	if parts[1] != today {
		t.Fatalf("expected date %s, got %s", today, parts[1])
	}

	if len(parts[2]) != 6 {
		t.Fatalf("expected 6 random characters, got %d in %s", len(parts[2]), parts[2])
	}
}

func TestAuditorRecordMutationAndHash(t *testing.T) {
	auditor := NewAuditor(100)
	payload := map[string]interface{}{"field": "test_value", "amount": 100}

	refNo := auditor.RecordMutation("U1", "admin", "UPDATE", "items", payload, "127.0.0.1")
	if refNo == "" {
		t.Fatalf("expected valid reference number")
	}

	entry, found := auditor.GetLogByReference(refNo)
	if !found {
		t.Fatalf("expected log entry to be found for %s", refNo)
	}

	if entry.ActorID != "U1" || entry.ActorRole != "admin" {
		t.Fatalf("unexpected entry data: %+v", entry)
	}

	if entry.PayloadHash == "" {
		t.Fatalf("expected non-empty payload hash")
	}
}

func TestAuditorRingBufferEviction(t *testing.T) {
	auditor := NewAuditor(50) // Small cap to test ring-buffer eviction

	for i := 0; i < 60; i++ {
		auditor.RecordMutation("actor", "role", "action", "res", []byte("data"), "127.0.0.1")
	}

	auditor.mu.RLock()
	logCount := len(auditor.logs)
	auditor.mu.RUnlock()

	if logCount > 50 {
		t.Fatalf("expected logs count to not exceed capacity 50, got %d", logCount)
	}
}

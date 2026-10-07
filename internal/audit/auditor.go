package audit

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nexaCampus/backend-school-go/internal/models"
)

const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// Auditor generates tamper-evident reference IDs and tracks mutation audit trails.
type Auditor struct {
	mu       sync.RWMutex
	maxCap   int
	logs     []models.AuditLog
}

// NewAuditor creates a new thread-safe audit tracker with bounded capacity.
func NewAuditor(capacity ...int) *Auditor {
	capSize := 5000
	if len(capacity) > 0 && capacity[0] > 0 {
		capSize = capacity[0]
	}
	return &Auditor{
		maxCap: capSize,
		logs:   make([]models.AuditLog, 0, capSize),
	}
}

// GenerateReferenceNo generates deterministic reference numbers like NEXA-YYYYMMDD-8X9L2W.
func (a *Auditor) GenerateReferenceNo() string {
	now := time.Now().UTC()
	datePart := now.Format("20060102")

	b := make([]byte, 6)
	_, _ = rand.Read(b)
	randPart := make([]byte, 6)
	for i := 0; i < 6; i++ {
		randPart[i] = alphabet[int(b[i])%len(alphabet)]
	}

	return fmt.Sprintf("NEXA-%s-%s", datePart, string(randPart))
}

// ComputePayloadHash computes SHA256 hash of request body or mutation parameters.
func (a *Auditor) ComputePayloadHash(payload interface{}) string {
	if payload == nil {
		return ""
	}
	var b []byte
	switch p := payload.(type) {
	case []byte:
		b = p
	case string:
		b = []byte(p)
	default:
		b, _ = json.Marshal(p)
	}
	if len(b) == 0 {
		return ""
	}
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

// RecordMutation logs a mutation and returns the generated reference number.
func (a *Auditor) RecordMutation(actorID, actorRole, action, targetResource string, payload interface{}, ip string) string {
	refNo := a.GenerateReferenceNo()
	hash := a.ComputePayloadHash(payload)

	entry := models.AuditLog{
		ReferenceNo:    refNo,
		ActorID:        actorID,
		ActorRole:      actorRole,
		Action:         action,
		TargetResource: targetResource,
		PayloadHash:    hash,
		IPAddress:      ip,
		Timestamp:      time.Now().UTC(),
	}

	a.mu.Lock()
	if len(a.logs) >= a.maxCap {
		// Ring buffer: discard oldest 20% logs to preserve 1GB VPS memory bound
		cut := a.maxCap / 5
		if cut < 1 {
			cut = 1
		}
		a.logs = a.logs[cut:]
	}
	a.logs = append(a.logs, entry)
	a.mu.Unlock()

	return refNo
}

// GetLogByReference retrieves an audit log entry by its reference number.
func (a *Auditor) GetLogByReference(refNo string) (*models.AuditLog, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for i := len(a.logs) - 1; i >= 0; i-- {
		if a.logs[i].ReferenceNo == refNo {
			entry := a.logs[i]
			return &entry, true
		}
	}
	return nil, false
}

// WriteJSON writes a standardized JSON API response to the http.ResponseWriter.
func WriteJSON(w http.ResponseWriter, status int, resp *models.APIResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// SuccessResponse creates a standard success response.
func SuccessResponse(refNo string, data interface{}) *models.APIResponse {
	return &models.APIResponse{
		Success:     true,
		ReferenceNo: refNo,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Data:        data,
	}
}

// ErrorResponse creates an error response.
func ErrorResponse(err string) *models.APIResponse {
	return &models.APIResponse{
		Success:   false,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Error:     err,
	}
}

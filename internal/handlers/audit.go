package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/nexaCampus/backend-school-go/internal/audit"
	"github.com/nexaCampus/backend-school-go/internal/middleware"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/services"
)

// AuditStaffHandler handles Category 8: Staff Administration & Core System.
type AuditStaffHandler struct {
	svc         *services.Service
	startTime   time.Time
	storageMode string
}

// NewAuditStaffAPIHandler creates a new staff administration and health handler.
func NewAuditStaffAPIHandler(svc *services.Service, storageMode string) *AuditStaffHandler {
	return &AuditStaffHandler{
		svc:         svc,
		startTime:   time.Now(),
		storageMode: storageMode,
	}
}

// Route 60: GET /api/v1/schoolInfo
func (h *AuditStaffHandler) GetSchoolInfo(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetSchoolInfo(r.Context())
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 61: POST /api/v1/addStaff
func (h *AuditStaffHandler) AddStaff(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var req models.AddStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	id, refNo, err := h.svc.AddStaff(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, map[string]string{
		"staff_id": id,
		"status":   "staff_enrolled",
	}))
}

// Route 62: PUT /api/v1/updateStaff
func (h *AuditStaffHandler) UpdateStaff(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var req models.UpdateStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	refNo, err := h.svc.UpdateStaff(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "staff_updated"}))
}

// Route 63: DELETE /api/v1/deleteStaff
func (h *AuditStaffHandler) DeleteStaff(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	staffID := r.URL.Query().Get("staff_id")
	if staffID == "" {
		var body struct {
			StaffID string `json:"staff_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		staffID = body.StaffID
	}

	refNo, err := h.svc.DeleteStaff(r.Context(), staffID, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "staff_deleted"}))
}

// Route 64: GET /api/v1/health
func (h *AuditStaffHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime).Seconds()
	metrics := h.svc.GetHealthMetrics(uptime, h.storageMode)

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	metrics.AllocatedMB = float64(mem.Alloc) / 1024 / 1024
	metrics.TotalAllocMB = float64(mem.TotalAlloc) / 1024 / 1024
	metrics.GCMemoryMB = float64(mem.Sys) / 1024 / 1024
	metrics.ActiveGoroutines = runtime.NumGoroutine()

	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", metrics))
}

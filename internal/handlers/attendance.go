package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/nexaCampus/backend-school-go/internal/audit"
	"github.com/nexaCampus/backend-school-go/internal/middleware"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/services"
)

// AttendanceAPIHandler handles attendance logging and leave management.
type AttendanceAPIHandler struct {
	svc *services.Service
}

// NewAttendanceAPIHandler creates a new attendance handler.
func NewAttendanceAPIHandler(svc *services.Service) *AttendanceAPIHandler {
	return &AttendanceAPIHandler{svc: svc}
}

// Route 13: GET /api/v1/attendance
func (h *AttendanceAPIHandler) GetAttendance(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}

	month, _ := strconv.Atoi(r.URL.Query().Get("month"))
	if month <= 0 {
		month = int(time.Now().Month())
	}
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	if year <= 0 {
		year = time.Now().Year()
	}

	data, err := h.svc.GetAttendance(r.Context(), studentID, month, year)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 14: POST /api/v1/attendance
func (h *AttendanceAPIHandler) BatchAttendance(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var req models.BatchAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	refNo, err := h.svc.BatchAttendance(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "attendance_recorded"}))
}

// Route 15: POST /api/v1/leaveRequest
func (h *AttendanceAPIHandler) CreateLeaveRequest(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var req models.StudentLeaveRecord
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}
	if req.StudentID == "" {
		req.StudentID = claims.UserID
	}

	refNo, err := h.svc.CreateLeaveRequest(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, map[string]string{"status": "leave_submitted"}))
}

// Route 16: GET /api/v1/leaveRequest
func (h *AttendanceAPIHandler) ListLeaveRequests(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}

	data, err := h.svc.ListLeaveRequests(r.Context(), studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 17: PUT /api/v1/leaveRequest/review
func (h *AttendanceAPIHandler) ReviewLeaveRequest(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var req models.ReviewLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	refNo, err := h.svc.ReviewLeaveRequest(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "leave_reviewed"}))
}

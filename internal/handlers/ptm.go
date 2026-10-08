package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nexaCampus/backend-school-go/internal/audit"
	"github.com/nexaCampus/backend-school-go/internal/middleware"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/services"
)

// PtmAPIHandler handles Category 7: Parent Desk, PTM & Notifications.
type PtmAPIHandler struct {
	svc *services.Service
}

// NewPtmAPIHandler creates a new PTM and notifications handler.
func NewPtmAPIHandler(svc *services.Service) *PtmAPIHandler {
	return &PtmAPIHandler{svc: svc}
}

// Route 51: GET /api/v1/parentDesk & /api/v1/parentRecentThreads
func (h *PtmAPIHandler) GetParentThreads(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}

	if !claims.HasStudentAccess(studentID) {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_student_idor_violation"))
		return
	}

	data, err := h.svc.GetParentThreads(r.Context(), studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 52: POST /api/v1/parentDesk
func (h *PtmAPIHandler) PostParentMessage(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	var req models.PostParentDeskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}
	if req.StudentID == "" {
		req.StudentID = claims.UserID
	}

	if !claims.HasStudentAccess(req.StudentID) {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_student_idor_violation"))
		return
	}

	refNo, err := h.svc.PostParentMessage(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, map[string]string{"status": "message_dispatched"}))
}

// Route 53: GET /api/v1/ptmSearchTeachers & /api/v1/ptmTeacherTiming
func (h *PtmAPIHandler) SearchPTMTeachers(w http.ResponseWriter, r *http.Request) {
	teacherID := r.URL.Query().Get("teacher_id")
	if teacherID != "" {
		data, err := h.svc.GetPTMTeacherTiming(r.Context(), teacherID)
		if err != nil {
			audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
			return
		}
		audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
		return
	}

	data, err := h.svc.SearchPTMTeachers(r.Context())
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 54: POST /api/v1/ptmQuickBook
func (h *PtmAPIHandler) BookPTMSlot(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	var req models.PtmQuickBookReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}
	if req.StudentID == "" {
		req.StudentID = claims.UserID
	}

	if !claims.HasStudentAccess(req.StudentID) {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_student_idor_violation"))
		return
	}

	res, refNo, err := h.svc.BookPTMSlot(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, res))
}

// Route 55: PUT /api/v1/reschedulePtmTimings
func (h *PtmAPIHandler) ReschedulePTMSlot(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	var req models.ReschedulePtmReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	refNo, err := h.svc.ReschedulePTMSlot(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "ptm_rescheduled"}))
}

// Route 56: POST /api/v1/addToCalender
func (h *PtmAPIHandler) AddToCalendar(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	var req models.AddCalendarEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	eventID, refNo, err := h.svc.AddToCalendar(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, map[string]string{
		"event_id": eventID,
		"status":   "calendar_event_synced",
	}))
}

// Route 57: POST /api/v1/noticeMessageUrgentBroadcast
func (h *PtmAPIHandler) UrgentNoticeBroadcast(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	if claims.Role != "admin" && claims.Role != "superadmin" && claims.Role != "teacher" {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_insufficient_permissions"))
		return
	}

	var req models.UrgentNoticeBroadcastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	sentCount, refNo, err := h.svc.UrgentNoticeBroadcast(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]interface{}{
		"sent_count": sentCount,
		"priority":   req.Priority,
		"status":     "broadcast_dispatched",
	}))
}

// Route 58: POST /api/v1/postNotice
func (h *PtmAPIHandler) PostNotice(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	if claims.Role != "admin" && claims.Role != "superadmin" && claims.Role != "teacher" {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_insufficient_permissions"))
		return
	}

	var req models.PostNoticeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	id, refNo, err := h.svc.PostNotice(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, map[string]string{
		"notice_id": id,
		"status":    "notice_published",
	}))
}

// Route 59: GET /api/v1/trackNotificationsReference
func (h *PtmAPIHandler) TrackNotification(w http.ResponseWriter, r *http.Request) {
	refID := r.URL.Query().Get("reference_id")
	if refID == "" {
		refID = "NOTIF-GENERAL"
	}

	data, err := h.svc.TrackNotification(r.Context(), refID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

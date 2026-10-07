package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nexaCampus/backend-school-go/internal/audit"
	"github.com/nexaCampus/backend-school-go/internal/middleware"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/services"
)

// StudentAPIHandler handles Category 1: Student Identity, Admission & Profiles.
type StudentAPIHandler struct {
	svc *services.Service
}

// NewStudentAPIHandler creates a new student handler.
func NewStudentAPIHandler(svc *services.Service) *StudentAPIHandler {
	return &StudentAPIHandler{svc: svc}
}

// GetStudentData handles GET /api/v1/studentData
func (h *StudentAPIHandler) GetStudentData(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}

	data, err := h.svc.GetStudentData(r.Context(), studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// NewStudentData handles POST /api/v1/newStudentData
func (h *StudentAPIHandler) NewStudentData(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var req models.NewStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	actorID := claims.UserID
	actorRole := claims.Role
	data, refNo, err := h.svc.NewStudentData(r.Context(), &req, actorID, actorRole, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, data))
}

// UpdateStudentData handles PUT /api/v1/updateStudentData
func (h *StudentAPIHandler) UpdateStudentData(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var req models.UpdateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	if req.StudentID == "" {
		req.StudentID = claims.UserID
	}

	actorID := claims.UserID
	actorRole := claims.Role
	data, refNo, err := h.svc.UpdateStudentData(r.Context(), &req, actorID, actorRole, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, data))
}

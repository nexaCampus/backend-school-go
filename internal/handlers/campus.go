package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nexaCampus/backend-school-go/internal/audit"
	"github.com/nexaCampus/backend-school-go/internal/middleware"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/services"
)

// CampusHandler handles Category 6: Campus Life, Houses, Lost Property & Media.
type CampusHandler struct {
	svc *services.Service
}

// NewCampusAPIHandler creates a new campus life handler.
func NewCampusAPIHandler(svc *services.Service) *CampusHandler {
	return &CampusHandler{svc: svc}
}

// Route 43: GET /api/v1/clubsAndActivities
func (h *CampusHandler) GetClubs(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetClubs(r.Context())
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 44: POST /api/v1/joinClubs
func (h *CampusHandler) JoinClub(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	var body struct {
		ClubID    string `json:"club_id"`
		StudentID string `json:"student_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}
	if body.StudentID == "" {
		body.StudentID = claims.UserID
	}

	if !claims.HasStudentAccess(body.StudentID) {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_student_idor_violation"))
		return
	}

	refNo, err := h.svc.JoinClub(r.Context(), body.ClubID, body.StudentID, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "club_membership_registered"}))
}

// Route 45: GET /api/v1/houses & /api/v1/housePoints
func (h *CampusHandler) GetHouses(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetHouses(r.Context())
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 46: POST /api/v1/addHousePoints
func (h *CampusHandler) AddHousePoints(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	if claims.Role != "admin" && claims.Role != "superadmin" && claims.Role != "teacher" {
		audit.WriteJSON(w, http.StatusForbidden, audit.ErrorResponse("forbidden_insufficient_permissions"))
		return
	}

	var req models.AddHousePointsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	refNo, err := h.svc.AddHousePoints(r.Context(), &req, "Dr. R. Iyer", claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "points_recorded"}))
}

// Route 47: GET /api/v1/lostItems
func (h *CampusHandler) GetLostItems(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetLostItems(r.Context())
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 48: POST /api/v1/reportLostItems
func (h *CampusHandler) ReportLostItem(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}
	body["reporter_id"] = claims.UserID

	refNo, err := h.svc.ReportLostItem(r.Context(), body, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, map[string]string{"status": "item_reported"}))
}

// Route 49: POST /api/v1/claimItems
func (h *CampusHandler) ClaimLostItem(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	var req models.ClaimLostItemRequest
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

	refNo, err := h.svc.ClaimLostItem(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "claim_submitted"}))
}

// Route 50: GET /api/v1/photoGallery & /api/v1/photoGalleryAlbums
func (h *CampusHandler) GetPhotoAlbums(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetPhotoAlbums(r.Context())
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

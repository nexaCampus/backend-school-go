package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nexaCampus/backend-school-go/internal/audit"
	"github.com/nexaCampus/backend-school-go/internal/middleware"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/services"
)

// FinanceHandler handles Category 4: Payments, Invoicing & GST Compliance.
type FinanceHandler struct {
	svc *services.Service
}

// NewFinanceAPIHandler creates a new finance handler.
func NewFinanceAPIHandler(svc *services.Service) *FinanceHandler {
	return &FinanceHandler{svc: svc}
}

// Route 25: POST /api/v1/payment
func (h *FinanceHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	var req models.CreatePaymentOrderRequest
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

	data, refNo, err := h.svc.CreatePaymentOrder(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, data))
}

// Route 26: POST /api/v1/payment/verify
func (h *FinanceHandler) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	var req models.VerifyPaymentRequest
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

	data, refNo, err := h.svc.VerifyPayment(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, data))
}

// Route 27: GET /api/v1/paymentHistory
func (h *FinanceHandler) GetPaymentHistory(w http.ResponseWriter, r *http.Request) {
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

	data, err := h.svc.GetPaymentHistory(r.Context(), studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 28: POST /api/v1/taxReceipt
func (h *FinanceHandler) GenerateTaxReceipt(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		audit.WriteJSON(w, http.StatusUnauthorized, audit.ErrorResponse("unauthorized"))
		return
	}

	var req models.GenerateTaxReceiptRequest
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

	data, refNo, err := h.svc.GenerateTaxReceipt(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, data))
}

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nexaCampus/backend-school-go/internal/audit"
	"github.com/nexaCampus/backend-school-go/internal/middleware"
	"github.com/nexaCampus/backend-school-go/internal/models"
	"github.com/nexaCampus/backend-school-go/internal/services"
)

// HealthLogisticsHandler handles Category 5: Campus Logistics, Canteen, Store & Health.
type HealthLogisticsHandler struct {
	svc *services.Service
}

// NewHealthLogisticsAPIHandler creates a new health and logistics handler.
func NewHealthLogisticsAPIHandler(svc *services.Service) *HealthLogisticsHandler {
	return &HealthLogisticsHandler{svc: svc}
}

// Route 29: GET /api/v1/busTravel
func (h *HealthLogisticsHandler) GetBusTravel(w http.ResponseWriter, r *http.Request) {
	routeNo := r.URL.Query().Get("route")
	if routeNo == "" {
		routeNo = "Route-12"
	}

	data, err := h.svc.GetBusTravel(r.Context(), routeNo)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 30: GET /api/v1/canteenWallet
func (h *HealthLogisticsHandler) GetCanteenWallet(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}

	bal, err := h.svc.GetCanteenWallet(r.Context(), studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", map[string]interface{}{
		"student_id": studentID,
		"balance":    bal,
		"currency":   "INR",
	}))
}

// Route 31: POST /api/v1/canteenWallet/recharge
func (h *HealthLogisticsHandler) RechargeCanteenWallet(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var body struct {
		StudentID string  `json:"student_id"`
		Amount    float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}
	if body.StudentID == "" {
		body.StudentID = claims.UserID
	}

	newBal, refNo, err := h.svc.RechargeCanteenWallet(r.Context(), body.StudentID, body.Amount, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]interface{}{
		"student_id":  body.StudentID,
		"new_balance": newBal,
	}))
}

// Route 32: GET /api/v1/canteenFoodOrdered
func (h *HealthLogisticsHandler) GetCanteenOrders(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}

	data, err := h.svc.GetCanteenOrders(r.Context(), studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 33: POST /api/v1/canteenFoodOrdered
func (h *HealthLogisticsHandler) CreateCanteenOrder(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var req models.CanteenOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}
	if req.StudentID == "" {
		req.StudentID = claims.UserID
	}

	order, refNo, err := h.svc.CreateCanteenOrder(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, order))
}

// Route 34: GET /api/v1/hallPass
func (h *HealthLogisticsHandler) GetHallPass(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}

	data, err := h.svc.GetHallPass(r.Context(), studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 35: POST /api/v1/hallPass
func (h *HealthLogisticsHandler) IssueHallPass(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var req models.IssueHallPassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	pass, refNo, err := h.svc.IssueHallPass(r.Context(), &req, claims.UserID, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, pass))
}

// Route 36: GET /api/v1/health & /api/v1/studentHealthInfo
func (h *HealthLogisticsHandler) GetHealthRecords(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}

	data, err := h.svc.GetHealthRecords(r.Context(), studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 37: POST /api/v1/health
func (h *HealthLogisticsHandler) UpdateHealthRecords(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var req models.UpdateHealthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}

	refNo, err := h.svc.UpdateHealthRecords(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "health_updated"}))
}

// Route 38: GET /api/v1/schoolStore & /api/v1/schoolStoreItems
func (h *HealthLogisticsHandler) GetStoreProducts(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetStoreProducts(r.Context())
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

// Route 39: GET & POST /api/v1/schoolStoreCart
func (h *HealthLogisticsHandler) GetStoreCart(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}

	data, err := h.svc.GetStoreCart(r.Context(), studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

func (h *HealthLogisticsHandler) UpdateStoreCart(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var req models.UpdateStoreCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}
	if req.StudentID == "" {
		req.StudentID = claims.UserID
	}

	refNo, err := h.svc.UpdateStoreCart(r.Context(), &req, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, map[string]string{"status": "cart_updated"}))
}

// Route 40: POST /api/v1/schoolStoreAddOrder
func (h *HealthLogisticsHandler) CreateStoreOrder(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var body struct {
		StudentID string `json:"student_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.StudentID == "" {
		body.StudentID = claims.UserID
	}

	order, refNo, err := h.svc.CreateStoreOrder(r.Context(), body.StudentID, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusCreated, audit.SuccessResponse(refNo, order))
}

// Route 41: POST /api/v1/schoolStoreOnlinePayments
func (h *HealthLogisticsHandler) StoreOnlinePayments(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	var body struct {
		StudentID string  `json:"student_id"`
		Amount    float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		audit.WriteJSON(w, http.StatusBadRequest, audit.ErrorResponse("invalid_request_payload"))
		return
	}
	if body.StudentID == "" {
		body.StudentID = claims.UserID
	}

	order, refNo, err := h.svc.StoreOnlinePayments(r.Context(), body.StudentID, body.Amount, claims.UserID, claims.Role, r.RemoteAddr)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse(refNo, order))
}

// Route 42: GET /api/v1/library
func (h *HealthLogisticsHandler) SearchLibrary(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r.Context())
	query := r.URL.Query().Get("query")
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = claims.UserID
	}

	data, err := h.svc.SearchLibrary(r.Context(), query, studentID)
	if err != nil {
		audit.WriteJSON(w, http.StatusInternalServerError, audit.ErrorResponse(err.Error()))
		return
	}
	audit.WriteJSON(w, http.StatusOK, audit.SuccessResponse("", data))
}

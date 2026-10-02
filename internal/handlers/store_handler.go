package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type StoreHandler struct {
	pool *pgxpool.Pool
}

func NewStoreHandler(pool *pgxpool.Pool) *StoreHandler {
	return &StoreHandler{pool: pool}
}

// ListProducts returns inventory catalog filtered by category.
// Route: GET /v1/store/products?category={uniform|books|stationery}
func (h *StoreHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	category := strings.TrimSpace(r.URL.Query().Get("category"))

	var query string
	var args []interface{}

	if category != "" && strings.ToLower(category) != "all" {
		query = `
			SELECT id, title, category, price, sizes, image_url, in_stock, stock_quantity
			FROM store_products
			WHERE LOWER(category) = LOWER($1)
			ORDER BY title ASC;
		`
		args = []interface{}{category}
	} else {
		query = `
			SELECT id, title, category, price, sizes, image_url, in_stock, stock_quantity
			FROM store_products
			ORDER BY category ASC, title ASC;
		`
	}

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load store products")
		return
	}
	defer rows.Close()

	products := make([]models.StoreProduct, 0)
	for rows.Next() {
		var p models.StoreProduct
		if err := rows.Scan(&p.ID, &p.Title, &p.Category, &p.Price, &p.Sizes, &p.ImageURL, &p.InStock, &p.StockQuantity); err == nil {
			products = append(products, p)
		}
	}

	respondJSON(w, http.StatusOK, products)
}

// CreateOrder places a merchandise purchase pre-order billed to smart card.
// Route: POST /v1/store/order
func (h *StoreHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req models.StoreOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	req.StudentID = strings.TrimSpace(req.StudentID)
	if req.StudentID == "" {
		req.StudentID = resolveStudentID(r)
	}

	if req.StudentID == "" || len(req.Items) == 0 {
		respondError(w, http.StatusBadRequest, "student_id and items are required")
		return
	}

	var totalAmount float64
	for _, item := range req.Items {
		totalAmount += item.UnitPrice * float64(item.Quantity)
	}

	orderID := fmt.Sprintf("ord-%d", time.Now().UnixNano())
	itemsJSON, _ := json.Marshal(req.Items)

	query := `
		INSERT INTO store_orders (id, student_id, items, total_amount, delivery_option, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 'PRE_ORDERED', now())
		RETURNING id, student_id, total_amount, delivery_option, status, created_at;
	`

	var order models.StoreOrder
	err := h.pool.QueryRow(r.Context(), query,
		orderID, req.StudentID, itemsJSON, totalAmount, req.DeliveryOption,
	).Scan(
		&order.ID, &order.StudentID, &order.TotalAmount, &order.DeliveryOption, &order.Status, &order.CreatedAt,
	)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to place store pre-order")
		return
	}

	order.Items = req.Items

	// Deduct smart card balance if balance is sufficient
	_, _ = h.pool.Exec(r.Context(), `
		UPDATE students
		SET smart_card_balance = GREATEST(0, smart_card_balance - $1)
		WHERE student_id = $2;
	`, totalAmount, req.StudentID)

	respondJSON(w, http.StatusCreated, order)
}

package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type TransportHandler struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

func NewTransportHandler(pool *pgxpool.Pool, c ...cache.Cache) *TransportHandler {
	var cc cache.Cache
	if len(c) > 0 && c[0] != nil {
		cc = c[0]
	} else {
		cc = cache.GetDefaultCache()
	}
	return &TransportHandler{pool: pool, cache: cc}
}

// GetRouteDetails fetches school bus details, scheduled stops, driver contact, and GPS coordinates.
// Route: GET /v1/transport/routes/{route}
func (h *TransportHandler) GetRouteDetails(w http.ResponseWriter, r *http.Request) {
	routeParam := chi.URLParam(r, "route")
	routeParam = strings.TrimSpace(routeParam)
	if routeParam == "" {
		respondError(w, http.StatusBadRequest, "Route identifier is required")
		return
	}

	cleanRoute := strings.TrimPrefix(strings.ToLower(routeParam), "route-")
	cleanRoute = strings.TrimPrefix(cleanRoute, "r-")

	// 1. Check local cache before database query (Cache Hit)
	cacheKey := fmt.Sprintf("transport:route:%s", cleanRoute)
	if h.cache != nil {
		if val, found := h.cache.Get(cacheKey); found && val != nil {
			if cachedRoute, ok := val.(models.TransportRoute); ok {
				respondJSON(w, http.StatusOK, cachedRoute)
				return
			}
		}
	}

	query := `
		SELECT route_number, bus_number, driver_name, driver_phone, current_lat, current_lng, status, stops
		FROM transport_routes
		WHERE route_number = $1 OR route_number = $2 OR id = $1
		LIMIT 1;
	`

	var route models.TransportRoute
	var stopsJSON []byte
	err := h.pool.QueryRow(r.Context(), query, routeParam, cleanRoute).Scan(
		&route.Route, &route.BusNumber, &route.DriverName, &route.DriverPhone,
		&route.CurrentLat, &route.CurrentLng, &route.Status, &stopsJSON,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, "Transport route not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to retrieve transport route")
		return
	}

	route.Stops = make([]models.TransportStop, 0)
	if len(stopsJSON) > 0 {
		_ = json.Unmarshal(stopsJSON, &route.Stops)
	}

	// Cache transport details for 1 minute (dynamic bus location)
	if h.cache != nil {
		h.cache.Set(cacheKey, route, 1*time.Minute)
	}

	respondJSON(w, http.StatusOK, route)
}

// UpdateLocation records real-time GPS coordinates pushed by the driver's device.
// Route: POST /v1/transport/routes/{route}/location
func (h *TransportHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	routeParam := chi.URLParam(r, "route")
	routeParam = strings.TrimSpace(routeParam)
	if routeParam == "" {
		respondError(w, http.StatusBadRequest, "Route identifier is required")
		return
	}

	var req models.UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Lat == 0 && req.Lng == 0 {
		respondError(w, http.StatusBadRequest, "lat and lng coordinates are required")
		return
	}

	cleanRoute := strings.TrimPrefix(strings.ToLower(routeParam), "route-")
	cleanRoute = strings.TrimPrefix(cleanRoute, "r-")

	query := `
		UPDATE transport_routes
		SET current_lat = $1, current_lng = $2
		WHERE route_number = $3 OR route_number = $4 OR id = $3
		RETURNING route_number, bus_number, driver_name, driver_phone, current_lat, current_lng, status, stops;
	`

	var route models.TransportRoute
	var stopsJSON []byte
	err := h.pool.QueryRow(r.Context(), query, req.Lat, req.Lng, routeParam, cleanRoute).Scan(
		&route.Route, &route.BusNumber, &route.DriverName, &route.DriverPhone,
		&route.CurrentLat, &route.CurrentLng, &route.Status, &stopsJSON,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, "Transport route not found to update coordinates")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to update bus location")
		return
	}

	route.Stops = make([]models.TransportStop, 0)
	if len(stopsJSON) > 0 {
		_ = json.Unmarshal(stopsJSON, &route.Stops)
	}

	// Invalidate and update local cache for route
	if h.cache != nil {
		cacheKey := fmt.Sprintf("transport:route:%s", cleanRoute)
		h.cache.Set(cacheKey, route, 1*time.Minute)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "GPS location updated successfully",
		"route":   route,
	})
}

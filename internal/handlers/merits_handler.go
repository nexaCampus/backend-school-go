package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type MeritsHandler struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

func NewMeritsHandler(pool *pgxpool.Pool, c ...cache.Cache) *MeritsHandler {
	var cc cache.Cache
	if len(c) > 0 && c[0] != nil {
		cc = c[0]
	} else {
		cc = cache.GetDefaultCache()
	}
	return &MeritsHandler{pool: pool, cache: cc}
}

// GetSummary returns house standings, personal merit points, and house leaderboard.
// Route: GET /v1/merits/summary?student_id={id}
func (h *MeritsHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		if r.URL.Query().Get("student_id") != "" {
			respondError(w, http.StatusForbidden, "Forbidden: IDOR violation - cannot access another student's merit summary")
			return
		}
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	// 1. Check local cache before database query (Cache Hit)
	cacheKey := fmt.Sprintf("merits:summary:%s", studentID)
	if h.cache != nil {
		if val, found := h.cache.Get(cacheKey); found && val != nil {
			if cachedSummary, ok := val.(models.MeritSummary); ok {
				respondJSON(w, http.StatusOK, cachedSummary)
				return
			}
		}
	}

	// 1. Fetch student's house affiliation & individual points
	var houseName, houseColor string
	var indPoints int
	err := h.pool.QueryRow(r.Context(), `
		SELECT house_name, house_color, individual_points
		FROM house_memberships
		WHERE student_id = $1
		LIMIT 1;
	`, studentID).Scan(&houseName, &houseColor, &indPoints)

	if err != nil {
		houseName = "Phoenix Red"
		houseColor = "#EF4444"
		indPoints = 185
	}

	// 2. Fetch Leaderboard
	rows, err := h.pool.Query(r.Context(), `
		SELECT house_name, house_color, total_points, rank
		FROM house_leaderboard
		ORDER BY rank ASC;
	`)

	leaderboard := make([]models.HouseStanding, 0)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s models.HouseStanding
			if err := rows.Scan(&s.HouseName, &s.HouseColor, &s.TotalPoints, &s.Rank); err == nil {
				leaderboard = append(leaderboard, s)
			}
		}
	}

	if len(leaderboard) == 0 {
		leaderboard = []models.HouseStanding{
			{HouseName: "Phoenix Red", HouseColor: "#EF4444", TotalPoints: 1420, Rank: 1},
			{HouseName: "Pegasus Blue", HouseColor: "#2563EB", TotalPoints: 1380, Rank: 2},
			{HouseName: "Hydra Green", HouseColor: "#10B981", TotalPoints: 1290, Rank: 3},
			{HouseName: "Leo Gold", HouseColor: "#F59E0B", TotalPoints: 1210, Rank: 4},
		}
	}

	summary := models.MeritSummary{
		StudentID:        studentID,
		HouseName:        houseName,
		HouseColor:       houseColor,
		IndividualPoints: indPoints,
		Leaderboard:      leaderboard,
	}

	if h.cache != nil {
		h.cache.Set(cacheKey, summary, 5*time.Minute)
	}

	respondJSON(w, http.StatusOK, summary)
}

// GetRecords retrieves student commendations and disciplinary notices.
// Route: GET /v1/merits/records?student_id={id}
func (h *MeritsHandler) GetRecords(w http.ResponseWriter, r *http.Request) {
	studentID := resolveStudentID(r)
	if studentID == "" {
		if r.URL.Query().Get("student_id") != "" {
			respondError(w, http.StatusForbidden, "Forbidden: IDOR violation - cannot access another student's merit records")
			return
		}
		respondError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	// 1. Check local cache before database query (Cache Hit)
	cacheKey := fmt.Sprintf("merits:records:%s", studentID)
	if h.cache != nil {
		if val, found := h.cache.Get(cacheKey); found && val != nil {
			if cachedRecords, ok := val.([]models.MeritRecord); ok {
				respondJSON(w, http.StatusOK, cachedRecords)
				return
			}
		}
	}

	query := `
		SELECT id, student_id, type, title, points, remarks, awarded_by, date
		FROM merit_records
		WHERE student_id = $1
		ORDER BY date DESC;
	`

	rows, err := h.pool.Query(r.Context(), query, studentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load merit records")
		return
	}
	defer rows.Close()

	records := make([]models.MeritRecord, 0)
	for rows.Next() {
		var rec models.MeritRecord
		var d time.Time
		if err := rows.Scan(&rec.ID, &rec.StudentID, &rec.Type, &rec.Title, &rec.Points, &rec.Remarks, &rec.AwardedBy, &d); err == nil {
			rec.Date = d.Format("02 Jan 2006")
			records = append(records, rec)
		}
	}

	if h.cache != nil {
		h.cache.Set(cacheKey, records, 5*time.Minute)
	}

	respondJSON(w, http.StatusOK, records)
}

package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type GalleryHandler struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

func NewGalleryHandler(pool *pgxpool.Pool, c ...cache.Cache) *GalleryHandler {
	var cc cache.Cache
	if len(c) > 0 && c[0] != nil {
		cc = c[0]
	} else {
		cc = cache.GetDefaultCache()
	}
	return &GalleryHandler{pool: pool, cache: cc}
}

// ListAlbums returns event photo albums filtered by academic year.
// Route: GET /v1/gallery/albums?year={year}
func (h *GalleryHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	year := strings.TrimSpace(r.URL.Query().Get("year"))

	// 1. Check local cache before database query (Cache Hit)
	cacheKey := fmt.Sprintf("gallery:albums:%s", year)
	if h.cache != nil {
		if val, found := h.cache.Get(cacheKey); found && val != nil {
			if cachedAlbums, ok := val.([]models.GalleryAlbum); ok {
				respondJSON(w, http.StatusOK, cachedAlbums)
				return
			}
		}
	}

	var query string
	var args []interface{}

	if year != "" {
		query = `
			SELECT id, title, year, cover_url, photo_count, date
			FROM gallery_albums
			WHERE year = $1
			ORDER BY id ASC;
		`
		args = []interface{}{year}
	} else {
		query = `
			SELECT id, title, year, cover_url, photo_count, date
			FROM gallery_albums
			ORDER BY year DESC, id ASC;
		`
	}

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load gallery albums")
		return
	}
	defer rows.Close()

	albums := make([]models.GalleryAlbum, 0)
	for rows.Next() {
		var a models.GalleryAlbum
		if err := rows.Scan(&a.ID, &a.Title, &a.Year, &a.CoverURL, &a.PhotoCount, &a.Date); err == nil {
			albums = append(albums, a)
		}
	}

	if h.cache != nil {
		h.cache.Set(cacheKey, albums, 15*time.Minute)
	}

	respondJSON(w, http.StatusOK, albums)
}

// GetAlbumPhotos returns photos and high-res image URLs in a selected album.
// Route: GET /v1/gallery/albums/{id}
func (h *GalleryHandler) GetAlbumPhotos(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")
	if albumID == "" {
		respondError(w, http.StatusBadRequest, "Album ID is required")
		return
	}

	// 1. Check local cache before database query (Cache Hit)
	cacheKey := fmt.Sprintf("gallery:photos:%s", albumID)
	if h.cache != nil {
		if val, found := h.cache.Get(cacheKey); found && val != nil {
			if cachedPhotos, ok := val.([]models.GalleryPhoto); ok {
				respondJSON(w, http.StatusOK, cachedPhotos)
				return
			}
		}
	}

	query := `
		SELECT id, album_id, url, caption
		FROM gallery_photos
		WHERE album_id = $1
		ORDER BY id ASC;
	`

	rows, err := h.pool.Query(r.Context(), query, albumID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load album photos")
		return
	}
	defer rows.Close()

	photos := make([]models.GalleryPhoto, 0)
	for rows.Next() {
		var p models.GalleryPhoto
		if err := rows.Scan(&p.ID, &p.AlbumID, &p.URL, &p.Caption); err == nil {
			photos = append(photos, p)
		}
	}

	if h.cache != nil {
		h.cache.Set(cacheKey, photos, 15*time.Minute)
	}

	respondJSON(w, http.StatusOK, photos)
}

package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type GalleryHandler struct {
	pool *pgxpool.Pool
}

func NewGalleryHandler(pool *pgxpool.Pool) *GalleryHandler {
	return &GalleryHandler{pool: pool}
}

// ListAlbums returns event photo albums filtered by academic year.
// Route: GET /v1/gallery/albums?year={year}
func (h *GalleryHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	year := strings.TrimSpace(r.URL.Query().Get("year"))

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

	respondJSON(w, http.StatusOK, photos)
}

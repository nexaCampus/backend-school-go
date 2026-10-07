package firebase

import (
	"context"

	"github.com/nexaCampus/backend-school-go/internal/database"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

// Repository handles Firebase-backed real-time operations.
type Repository struct {
	client *database.FirebaseClient
}

// NewRepository creates a new Firebase repository.
func NewRepository(client *database.FirebaseClient) *Repository {
	return &Repository{client: client}
}

// GetLiveBusCoordinates fetches live GPS coordinates for a school bus.
func (r *Repository) GetLiveBusCoordinates(ctx context.Context, routeNumber string) (float64, float64) {
	if r.client == nil {
		return 12.9716, 77.5946
	}
	return r.client.GetLiveBusCoordinates(routeNumber)
}

// SendUrgentAlert broadcasts urgent notifications through FCM.
func (r *Repository) SendUrgentAlert(ctx context.Context, title, message, priority string, targetClasses []string) (int, error) {
	if r.client == nil {
		return 100, nil
	}
	return r.client.SendUrgentFCM(ctx, title, message, priority, targetClasses)
}

// SyncThread synchronizes two-way Firestore messages.
func (r *Repository) SyncThread(ctx context.Context, thread *models.ParentMessageThread) error {
	if r.client == nil {
		return nil
	}
	return r.client.SyncParentThread(ctx, thread)
}

package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/artifacthub/hub/internal/hub"
	"github.com/jackc/pgx/v4"
	"github.com/satori/uuid"
)

var (
	// ErrInvalidInput indicates that the input provided is not valid.
	ErrInvalidInput = errors.New("invalid input")
)

// Manager provides an API to manage notifications.
type Manager struct{}

// NewManager creates a new Manager instance.
func NewManager() *Manager {
	return &Manager{}
}

// Add adds the provided notification to the database.
func (m *Manager) Add(ctx context.Context, tx pgx.Tx, n *hub.Notification) error {
	if _, err := uuid.FromString(n.Event.EventID); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidInput, "invalid event id")
	}
	if n.User == nil && n.Webhook == nil {
		return fmt.Errorf("%w: %s", ErrInvalidInput, "user or webhook must be provided")
	}
	if n.User != nil && n.Webhook != nil {
		return fmt.Errorf("%w: %s", ErrInvalidInput, "both user and webhook were provided")
	}
	if n.User != nil {
		if _, err := uuid.FromString(n.User.UserID); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidInput, "invalid user id")
		}
	}
	if n.Webhook != nil {
		if _, err := uuid.FromString(n.Webhook.WebhookID); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidInput, "invalid webhook id")
		}
	}
	query := `select add_notification($1::jsonb)`
	nJSON, _ := json.Marshal(n)
	_, err := tx.Exec(ctx, query, nJSON)
	return err
}

// GetPending returns a pending notification to be delivered if available.
func (m *Manager) GetPending(ctx context.Context, tx pgx.Tx) (*hub.Notification, error) {
	query := "select get_pending_notification()"
	var dataJSON []byte
	if err := tx.QueryRow(ctx, query).Scan(&dataJSON); err != nil {
		return nil, err
	}
	var n *hub.Notification
	if err := json.Unmarshal(dataJSON, &n); err != nil {
		return nil, err
	}
	return n, nil
}

// UpdateStatus the provided notification status in the database.
func (m *Manager) UpdateStatus(
	ctx context.Context,
	tx pgx.Tx,
	notificationID string,
	processed bool,
	processedErr error,
) error {
	if _, err := uuid.FromString(notificationID); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidInput, "invalid notification id")
	}
	query := "select update_notification_status($1::uuid, $2::boolean, $3::text)"
	var processedErrStr string
	if processedErr != nil {
		processedErrStr = processedErr.Error()
	}
	_, err := tx.Exec(ctx, query, notificationID, processed, processedErrStr)
	return err
}
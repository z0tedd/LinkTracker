package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/central-university-dev/go-z0tedd/internal/config"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/pkg"
)

type Creator struct {
	config *config.Config
	logger *slog.Logger
}

func NewCreator(logger *slog.Logger, cfg *config.Config) Creator {
	return Creator{config: cfg, logger: logger}
}

func (c Creator) Create(ctx context.Context) (Repository, error) {
	var repo Repository

	accessType := c.config.AccessType

	switch accessType {
	case "IN-MEMORY":
		repo = NewInMemoryRepository(c.logger)
	case "SQL":
		pool, err := pgxpool.New(ctx, c.config.DBURL)
		if err != nil {
			return nil, fmt.Errorf("sql repository creation: %w", err)
		}

		repo = NewSQLRepository(pool, c.logger)
	case "ORM":
		pool, err := pgxpool.New(ctx, c.config.DBURL)
		if err != nil {
			return nil, fmt.Errorf("orm repository creation: %w", err)
		}

		repo = NewORMRepository(pool, c.logger)
	default:
		repo = NewInMemoryRepository(c.logger)
	}

	return repo, nil
}

// SQLRepository implements the Repository interface using PostgreSQL.
type SQLRepository struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

// NewSQLRepository creates a new instance of SQLRepository.
func NewSQLRepository(db *pgxpool.Pool, logger *slog.Logger) *SQLRepository {
	return &SQLRepository{
		db:     db,
		logger: logger,
	}
}

// RegisterUser registers a new user.
func (r *SQLRepository) RegisterUser(_ context.Context, userID int64) error {
	r.logger.Debug("user registration", "userID", userID)
	return nil
}

//nolint:dupl //SQL and ORM repository has the same logic, but in specification we must create the same modules
func (r *SQLRepository) DeleteUser(ctx context.Context, userID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		r.logger.Error("failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			r.logger.Error("failed to rollback", "error", rollbackErr)
		}
	}()

	// Step 1: Retrieve all subIDs associated with the user
	subIDs, err := r.retrieveSubIDsForUser(ctx, tx, userID)
	if err != nil {
		return err
	}

	// Step 2: Remove the userID from tgChatIDs for each subID
	if err := r.removeUserIDFromTgChatIDs(ctx, tx, userID, subIDs); err != nil {
		return err
	}

	// Step 3: Delete the user's preferences from the users_preferences table
	if err := r.deleteUserPreferences(ctx, tx, userID); err != nil {
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Helper function to retrieve subIDs for a user.
func (r *SQLRepository) retrieveSubIDsForUser(ctx context.Context, tx pgx.Tx, userID int64) ([]int64, error) {
	query := `
        SELECT sub_id FROM users_preferences WHERE user_id = $1
    `

	rows, err := tx.Query(ctx, query, userID)
	if err != nil {
		r.logger.Error("failed to retrieve subIDs for user", "error", err)
		return nil, fmt.Errorf("failed to retrieve subIDs for user: %w", err)
	}
	defer rows.Close()

	var subIDs []int64

	for rows.Next() {
		var subID int64
		if err := rows.Scan(&subID); err != nil {
			r.logger.Error("failed to scan subID", "error", err)
			return nil, fmt.Errorf("failed to scan subID: %w", err)
		}

		subIDs = append(subIDs, subID)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration error", "error", err)
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return subIDs, nil
}

// Helper function to remove userID from tgChatIDs for each subscription.
func (r *SQLRepository) removeUserIDFromTgChatIDs(ctx context.Context, tx pgx.Tx, userID int64, subIDs []int64) error {
	for _, subID := range subIDs {
		updateQuery := `
            UPDATE subscriptions
            SET tg_chat_ids = array_remove(tg_chat_ids, $1)
            WHERE sub_id = $2
        `

		_, err := tx.Exec(ctx, updateQuery, userID, subID)
		if err != nil {
			r.logger.Error("failed to remove userID from tgChatIDs", "error", err)
			return fmt.Errorf("failed to remove userID from tgChatIDs: %w", err)
		}
	}

	return nil
}

// Helper function to delete user preferences.
func (r *SQLRepository) deleteUserPreferences(ctx context.Context, tx pgx.Tx, userID int64) error {
	deleteQuery := `
        DELETE FROM users_preferences WHERE user_id = $1
    `

	_, err := tx.Exec(ctx, deleteQuery, userID)
	if err != nil {
		r.logger.Error("failed to delete user preferences", "error", err)
		return fmt.Errorf("failed to delete user preferences: %w", err)
	}

	return nil
}

func (r *SQLRepository) AddSubscription(ctx context.Context, userID int64,
	sub *domain.Subscription, subPreferences domain.UserPreferences,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			r.logger.Error("failed to rollback", "error", rollbackErr)
		}
	}()

	// Step 1: Check if a subscription with the same URL exists
	subID, err := r.findOrCreateSubscription(ctx, tx, sub, userID)
	if err != nil {
		return err
	}

	// Step 2: Insert or update user preferences
	if err := r.upsertUserPreferences(ctx, tx, userID, subID, subPreferences); err != nil {
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Helper function to find or create a subscription.
func (r *SQLRepository) findOrCreateSubscription(ctx context.Context, tx pgx.Tx, sub *domain.Subscription, userID int64) (int64, error) {
	query := `
        SELECT sub_id FROM subscriptions WHERE url = $1
    `

	var subID int64
	err := tx.QueryRow(ctx, query, sub.URL).Scan(&subID)

	switch {
	case err == pgx.ErrNoRows:
		// Create a new subscription
		subID = time.Now().Unix()

		lastActivityJSON, err := json.Marshal(sub.LastActivity)
		if err != nil {
			r.logger.Error("failed to serialize lastActivity", "error", err)
			return 0, fmt.Errorf("failed to serialize lastActivity: %w", err)
		}

		insertQuery := `
            INSERT INTO subscriptions (sub_id, url, tg_chat_ids, last_activity)
            VALUES ($1, $2, $3, $4::JSONB)
        `

		_, err = tx.Exec(ctx, insertQuery, subID, sub.URL, []int64{userID}, lastActivityJSON)
		if err != nil {
			r.logger.Error("failed to insert new subscription", "error", err)
			return 0, fmt.Errorf("failed to insert new subscription: %w", err)
		}

	case err != nil:
		r.logger.Error("failed to find subscription", "error", err)
		return 0, fmt.Errorf("failed to find subscription: %w", err)

	default:
		// Add userID to tgChatIDs if not already present
		updateQuery := `
            UPDATE subscriptions
            SET tg_chat_ids = array_append(tg_chat_ids, $1)
            WHERE sub_id = $2 AND NOT ($1 = ANY(tg_chat_ids))
        `

		_, err = tx.Exec(ctx, updateQuery, userID, subID)
		if err != nil {
			r.logger.Error("failed to update subscription tgChatIDs", "error", err)
			return 0, fmt.Errorf("failed to update subscription tgChatIDs: %w", err)
		}
	}

	return subID, nil
}

// Helper function to insert or update user preferences.
func (r *SQLRepository) upsertUserPreferences(ctx context.Context, tx pgx.Tx,
	userID, subID int64, subPreferences domain.UserPreferences,
) error {
	prefQuery := `
        INSERT INTO users_preferences (user_id, sub_id, filters, tags, url)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (user_id, sub_id) DO UPDATE SET
            filters = EXCLUDED.filters,
            tags = EXCLUDED.tags,
            url = EXCLUDED.url
    `

	_, err := tx.Exec(ctx, prefQuery, userID, subID,
		pkg.ConvertToArray(subPreferences.Filters), subPreferences.Tags, subPreferences.URL)
	if err != nil {
		r.logger.Error("failed to add user preferences", "error", err)
		return fmt.Errorf("failed to add user preferences: %w", err)
	}

	return nil
}

// RemoveSubscription removes a subscription for a user.
func (r *SQLRepository) RemoveSubscription(ctx context.Context, userID int64, link string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			r.logger.Error("failed to rollback", "error", rollbackErr)
		}
	}()

	// Find subscription ID by URL
	var subID int64

	query := `SELECT sub_id FROM subscriptions WHERE url = $1`

	err = tx.QueryRow(ctx, query, link).Scan(&subID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("subscription not found")
		}

		r.logger.Error("failed to find subscription", "error", err)

		return fmt.Errorf("failed to find subscription: %w", err)
	}

	// Remove user from subscription's tgChatIDs
	updateQuery := `
        UPDATE subscriptions
        SET tg_chat_ids = array_remove(tg_chat_ids, $1)
        WHERE sub_id = $2
    `

	_, err = tx.Exec(ctx, updateQuery, userID, subID)
	if err != nil {
		r.logger.Error("failed to remove user from subscription", "error", err)
		return fmt.Errorf("failed to remove user from subscription: %w", err)
	}

	// Remove user preferences
	deleteQuery := `
        DELETE FROM users_preferences
        WHERE user_id = $1 AND sub_id = $2
    `

	_, err = tx.Exec(ctx, deleteQuery, userID, subID)
	if err != nil {
		r.logger.Error("failed to remove user preferences", "error", err)
		return fmt.Errorf("failed to remove user preferences: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetSubscriptionsForUser retrieves all subscriptions for a user.
func (r *SQLRepository) GetSubscriptionsForUser(ctx context.Context, tgChatID int64) ([]*domain.UserPreferences, error) {
	query := `
        SELECT sub_id, filters, tags, url
        FROM users_preferences
        WHERE user_id = $1
    `

	rows, err := r.db.Query(ctx, query, tgChatID)
	if err != nil {
		r.logger.Error("failed to get subscriptions for user", "error", err)
		return nil, fmt.Errorf("failed to get subscriptions for user: %w", err)
	}

	defer rows.Close()

	var result []*domain.UserPreferences

	for rows.Next() {
		var prefs domain.UserPreferences

		var filters []string
		if err := rows.Scan(&prefs.SubID, &filters, &prefs.Tags, &prefs.URL); err != nil {
			r.logger.Error("failed to scan user preferences", "error", err)
			return nil, fmt.Errorf("failed to scan user preferences: %w", err)
		}

		prefs.Filters = pkg.ConvertToMap(filters)
		result = append(result, &prefs)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration error", "error", err)
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return result, nil
}

// GetSubscription retrieves a subscription by ID.
func (r *SQLRepository) GetSubscription(ctx context.Context, subID int64) (*domain.Subscription, error) {
	query := `
        SELECT sub_id, url, tg_chat_ids, last_activity
        FROM subscriptions
        WHERE sub_id = $1
    `
	row := r.db.QueryRow(ctx, query, subID)

	var sub domain.Subscription

	var lastActivityJSON []byte
	if err := row.Scan(&sub.ID, &sub.URL, &sub.TgChatIDs, &lastActivityJSON); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &domain.Subscription{}, errors.New("subscription not found")
		}

		r.logger.Error("failed to get subscription", "error", err)

		return &domain.Subscription{}, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Deserialize lastActivity JSON
	if err := json.Unmarshal(lastActivityJSON, &sub.LastActivity); err != nil {
		r.logger.Error("failed to deserialize lastActivity", "error", err)
		return &domain.Subscription{}, fmt.Errorf("failed to deserialize lastActivity: %w", err)
	}

	return &sub, nil
}

// UpdateSubscription updates a subscription.
func (r *SQLRepository) UpdateSubscription(ctx context.Context, subID int64, newSub *domain.Subscription) error {
	query := `
        UPDATE subscriptions
        SET url = $1, tg_chat_ids = $2, last_activity = $3::JSONB
        WHERE sub_id = $4
    `

	// Marshal LastActivity to JSON
	lastActivityJSON, err := json.Marshal(newSub.LastActivity)
	if err != nil {
		r.logger.Error("failed to serialize lastActivity", "error", err)
		return fmt.Errorf("failed to serialize lastActivity: %w", err)
	}

	// Execute the query
	_, err = r.db.Exec(ctx, query, newSub.URL, newSub.TgChatIDs, lastActivityJSON, subID)
	if err != nil {
		r.logger.Error("failed to update subscription", "error", err)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

// UpdateSubscriptionActivity updates the activity of a subscription.
func (r *SQLRepository) UpdateSubscriptionActivity(ctx context.Context, subID int64, newActivity domain.Activity) error {
	query := `
        UPDATE subscriptions
        SET last_activity = $1::JSONB
        WHERE sub_id = $2
    `

	activityJSON, err := json.Marshal(newActivity)
	if err != nil {
		r.logger.Error("failed to serialize activity", "error", err)
		return fmt.Errorf("failed to serialize activity: %w", err)
	}

	_, err = r.db.Exec(ctx, query, activityJSON, subID)
	if err != nil {
		r.logger.Error("failed to update subscription activity", "error", err)
		return fmt.Errorf("failed to update subscription activity: %w", err)
	}

	return nil
}

// GetSubsID retrieves all subscription IDs.
func (r *SQLRepository) GetSubsID(ctx context.Context) *domain.Set {
	query := `
        SELECT sub_id FROM subscriptions
    `

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to get subscription IDs", "error", err)
		return nil
	}

	defer rows.Close()

	result := make(domain.Set)

	for rows.Next() {
		var subID int64
		if err := rows.Scan(&subID); err != nil {
			r.logger.Error("failed to scan subscription ID", "error", err)
			return nil
		}

		result.Add(subID)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration error", "error", err)
		return nil
	}

	return &result
}

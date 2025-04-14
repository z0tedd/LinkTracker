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

func NewCreator(logger *slog.Logger, config *config.Config) Creator {
	return Creator{config: config, logger: logger}
}

func (c Creator) Create() (Repository, error) {
	var repo Repository

	accessType := c.config.AccessType

	switch accessType {
	case "IN-MEMORY":
		repo = NewInMemoryRepository(c.logger)
	case "SQL":
		pool, err := pgxpool.New(context.Background(), c.config.DBURL)
		if err != nil {
			return nil, fmt.Errorf("sql repository creation: %w", err)
		}

		repo = NewSQLRepository(pool, c.logger)
	case "ORM":
		pool, err := pgxpool.New(context.Background(), c.config.DBURL)
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
func (r *SQLRepository) RegisterUser(userID int64) error {
	r.logger.Debug("user registration", "userID", userID)
	return nil
}

//nolint:dupl //SQL and ORM repository has the same logic, but in specification we must create the same modules
func (r *SQLRepository) DeleteUser(userID int64) error {
	tx, err := r.db.Begin(context.Background())
	if err != nil {
		r.logger.Error("failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := tx.Rollback(context.Background()); rollbackErr != nil {
			r.logger.Error("failed to rollback", "error", rollbackErr)
		}
	}()

	// Step 1: Retrieve all subIDs associated with the user
	subIDs, err := r.retrieveSubIDsForUser(tx, userID)
	if err != nil {
		return err
	}

	// Step 2: Remove the userID from tgChatIDs for each subID
	if err := r.removeUserIDFromTgChatIDs(tx, userID, subIDs); err != nil {
		return err
	}

	// Step 3: Delete the user's preferences from the users_preferences table
	if err := r.deleteUserPreferences(tx, userID); err != nil {
		return err
	}

	// Commit the transaction
	if err := tx.Commit(context.Background()); err != nil {
		r.logger.Error("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Helper function to retrieve subIDs for a user.
func (r *SQLRepository) retrieveSubIDsForUser(tx pgx.Tx, userID int64) ([]int64, error) {
	query := `
        SELECT subID FROM users_preferences WHERE userID = $1
    `

	rows, err := tx.Query(context.Background(), query, userID)
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
func (r *SQLRepository) removeUserIDFromTgChatIDs(tx pgx.Tx, userID int64, subIDs []int64) error {
	for _, subID := range subIDs {
		updateQuery := `
            UPDATE subscriptions
            SET tgChatIDs = array_remove(tgChatIDs, $1)
            WHERE subID = $2
        `

		_, err := tx.Exec(context.Background(), updateQuery, userID, subID)
		if err != nil {
			r.logger.Error("failed to remove userID from tgChatIDs", "error", err)
			return fmt.Errorf("failed to remove userID from tgChatIDs: %w", err)
		}
	}

	return nil
}

// Helper function to delete user preferences.
func (r *SQLRepository) deleteUserPreferences(tx pgx.Tx, userID int64) error {
	deleteQuery := `
        DELETE FROM users_preferences WHERE userID = $1
    `

	_, err := tx.Exec(context.Background(), deleteQuery, userID)
	if err != nil {
		r.logger.Error("failed to delete user preferences", "error", err)
		return fmt.Errorf("failed to delete user preferences: %w", err)
	}

	return nil
}

//nolint:dupl //SQL and ORM repository has the same logic, but in specification we must create the same modules
func (r *SQLRepository) AddSubscription(userID int64, sub *domain.Subscription, subPreferences domain.UserPreferences) error {
	tx, err := r.db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := tx.Rollback(context.Background()); rollbackErr != nil {
			r.logger.Error("failed to rollback", "error", rollbackErr)
		}
	}()

	// Step 1: Check if a subscription with the same URL exists
	subID, err := r.findOrCreateSubscription(tx, sub, userID)
	if err != nil {
		return err
	}

	// Step 2: Insert or update user preferences
	if err := r.upsertUserPreferences(tx, userID, subID, subPreferences); err != nil {
		return err
	}

	// Commit the transaction
	if err := tx.Commit(context.Background()); err != nil {
		r.logger.Error("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Helper function to find or create a subscription.
func (r *SQLRepository) findOrCreateSubscription(tx pgx.Tx, sub *domain.Subscription, userID int64) (int64, error) {
	query := `
        SELECT subID FROM subscriptions WHERE url = $1
    `

	var subID int64
	err := tx.QueryRow(context.Background(), query, sub.URL).Scan(&subID)

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
            INSERT INTO subscriptions (subID, url, tgChatIDs, lastActivity)
            VALUES ($1, $2, $3, $4::JSONB)
        `

		_, err = tx.Exec(context.Background(), insertQuery, subID, sub.URL, []int64{userID}, lastActivityJSON)
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
            SET tgChatIDs = array_append(tgChatIDs, $1)
            WHERE subID = $2 AND NOT ($1 = ANY(tgChatIDs))
        `

		_, err = tx.Exec(context.Background(), updateQuery, userID, subID)
		if err != nil {
			r.logger.Error("failed to update subscription tgChatIDs", "error", err)
			return 0, fmt.Errorf("failed to update subscription tgChatIDs: %w", err)
		}
	}

	return subID, nil
}

// Helper function to insert or update user preferences.
func (r *SQLRepository) upsertUserPreferences(tx pgx.Tx, userID, subID int64, subPreferences domain.UserPreferences) error {
	prefQuery := `
        INSERT INTO users_preferences (userID, subID, filters, tags, url)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (userID, subID) DO UPDATE SET
            filters = EXCLUDED.filters,
            tags = EXCLUDED.tags,
            url = EXCLUDED.url
    `

	_, err := tx.Exec(context.Background(), prefQuery, userID, subID,
		pkg.ConvertToArray(subPreferences.Filters), subPreferences.Tags, subPreferences.URL)
	if err != nil {
		r.logger.Error("failed to add user preferences", "error", err)
		return fmt.Errorf("failed to add user preferences: %w", err)
	}

	return nil
}

// RemoveSubscription removes a subscription for a user.
func (r *SQLRepository) RemoveSubscription(userID int64, link string) error {
	tx, err := r.db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := tx.Rollback(context.Background()); rollbackErr != nil {
			r.logger.Error("failed to rollback", "error", rollbackErr)
		}
	}()

	// Find subscription ID by URL
	var subID int64

	query := `SELECT subID FROM subscriptions WHERE url = $1`

	err = tx.QueryRow(context.Background(), query, link).Scan(&subID)
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
        SET tgChatIDs = array_remove(tgChatIDs, $1)
        WHERE subID = $2
    `

	_, err = tx.Exec(context.Background(), updateQuery, userID, subID)
	if err != nil {
		r.logger.Error("failed to remove user from subscription", "error", err)
		return fmt.Errorf("failed to remove user from subscription: %w", err)
	}

	// Remove user preferences
	deleteQuery := `
        DELETE FROM users_preferences
        WHERE userID = $1 AND subID = $2
    `

	_, err = tx.Exec(context.Background(), deleteQuery, userID, subID)
	if err != nil {
		r.logger.Error("failed to remove user preferences", "error", err)
		return fmt.Errorf("failed to remove user preferences: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		r.logger.Error("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetSubscriptionsForUser retrieves all subscriptions for a user.
func (r *SQLRepository) GetSubscriptionsForUser(tgChatID int64) ([]domain.UserPreferences, error) {
	query := `
        SELECT subID, filters, tags, url
        FROM users_preferences
        WHERE userID = $1
    `

	rows, err := r.db.Query(context.Background(), query, tgChatID)
	if err != nil {
		r.logger.Error("failed to get subscriptions for user", "error", err)
		return nil, fmt.Errorf("failed to get subscriptions for user: %w", err)
	}

	defer rows.Close()

	var result []domain.UserPreferences

	for rows.Next() {
		var prefs domain.UserPreferences

		var filters []string
		if err := rows.Scan(&prefs.SubID, &filters, &prefs.Tags, &prefs.URL); err != nil {
			r.logger.Error("failed to scan user preferences", "error", err)
			return nil, fmt.Errorf("failed to scan user preferences: %w", err)
		}

		prefs.Filters = pkg.ConvertToMap(filters)
		result = append(result, prefs)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration error", "error", err)
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return result, nil
}

// GetSubscription retrieves a subscription by ID.
func (r *SQLRepository) GetSubscription(subID int64) (domain.Subscription, error) {
	query := `
        SELECT subID, url, tgChatIDs, lastActivity
        FROM subscriptions
        WHERE subID = $1
    `
	row := r.db.QueryRow(context.Background(), query, subID)

	var sub domain.Subscription

	var lastActivityJSON []byte
	if err := row.Scan(&sub.ID, &sub.URL, &sub.TgChatIDs, &lastActivityJSON); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Subscription{}, errors.New("subscription not found")
		}

		r.logger.Error("failed to get subscription", "error", err)

		return domain.Subscription{}, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Deserialize lastActivity JSON
	if err := json.Unmarshal(lastActivityJSON, &sub.LastActivity); err != nil {
		r.logger.Error("failed to deserialize lastActivity", "error", err)
		return domain.Subscription{}, fmt.Errorf("failed to deserialize lastActivity: %w", err)
	}

	return sub, nil
}

// UpdateSubscription updates a subscription.
func (r *SQLRepository) UpdateSubscription(subID int64, newSub *domain.Subscription) error {
	query := `
        UPDATE subscriptions
        SET url = $1, tgChatIDs = $2, lastActivity = $3::JSONB
        WHERE subID = $4
    `

	// Marshal LastActivity to JSON
	lastActivityJSON, err := json.Marshal(newSub.LastActivity)
	if err != nil {
		r.logger.Error("failed to serialize lastActivity", "error", err)
		return fmt.Errorf("failed to serialize lastActivity: %w", err)
	}

	// Execute the query
	_, err = r.db.Exec(context.Background(), query, newSub.URL, newSub.TgChatIDs, lastActivityJSON, subID)
	if err != nil {
		r.logger.Error("failed to update subscription", "error", err)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

// UpdateSubscriptionActivity updates the activity of a subscription.
func (r *SQLRepository) UpdateSubscriptionActivity(subID int64, newActivity domain.Activity) error {
	query := `
        UPDATE subscriptions
        SET lastActivity = $1::JSONB
        WHERE subID = $2
    `

	activityJSON, err := json.Marshal(newActivity)
	if err != nil {
		r.logger.Error("failed to serialize activity", "error", err)
		return fmt.Errorf("failed to serialize activity: %w", err)
	}

	_, err = r.db.Exec(context.Background(), query, activityJSON, subID)
	if err != nil {
		r.logger.Error("failed to update subscription activity", "error", err)
		return fmt.Errorf("failed to update subscription activity: %w", err)
	}

	return nil
}

// GetSubsID retrieves all subscription IDs.
func (r *SQLRepository) GetSubsID() domain.Set {
	query := `
        SELECT subID FROM subscriptions
    `

	rows, err := r.db.Query(context.Background(), query)
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

	return result
}

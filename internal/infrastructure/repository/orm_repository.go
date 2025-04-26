package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/pkg"
)

// ORMRepository implements the Repository interface using squirrel as the SQL builder.
type ORMRepository struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

// NewORMRepository creates a new instance of ORMRepository.
func NewORMRepository(db *pgxpool.Pool, logger *slog.Logger) *ORMRepository {
	return &ORMRepository{
		db:     db,
		logger: logger,
	}
}

// RegisterUser registers a new user.
func (r *ORMRepository) RegisterUser(_ context.Context, userID int64) error {
	r.logger.Debug("user registration", "userID", userID)
	return nil
}

// DeleteUser deletes a user and removes their preferences.
//
//nolint:dupl //SQL and ORM repository has the same logic, but in specification we must create the same modules
func (r *ORMRepository) DeleteUser(ctx context.Context, userID int64) error {
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
	if err := r.removeUserIDFromTgChatIDs(ctx, tx, subIDs, userID); err != nil {
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

// Helper function to retrieve subIDs for a given user.
func (r *ORMRepository) retrieveSubIDsForUser(ctx context.Context, tx pgx.Tx, userID int64) ([]int64, error) {
	query := squirrel.Select("subID").
		From("users_preferences").
		Where(squirrel.Eq{"userID": userID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := tx.Query(ctx, sql, args...)
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

// Helper function to remove userID from tgChatIDs for each subID.
func (r *ORMRepository) removeUserIDFromTgChatIDs(ctx context.Context, tx pgx.Tx, subIDs []int64, userID int64) error {
	for _, subID := range subIDs {
		updateQuery := squirrel.Update("subscriptions").
			Set("tgChatIDs", squirrel.Expr("array_remove(tgChatIDs, ?)", userID)).
			Where(squirrel.Eq{"subID": subID}).
			PlaceholderFormat(squirrel.Dollar)

		sql, args, err := updateQuery.ToSql()
		if err != nil {
			r.logger.Error("failed to build update query", "error", err)
			return fmt.Errorf("failed to build update query: %w", err)
		}

		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			r.logger.Error("failed to remove userID from tgChatIDs", "error", err)
			return fmt.Errorf("failed to remove userID from tgChatIDs: %w", err)
		}
	}

	return nil
}

// Helper function to delete user preferences.
func (r *ORMRepository) deleteUserPreferences(ctx context.Context, tx pgx.Tx, userID int64) error {
	deleteQuery := squirrel.Delete("users_preferences").
		Where(squirrel.Eq{"userID": userID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := deleteQuery.ToSql()
	if err != nil {
		r.logger.Error("failed to build delete query", "error", err)
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		r.logger.Error("failed to delete user preferences", "error", err)
		return fmt.Errorf("failed to delete user preferences: %w", err)
	}

	return nil
}

// AddSubscription adds a new subscription for a user.
//

func (r *ORMRepository) AddSubscription(ctx context.Context, userID int64, sub *domain.Subscription,
	subPreferences domain.UserPreferences,
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
func (r *ORMRepository) findOrCreateSubscription(ctx context.Context, tx pgx.Tx, sub *domain.Subscription, userID int64) (int64, error) {
	query := squirrel.Select("subID").
		From("subscriptions").
		Where(squirrel.Eq{"url": sub.URL}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return 0, fmt.Errorf("failed to build query: %w", err)
	}

	var subID int64
	err = tx.QueryRow(ctx, sql, args...).Scan(&subID)

	switch {
	case err == pgx.ErrNoRows:
		// Create a new subscription
		subID = time.Now().Unix()

		lastActivityJSON, err := json.Marshal(sub.LastActivity)
		if err != nil {
			r.logger.Error("failed to serialize lastActivity", "error", err)
			return 0, fmt.Errorf("failed to serialize lastActivity: %w", err)
		}

		insertQuery := squirrel.Insert("subscriptions").
			Columns("subID", "url", "tgChatIDs", "lastActivity").
			Values(subID, sub.URL, []int64{userID}, lastActivityJSON).
			PlaceholderFormat(squirrel.Dollar)

		sql, args, err := insertQuery.ToSql()
		if err != nil {
			r.logger.Error("failed to build insert query", "error", err)
			return 0, fmt.Errorf("failed to build insert query: %w", err)
		}

		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			r.logger.Error("failed to insert new subscription", "error", err)
			return 0, fmt.Errorf("failed to insert new subscription: %w", err)
		}

	case err != nil:
		r.logger.Error("failed to find subscription", "error", err)
		return 0, fmt.Errorf("failed to find subscription: %w", err)

	default:
		// Add userID to tgChatIDs if not already present
		updateQuery := squirrel.Update("subscriptions").
			Set("tgChatIDs", squirrel.Expr("array_append(tgChatIDs, ?)", userID)).
			Where(squirrel.Eq{"subID": subID}).
			Where(squirrel.Expr("NOT (? = ANY(tgChatIDs))", userID)).
			PlaceholderFormat(squirrel.Dollar)

		sql, args, err := updateQuery.ToSql()
		if err != nil {
			r.logger.Error("failed to build update query", "error", err)
			return 0, fmt.Errorf("failed to build update query: %w", err)
		}

		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			r.logger.Error("failed to update subscription tgChatIDs", "error", err)
			return 0, fmt.Errorf("failed to update subscription tgChatIDs: %w", err)
		}
	}

	return subID, nil
}

// Helper function to insert or update user preferences.
func (r *ORMRepository) upsertUserPreferences(ctx context.Context, tx pgx.Tx, userID, subID int64,
	subPreferences domain.UserPreferences,
) error {
	prefQuery := squirrel.Insert("users_preferences").
		Columns("userID", "subID", "filters", "tags", "url").
		Values(userID, subID, pkg.ConvertToArray(subPreferences.Filters), subPreferences.Tags, subPreferences.URL).
		Suffix("ON CONFLICT (userID, subID) DO UPDATE SET filters = EXCLUDED.filters, tags = EXCLUDED.tags, url = EXCLUDED.url").
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := prefQuery.ToSql()
	if err != nil {
		r.logger.Error("failed to build preferences query", "error", err)
		return fmt.Errorf("failed to build preferences query: %w", err)
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		r.logger.Error("failed to add user preferences", "error", err)
		return fmt.Errorf("failed to add user preferences: %w", err)
	}

	return nil
}

// RemoveSubscription removes a subscription for a user.
func (r *ORMRepository) RemoveSubscription(ctx context.Context, userID int64, link string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			r.logger.Error("failed to rollback", "error", rollbackErr)
		}
	}()

	// Step 1: Find subscription ID by URL
	subID, err := r.findSubscriptionIDByURL(ctx, tx, link)
	if err != nil {
		return err
	}

	// Step 2: Remove user from subscription's tgChatIDs
	if err := r.removeUserFromTgChatIDs(ctx, tx, subID, userID); err != nil {
		return err
	}

	// Step 3: Remove user preferences
	if err := r.deleteUserPreferencesForSub(ctx, tx, userID, subID); err != nil {
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Helper function to find subscription ID by URL.
func (r *ORMRepository) findSubscriptionIDByURL(ctx context.Context, tx pgx.Tx, link string) (int64, error) {
	query := squirrel.Select("subID").
		From("subscriptions").
		Where(squirrel.Eq{"url": link}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return 0, fmt.Errorf("failed to build query: %w", err)
	}

	var subID int64

	err = tx.QueryRow(ctx, sql, args...).Scan(&subID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("subscription not found")
		}

		r.logger.Error("failed to find subscription", "error", err)

		return 0, fmt.Errorf("failed to find subscription: %w", err)
	}

	return subID, nil
}

// Helper function to remove a user from tgChatIDs for a subscription.
func (r *ORMRepository) removeUserFromTgChatIDs(ctx context.Context, tx pgx.Tx, subID, userID int64) error {
	updateQuery := squirrel.Update("subscriptions").
		Set("tgChatIDs", squirrel.Expr("array_remove(tgChatIDs, ?)", userID)).
		Where(squirrel.Eq{"subID": subID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := updateQuery.ToSql()
	if err != nil {
		r.logger.Error("failed to build update query", "error", err)
		return fmt.Errorf("failed to build update query: %w", err)
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		r.logger.Error("failed to remove user from subscription", "error", err)
		return fmt.Errorf("failed to remove user from subscription: %w", err)
	}

	return nil
}

// Helper function to delete user preferences for a subscription.
func (r *ORMRepository) deleteUserPreferencesForSub(ctx context.Context, tx pgx.Tx, userID, subID int64) error {
	deleteQuery := squirrel.Delete("users_preferences").
		Where(squirrel.Eq{"userID": userID, "subID": subID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := deleteQuery.ToSql()
	if err != nil {
		r.logger.Error("failed to build delete query", "error", err)
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		r.logger.Error("failed to remove user preferences", "error", err)
		return fmt.Errorf("failed to remove user preferences: %w", err)
	}

	return nil
}

// GetSubscriptionsForUser retrieves all subscriptions for a user.
func (r *ORMRepository) GetSubscriptionsForUser(ctx context.Context, tgChatID int64) ([]*domain.UserPreferences, error) {
	query := squirrel.Select("subID", "filters", "tags", "url").
		From("users_preferences").
		Where(squirrel.Eq{"userID": tgChatID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
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
func (r *ORMRepository) GetSubscription(ctx context.Context, subID int64) (*domain.Subscription, error) {
	query := squirrel.Select("subID", "url", "tgChatIDs", "lastActivity").
		From("subscriptions").
		Where(squirrel.Eq{"subID": subID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return &domain.Subscription{}, fmt.Errorf("failed to build query: %w", err)
	}

	row := r.db.QueryRow(ctx, sql, args...)

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
func (r *ORMRepository) UpdateSubscription(ctx context.Context, subID int64, newSub *domain.Subscription) error {
	query := squirrel.Update("subscriptions").
		Set("url", newSub.URL).
		Set("tgChatIDs", newSub.TgChatIDs).
		Set("lastActivity", squirrel.Expr("?::JSONB", newSub.LastActivity)).
		Where(squirrel.Eq{"subID": subID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return fmt.Errorf("failed to build query: %w", err)
	}

	// Marshal LastActivity to JSON
	lastActivityJSON, err := json.Marshal(newSub.LastActivity)
	if err != nil {
		r.logger.Error("failed to serialize lastActivity", "error", err)
		return fmt.Errorf("failed to serialize lastActivity: %w", err)
	}

	args[len(args)-1] = lastActivityJSON // Replace the placeholder with the serialized JSON

	_, err = r.db.Exec(ctx, sql, args...)
	if err != nil {
		r.logger.Error("failed to update subscription", "error", err)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

// UpdateSubscriptionActivity updates the activity of a subscription.
func (r *ORMRepository) UpdateSubscriptionActivity(ctx context.Context, subID int64, newActivity domain.Activity) error {
	query := squirrel.Update("subscriptions").
		Set("lastActivity", squirrel.Expr("?::JSONB", newActivity)).
		Where(squirrel.Eq{"subID": subID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return fmt.Errorf("failed to build query: %w", err)
	}

	activityJSON, err := json.Marshal(newActivity)
	if err != nil {
		r.logger.Error("failed to serialize activity", "error", err)
		return fmt.Errorf("failed to serialize activity: %w", err)
	}

	args[len(args)-1] = activityJSON // Replace the placeholder with the serialized JSON

	_, err = r.db.Exec(ctx, sql, args...)
	if err != nil {
		r.logger.Error("failed to update subscription activity", "error", err)
		return fmt.Errorf("failed to update subscription activity: %w", err)
	}

	return nil
}

// GetSubsID retrieves all subscription IDs.
func (r *ORMRepository) GetSubsID(ctx context.Context) *domain.Set {
	query := squirrel.Select("subID").
		From("subscriptions").
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return nil
	}

	rows, err := r.db.Query(ctx, sql, args...)
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

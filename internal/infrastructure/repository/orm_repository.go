package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
func (r *ORMRepository) RegisterUser(userID int64) error {
	r.logger.Debug("user registration", "userID", userID)
	return nil
}

// DeleteUser deletes a user and removes their preferences.
func (r *ORMRepository) DeleteUser(userID int64) error {
	tx, err := r.db.Begin(context.Background())
	if err != nil {
		r.logger.Error("failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	// Step 1: Retrieve all subIDs associated with the user
	query := squirrel.Select("subID").
		From("users_preferences").
		Where(squirrel.Eq{"userID": userID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := tx.Query(context.Background(), sql, args...)
	if err != nil {
		r.logger.Error("failed to retrieve subIDs for user", "error", err)
		return fmt.Errorf("failed to retrieve subIDs for user: %w", err)
	}
	defer rows.Close()

	var subIDs []int64
	for rows.Next() {
		var subID int64
		if err := rows.Scan(&subID); err != nil {
			r.logger.Error("failed to scan subID", "error", err)
			return fmt.Errorf("failed to scan subID: %w", err)
		}
		subIDs = append(subIDs, subID)
	}
	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration error", "error", err)
		return fmt.Errorf("row iteration error: %w", err)
	}

	// Step 2: Remove the userID from tgChatIDs for each subID
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

		_, err = tx.Exec(context.Background(), sql, args...)
		if err != nil {
			r.logger.Error("failed to remove userID from tgChatIDs", "error", err)
			return fmt.Errorf("failed to remove userID from tgChatIDs: %w", err)
		}
	}

	// Step 3: Delete the user's preferences from the users_preferences table
	deleteQuery := squirrel.Delete("users_preferences").
		Where(squirrel.Eq{"userID": userID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err = deleteQuery.ToSql()
	if err != nil {
		r.logger.Error("failed to build delete query", "error", err)
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = tx.Exec(context.Background(), sql, args...)
	if err != nil {
		r.logger.Error("failed to delete user preferences", "error", err)
		return fmt.Errorf("failed to delete user preferences: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(context.Background()); err != nil {
		r.logger.Error("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// AddSubscription adds a new subscription for a user.
func (r *ORMRepository) AddSubscription(userID int64, sub domain.Subscription, subPreferences domain.UserPreferences) error {
	tx, err := r.db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	// Step 1: Check if a subscription with the same URL exists
	query := squirrel.Select("subID").
		From("subscriptions").
		Where(squirrel.Eq{"url": sub.URL}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return fmt.Errorf("failed to build query: %w", err)
	}

	var subID int64
	err = tx.QueryRow(context.Background(), sql, args...).Scan(&subID)
	if err == pgx.ErrNoRows {
		// Step 2: If no subscription exists, create a new one
		subID = time.Now().Unix()
		lastActivityJSON, err := json.Marshal(sub.LastActivity)
		if err != nil {
			r.logger.Error("failed to serialize lastActivity", "error", err)
			return fmt.Errorf("failed to serialize lastActivity: %w", err)
		}

		insertQuery := squirrel.Insert("subscriptions").
			Columns("subID", "url", "tgChatIDs", "lastActivity").
			Values(subID, sub.URL, []int64{userID}, lastActivityJSON).
			PlaceholderFormat(squirrel.Dollar)

		sql, args, err := insertQuery.ToSql()
		if err != nil {
			r.logger.Error("failed to build insert query", "error", err)
			return fmt.Errorf("failed to build insert query: %w", err)
		}

		_, err = tx.Exec(context.Background(), sql, args...)
		if err != nil {
			r.logger.Error("failed to insert new subscription", "error", err)
			return fmt.Errorf("failed to insert new subscription: %w", err)
		}
	} else if err != nil {
		r.logger.Error("failed to find subscription", "error", err)
		return fmt.Errorf("failed to find subscription: %w", err)
	} else {
		// Step 3: If a subscription exists, add userID to tgChatIDs
		updateQuery := squirrel.Update("subscriptions").
			Set("tgChatIDs", squirrel.Expr("array_append(tgChatIDs, ?)", userID)).
			Where(squirrel.Eq{"subID": subID}).
			Where(squirrel.Expr("NOT (? = ANY(tgChatIDs))", userID)).
			PlaceholderFormat(squirrel.Dollar)

		sql, args, err := updateQuery.ToSql()
		if err != nil {
			r.logger.Error("failed to build update query", "error", err)
			return fmt.Errorf("failed to build update query: %w", err)
		}

		_, err = tx.Exec(context.Background(), sql, args...)
		if err != nil {
			r.logger.Error("failed to update subscription tgChatIDs", "error", err)
			return fmt.Errorf("failed to update subscription tgChatIDs: %w", err)
		}
	}

	// Step 4: Insert or update user preferences
	prefQuery := squirrel.Insert("users_preferences").
		Columns("userID", "subID", "filters", "tags", "url").
		Values(userID, subID, convertToArray(subPreferences.Filters), subPreferences.Tags, subPreferences.URL).
		Suffix("ON CONFLICT (userID, subID) DO UPDATE SET filters = EXCLUDED.filters, tags = EXCLUDED.tags, url = EXCLUDED.url").
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err = prefQuery.ToSql()
	if err != nil {
		r.logger.Error("failed to build preferences query", "error", err)
		return fmt.Errorf("failed to build preferences query: %w", err)
	}

	_, err = tx.Exec(context.Background(), sql, args...)
	if err != nil {
		r.logger.Error("failed to add user preferences", "error", err)
		return fmt.Errorf("failed to add user preferences: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(context.Background()); err != nil {
		r.logger.Error("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// RemoveSubscription removes a subscription for a user.
func (r *ORMRepository) RemoveSubscription(userID int64, link string) error {
	tx, err := r.db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	// Find subscription ID by URL
	query := squirrel.Select("subID").
		From("subscriptions").
		Where(squirrel.Eq{"url": link}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return fmt.Errorf("failed to build query: %w", err)
	}

	var subID int64
	err = tx.QueryRow(context.Background(), sql, args...).Scan(&subID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("subscription not found")
		}
		r.logger.Error("failed to find subscription", "error", err)
		return fmt.Errorf("failed to find subscription: %w", err)
	}

	// Remove user from subscription's tgChatIDs
	updateQuery := squirrel.Update("subscriptions").
		Set("tgChatIDs", squirrel.Expr("array_remove(tgChatIDs, ?)", userID)).
		Where(squirrel.Eq{"subID": subID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err = updateQuery.ToSql()
	if err != nil {
		r.logger.Error("failed to build update query", "error", err)
		return fmt.Errorf("failed to build update query: %w", err)
	}

	_, err = tx.Exec(context.Background(), sql, args...)
	if err != nil {
		r.logger.Error("failed to remove user from subscription", "error", err)
		return fmt.Errorf("failed to remove user from subscription: %w", err)
	}

	// Remove user preferences
	deleteQuery := squirrel.Delete("users_preferences").
		Where(squirrel.Eq{"userID": userID, "subID": subID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err = deleteQuery.ToSql()
	if err != nil {
		r.logger.Error("failed to build delete query", "error", err)
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = tx.Exec(context.Background(), sql, args...)
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
func (r *ORMRepository) GetSubscriptionsForUser(tgChatID int64) ([]domain.UserPreferences, error) {
	query := squirrel.Select("subID", "filters", "tags", "url").
		From("users_preferences").
		Where(squirrel.Eq{"userID": tgChatID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(context.Background(), sql, args...)
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
		prefs.Filters = convertToMap(filters)
		result = append(result, prefs)
	}
	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration error", "error", err)
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	return result, nil
}

// GetSubscription retrieves a subscription by ID.
func (r *ORMRepository) GetSubscription(subID int64) (domain.Subscription, error) {
	query := squirrel.Select("subID", "url", "tgChatIDs", "lastActivity").
		From("subscriptions").
		Where(squirrel.Eq{"subID": subID}).
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return domain.Subscription{}, fmt.Errorf("failed to build query: %w", err)
	}

	row := r.db.QueryRow(context.Background(), sql, args...)
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
func (r *ORMRepository) UpdateSubscription(subID int64, newSub domain.Subscription) error {
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

	_, err = r.db.Exec(context.Background(), sql, args...)
	if err != nil {
		r.logger.Error("failed to update subscription", "error", err)
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	return nil
}

// UpdateSubscriptionActivity updates the activity of a subscription.
func (r *ORMRepository) UpdateSubscriptionActivity(subID int64, newActivity domain.Activity) error {
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

	_, err = r.db.Exec(context.Background(), sql, args...)
	if err != nil {
		r.logger.Error("failed to update subscription activity", "error", err)
		return fmt.Errorf("failed to update subscription activity: %w", err)
	}
	return nil
}

// GetSubsID retrieves all subscription IDs.
func (r *ORMRepository) GetSubsID() domain.Set {
	query := squirrel.Select("subID").
		From("subscriptions").
		PlaceholderFormat(squirrel.Dollar)

	sql, args, err := query.ToSql()
	if err != nil {
		r.logger.Error("failed to build query", "error", err)
		return nil
	}

	rows, err := r.db.Query(context.Background(), sql, args...)
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

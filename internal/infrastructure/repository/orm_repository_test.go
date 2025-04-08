package repository_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/central-university-dev/go-z0tedd/internal/domain"
	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/repository"
)

// setupPostgresContainer starts a PostgreSQL container and returns a connection pool.
func setupPostgresContainer(t *testing.T) (pool *pgxpool.Pool, cleanup func()) {
	ctx := context.Background()

	// Define PostgreSQL container request
	req := testcontainers.ContainerRequest{
		Image:        "postgres:latest",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpassword",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	// Start the container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start PostgreSQL container: %v", err)
	}

	// Get the host and port of the container
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	// Construct the connection string
	// postgresql://postgres_user:postgres_password@localhost:5430/postgres_db
	connString := fmt.Sprintf("postgresql://testuser:testpassword@%s:%s/testdb?sslmode=disable", host, port.Port())
	t.Log(host, port.Port())

	// Connect to the database
	pool, err = pgxpool.New(ctx, connString)
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}

	t.Log("pool connected")

	// Apply migrations (e.g., create tables)
	err = applyMigrations(pool, t)
	if err != nil {
		retryCount := 10
		for range retryCount {
			err = applyMigrations(pool, t)
			if err == nil {
				break
			}

			time.Sleep(2 * time.Second)
		}
	}

	if err != nil {
		t.Fatal(err)
	}

	// Cleanup function
	cleanup = func() {
		pool.Close()

		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %v", err)
		}
	}

	return pool, cleanup
}

// applyMigrations applies schema migrations to the database.
func applyMigrations(pool *pgxpool.Pool, t *testing.T) error {
	ctx := context.Background()

	// Define SQL schema for testing
	schema := `
    CREATE TABLE IF NOT EXISTS subscriptions (
        subID BIGINT PRIMARY KEY,
        url TEXT NOT NULL,
        tgChatIDs BIGINT[] NOT NULL,
        lastActivity JSONB NOT NULL
    );

    CREATE TABLE IF NOT EXISTS users_preferences (
        userID BIGINT NOT NULL,
        subID BIGINT NOT NULL,
        filters TEXT[] NOT NULL,
        tags TEXT[] NOT NULL,
        url TEXT NOT NULL,
        PRIMARY KEY (userID, subID)
    );
    `

	// Execute schema creation
	_, err := pool.Exec(ctx, schema)
	if err != nil {
		t.Logf("failed to apply migrations: %v", err)
		return err
	}

	return nil
}

func TestRegisterUser(t *testing.T) {
	pool, cleanup := setupPostgresContainer(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := repository.NewORMRepository(pool, logger)

	err := repo.RegisterUser(12345)
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}
}

func TestDeleteUser(t *testing.T) {
	pool, cleanup := setupPostgresContainer(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := repository.NewORMRepository(pool, logger)

	// Insert test data
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
        INSERT INTO users_preferences (userID, subID, filters, tags, url)
        VALUES (12345, 67890, '{"filter1"}', '{"tag1"}', 'http://example.com');
    `)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Call DeleteUser
	err = repo.DeleteUser(12345)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// Verify deletion
	var count int

	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM users_preferences WHERE userID = $1", 12345).Scan(&count)
	if err != nil {
		t.Fatalf("failed to verify deletion: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 rows, got %d", count)
	}
}

func TestAddSubscription(t *testing.T) {
	pool, cleanup := setupPostgresContainer(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := repository.NewORMRepository(pool, logger)

	sub := domain.Subscription{
		URL: "http://example.com",
		LastActivity: domain.Activity{
			Title:    "Test Title",
			Username: "testuser",
			DateUnix: time.Now().Unix(),
		},
	}
	prefs := domain.UserPreferences{
		Filters: map[string]string{"key": "value"},
		Tags:    []string{"tag1"},
		URL:     "http://example.com",
	}

	err := repo.AddSubscription(12345, &sub, prefs)
	if err != nil {
		t.Fatalf("AddSubscription failed: %v", err)
	}

	// Verify subscription was added
	var count int

	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM subscriptions WHERE url = $1", sub.URL).Scan(&count)
	if err != nil {
		t.Fatalf("failed to verify subscription: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 subscription, got %d", count)
	}
}

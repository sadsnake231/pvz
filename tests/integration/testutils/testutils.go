package testutils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SetupTestDB(ctx context.Context, t *testing.T) (testcontainers.Container, *pgxpool.Pool) {
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		t.Fatalf("Не смог запустить контейнер: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Не смог получить адрес соединения: %v", err)
	}

	if err := applyMigrations(connStr); err != nil {
		t.Fatalf("FНе смог накатить миграции: %v", err)
	}

	db, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("Не смог подключиться к БД: %v", err)
	}

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("Не смог пингануть БД: %v", err)
	}

	return pgContainer, db
}

func TeardownTestDB(ctx context.Context, t *testing.T, container testcontainers.Container, db *pgxpool.Pool) {
	if db != nil {
		db.Close()
	}
	if container != nil {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Не смог остановить контейнер: %v", err)
		}
	}
}

func applyMigrations(connStr string) error {
	cmd := exec.Command("goose", "-dir", "../../../migrations", "postgres", connStr, "up")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ClearDatabase(ctx context.Context, db *pgxpool.Pool) error {
	tables := []string{"orders", "users", "packaging_types"}
	for _, table := range tables {
		if _, err := db.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			return err
		}
	}
	return nil
}

//go:build integration

package dbkit_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	dbkit "github.com/Hallucination-Team/go-dbkit"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// mustOpen runs the full verification chain: open, ping, SELECT 1.
func mustOpen(t *testing.T, cfg dbkit.Config) *sql.DB {
	t.Helper()
	database, err := dbkit.Open(cfg)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	var got int
	if err := database.QueryRowContext(ctx, "SELECT 1").Scan(&got); err != nil {
		t.Fatalf("SELECT 1: %v", err)
	}
	if got != 1 {
		t.Fatalf("SELECT 1 = %d, want 1", got)
	}
	return database
}

func TestIntegrationPostgresStandalone(t *testing.T) {
	mustOpen(t, dbkit.Config{
		Driver:   dbkit.DriverPostgres,
		Mode:     dbkit.ModeStandalone,
		Host:     envOr("DBKIT_TEST_PG_HOST", "127.0.0.1"),
		Port:     5432,
		Username: envOr("DBKIT_TEST_PG_USER", "app_user"),
		Password: envOr("DBKIT_TEST_PG_PASSWORD", "your_password"),
		Database: envOr("DBKIT_TEST_PG_DATABASE", "app_db"),
		Options: map[string]string{
			"sslmode":              "disable",
			"target_session_attrs": "read-write",
			"connect_timeout":      "10",
		},
	})
}

func TestIntegrationPostgresCluster(t *testing.T) {
	// a real node plus an unreachable one: pgx must traverse the host list
	// and still connect
	mustOpen(t, dbkit.Config{
		Driver: dbkit.DriverPostgres,
		Mode:   dbkit.ModeCluster,
		Nodes: []dbkit.NodeConfig{
			{Host: envOr("DBKIT_TEST_PG_HOST", "127.0.0.1"), Port: 5432},
			{Host: "127.0.0.1", Port: 59999}, // deliberately unreachable
		},
		Username: envOr("DBKIT_TEST_PG_USER", "app_user"),
		Password: envOr("DBKIT_TEST_PG_PASSWORD", "your_password"),
		Database: envOr("DBKIT_TEST_PG_DATABASE", "app_db"),
		Options: map[string]string{
			"sslmode":              "disable",
			"target_session_attrs": "read-write",
			"connect_timeout":      "3",
		},
	})
}

func TestIntegrationDMStandalone(t *testing.T) {
	mustOpen(t, dbkit.Config{
		Driver:   dbkit.DriverDM,
		Mode:     dbkit.ModeStandalone,
		Host:     envOr("DBKIT_TEST_DM_HOST", "127.0.0.1"),
		Port:     25236,
		Username: envOr("DBKIT_TEST_DM_USER", "SYSDBA"),
		Password: envOr("DBKIT_TEST_DM_PASSWORD", "SYSDBA001"),
		Database: envOr("DBKIT_TEST_DM_DATABASE", "SYSDBA"),
		Options: map[string]string{
			"switchTimes": "3",
		},
	})
}

func TestIntegrationDMCluster(t *testing.T) {
	// an unreachable node plus the real one: the driver's traverseServerList
	// must skip the dead node and connect, proving the dynamic service-name
	// multi-node DSN really works
	mustOpen(t, dbkit.Config{
		Driver: dbkit.DriverDM,
		Mode:   dbkit.ModeCluster,
		Nodes: []dbkit.NodeConfig{
			{Host: "127.0.0.1", Port: 25999}, // deliberately unreachable
			{Host: envOr("DBKIT_TEST_DM_HOST", "127.0.0.1"), Port: 25236},
		},
		Username: envOr("DBKIT_TEST_DM_USER", "SYSDBA"),
		Password: envOr("DBKIT_TEST_DM_PASSWORD", "SYSDBA001"),
		Database: envOr("DBKIT_TEST_DM_DATABASE", "SYSDBA"),
		Options: map[string]string{
			"epSelector":     "0",
			"switchTimes":    "2",
			"switchInterval": "200",
		},
	})
}

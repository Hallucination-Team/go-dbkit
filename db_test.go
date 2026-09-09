package dbkit

import (
	"errors"
	"strings"
	"testing"
)

func TestOpenUnknownDriver(t *testing.T) {
	_, err := Open(Config{
		Driver: "mysql", Mode: ModeStandalone, Host: "h", Port: 1,
		Username: "u", Password: "p", Database: "d",
	})
	if !errors.Is(err, ErrUnknownDriver) {
		t.Fatalf("want ErrUnknownDriver, got: %v", err)
	}
	// Registry.Lookup lists registered names sorted (Task 2 driver.go):
	// "dm, postgres".
	if !strings.Contains(err.Error(), "dm, postgres") {
		t.Fatalf("error should list registered drivers, got: %v", err)
	}
}

func TestOpenBuiltinDriversRegistered(t *testing.T) {
	for _, name := range []string{DriverPostgres, DriverDM} {
		if _, err := defaultRegistry.Lookup(name); err != nil {
			t.Fatalf("built-in driver %q not registered: %v", name, err)
		}
	}
}

func TestOpenValidPostgresConfig(t *testing.T) {
	// sql.Open connects lazily: a valid DSN returns immediately with no I/O
	database, err := Open(Config{
		Driver: DriverPostgres, Mode: ModeStandalone, Host: "127.0.0.1", Port: 5432,
		Username: "u", Password: "p", Database: "d",
		Options: map[string]string{"sslmode": "disable"},
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if database == nil {
		t.Fatal("want a non-nil *sql.DB")
	}
	database.Close()
}

func TestOpenValidDMConfig(t *testing.T) {
	database, err := Open(Config{
		Driver: DriverDM, Mode: ModeStandalone, Host: "127.0.0.1", Port: 5236,
		Username: "u", Password: "p", Database: "d",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if database == nil {
		t.Fatal("want a non-nil *sql.DB")
	}
	database.Close()
}

func TestOpenAggregatedValidationErrors(t *testing.T) {
	_, err := Open(Config{Driver: DriverPostgres, Mode: ModeStandalone, Port: 99999})
	if err == nil || !strings.Contains(err.Error(), "port") {
		t.Fatalf("want a port validation error, got: %v", err)
	}
}

func TestOpenCustomRegistryDuplicate(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(PostgresDriver{}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	err := r.Register(PostgresDriver{})
	if !errors.Is(err, ErrDuplicateDriver) {
		t.Fatalf("want ErrDuplicateDriver, got: %v", err)
	}
}

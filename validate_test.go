package dbkit

import (
	"errors"
	"strings"
	"testing"
)

// testRegistry is a local registry with the "postgres" name pre-registered by
// fakeDriver. Validate tests must not depend on the package default registry,
// whose built-in drivers are wired in db.go's init (Task 7, later than this
// task).
var testRegistry = func() *Registry {
	r := NewRegistry()
	if err := r.Register(fakeDriver{name: DriverPostgres}); err != nil {
		panic(err)
	}
	return r
}()

func validStandalone() Config {
	return Config{
		Driver: DriverPostgres, Mode: ModeStandalone,
		Host: "10.0.0.1", Port: 5432,
		Username: "app", Password: "pw", Database: "myapp",
	}
}

func validCluster() Config {
	return Config{
		Driver: DriverPostgres, Mode: ModeCluster,
		Nodes: []NodeConfig{{Host: "10.0.0.1", Port: 5432}, {Host: "10.0.0.2", Port: 5432}},
		Username: "app", Password: "pw", Database: "myapp",
	}
}

func TestValidateStandaloneOK(t *testing.T) {
	if err := validateConfig(validStandalone(), testRegistry); err != nil {
		t.Fatalf("valid standalone config should not fail: %v", err)
	}
}

func TestValidateClusterOK(t *testing.T) {
	if err := validateConfig(validCluster(), testRegistry); err != nil {
		t.Fatalf("valid cluster config should not fail: %v", err)
	}
}

func TestValidateStandaloneMissingHost(t *testing.T) {
	cfg := validStandalone()
	cfg.Host = ""
	err := validateConfig(cfg, testRegistry)
	if err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("want a missing-host error, got: %v", err)
	}
}

func TestValidateStandaloneWithNodes(t *testing.T) {
	cfg := validStandalone()
	cfg.Nodes = []NodeConfig{{Host: "10.0.0.1", Port: 5432}}
	err := validateConfig(cfg, testRegistry)
	if err == nil || !strings.Contains(err.Error(), "nodes") {
		t.Fatalf("standalone with nodes should fail, got: %v", err)
	}
}

func TestValidateClusterMissingNodes(t *testing.T) {
	cfg := validCluster()
	cfg.Nodes = nil
	err := validateConfig(cfg, testRegistry)
	if err == nil || !strings.Contains(err.Error(), "nodes") {
		t.Fatalf("cluster without nodes should fail, got: %v", err)
	}
}

func TestValidateClusterNodeInvalid(t *testing.T) {
	cfg := validCluster()
	cfg.Nodes[0].Host = ""
	cfg.Nodes[1].Port = 0
	err := validateConfig(cfg, testRegistry)
	if err == nil {
		t.Fatal("invalid node fields should fail")
	}
	if !strings.Contains(err.Error(), "nodes[0].host") || !strings.Contains(err.Error(), "nodes[1].port") {
		t.Fatalf("error should name the exact node fields, got: %v", err)
	}
}

func TestValidateInvalidPort(t *testing.T) {
	cfg := validStandalone()
	cfg.Port = 0
	err := validateConfig(cfg, testRegistry)
	if err == nil || !strings.Contains(err.Error(), "port") {
		t.Fatalf("want an invalid-port error, got: %v", err)
	}
	cfg.Port = 65536
	err = validateConfig(cfg, testRegistry)
	if err == nil || !strings.Contains(err.Error(), "port") {
		t.Fatalf("want an out-of-range port error, got: %v", err)
	}
}

func TestValidateInvalidMode(t *testing.T) {
	cfg := validStandalone()
	cfg.Mode = "ha"
	err := validateConfig(cfg, testRegistry)
	if err == nil || !strings.Contains(err.Error(), "mode") {
		t.Fatalf("want an invalid-mode error, got: %v", err)
	}
	cfg.Mode = ""
	err = validateConfig(cfg, testRegistry)
	if err == nil || !strings.Contains(err.Error(), "mode") {
		t.Fatalf("want an empty-mode error, got: %v", err)
	}
}

func TestValidateInvalidDriver(t *testing.T) {
	cfg := validStandalone()
	cfg.Driver = "mysql"
	err := validateConfig(cfg, testRegistry)
	if !errors.Is(err, ErrUnknownDriver) {
		t.Fatalf("want ErrUnknownDriver, got: %v", err)
	}
	cfg.Driver = ""
	err = validateConfig(cfg, testRegistry)
	if err == nil || !strings.Contains(err.Error(), "driver") {
		t.Fatalf("want an empty-driver error, got: %v", err)
	}
}

func TestValidateCredentialsAndDatabase(t *testing.T) {
	cfg := validStandalone()
	cfg.Username = ""
	cfg.Password = ""
	cfg.Database = ""
	err := validateConfig(cfg, testRegistry)
	if err == nil {
		t.Fatal("missing credentials and database should fail")
	}
	for _, field := range []string{"username", "password", "database"} {
		if !strings.Contains(err.Error(), field) {
			t.Fatalf("error should mention %q, got: %v", field, err)
		}
	}
}

func TestValidateAggregatesAllErrors(t *testing.T) {
	cfg := Config{Mode: ModeStandalone, Port: 99999}
	err := validateConfig(cfg, testRegistry)
	if err == nil {
		t.Fatal("should fail")
	}
	// one shot: empty driver, missing host, out-of-range port, missing credentials
	for _, want := range []string{"driver", "host", "port", "username", "password", "database"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("aggregated error should contain %q, got: %v", want, err)
		}
	}
}

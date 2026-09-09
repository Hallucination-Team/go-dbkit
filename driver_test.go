package dbkit

import (
	"errors"
	"strings"
	"testing"
)

// fakeDriver is a minimal Driver implementation used only by registry tests.
type fakeDriver struct{ name string }

func (f fakeDriver) Name() string          { return f.name }
func (f fakeDriver) SQLDriverName() string { return f.name }
func (f fakeDriver) BuildDSN(Config) (string, error) {
	return "", nil
}

func TestRegistryRegisterAndLookup(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(fakeDriver{name: "a"}); err != nil {
		t.Fatalf("Register(a): %v", err)
	}
	d, err := r.Lookup("a")
	if err != nil {
		t.Fatalf("Lookup(a): %v", err)
	}
	if d.Name() != "a" {
		t.Fatalf("Lookup returned %q, want a", d.Name())
	}
}

func TestRegistryDuplicateRegister(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(fakeDriver{name: "a"}); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	err := r.Register(fakeDriver{name: "a"})
	if !errors.Is(err, ErrDuplicateDriver) {
		t.Fatalf("want ErrDuplicateDriver, got %v", err)
	}
}

func TestRegistryLookupUnknown(t *testing.T) {
	r := NewRegistry()
	_, err := r.Lookup("nope")
	if !errors.Is(err, ErrUnknownDriver) {
		t.Fatalf("want ErrUnknownDriver, got %v", err)
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Fatalf("error should mention the driver name, got: %v", err)
	}
}

func TestRegistryRegisterNil(t *testing.T) {
	r := NewRegistry()
	var d Driver
	if err := r.Register(d); err == nil {
		t.Fatal("registering a nil driver should fail")
	}
}

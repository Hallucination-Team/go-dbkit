package dbkit

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Driver is the adapter interface for a database driver. Implementations
// convert a Config into the driver's native DSN. BuildDSN is a pure function
// and performs no I/O; DSN format and parameter semantics follow each
// driver's official documentation.
type Driver interface {
	// Name is the driver identifier in configuration (Registry key),
	// e.g. postgres, dm.
	Name() string
	// SQLDriverName is the database/sql registered driver name, e.g. pgx, dm.
	// PostgreSQL differs between the two, hence the separate method.
	SQLDriverName() string
	// BuildDSN builds the DSN; the given Config is assumed to be validated.
	BuildDSN(cfg Config) (string, error)
}

var (
	// ErrUnknownDriver is returned when the requested driver is not registered.
	ErrUnknownDriver = errors.New("dbkit: unknown driver")
	// ErrDuplicateDriver is returned when registering a driver name twice.
	ErrDuplicateDriver = errors.New("dbkit: duplicate driver")
)

// Registry is a concurrency-safe driver registry.
type Registry struct {
	mu      sync.RWMutex
	drivers map[string]Driver
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{drivers: make(map[string]Driver)}
}

// Register registers a driver; a duplicate name returns ErrDuplicateDriver.
func (r *Registry) Register(d Driver) error {
	if d == nil {
		return errors.New("dbkit: cannot register a nil driver")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.drivers[d.Name()]; ok {
		return fmt.Errorf("%w: %q", ErrDuplicateDriver, d.Name())
	}
	r.drivers[d.Name()] = d
	return nil
}

// Lookup finds a driver by name. An unregistered name returns
// ErrUnknownDriver with the list of currently registered drivers.
func (r *Registry) Lookup(name string) (Driver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.drivers[name]
	if !ok {
		return nil, fmt.Errorf("%w %q, registered: %s", ErrUnknownDriver, name, strings.Join(r.namesLocked(), ", "))
	}
	return d, nil
}

// namesLocked returns the sorted driver names; the caller must hold the lock.
func (r *Registry) namesLocked() []string {
	names := make([]string, 0, len(r.drivers))
	for n := range r.drivers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// defaultRegistry is the package-level registry used by Open; built-in
// drivers are pre-registered in db.go's init.
var defaultRegistry = NewRegistry()

// Register registers a custom driver into the default registry (extension point).
func Register(d Driver) error {
	return defaultRegistry.Register(d)
}

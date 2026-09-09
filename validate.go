package dbkit

import (
	"errors"
	"fmt"
)

// validateConfig checks the whole configuration before any connection attempt
// and aggregates every problem it finds with errors.Join, so callers can fix
// everything in one round. Error messages name the exact fields and reasons.
func validateConfig(cfg Config, reg *Registry) error {
	var errs []error

	switch cfg.Driver {
	case "":
		errs = append(errs, errors.New("dbkit: driver must not be empty"))
	default:
		if _, err := reg.Lookup(cfg.Driver); err != nil {
			errs = append(errs, err)
		}
	}

	switch cfg.Mode {
	case ModeStandalone:
		if cfg.Host == "" {
			errs = append(errs, errors.New("dbkit: mode=standalone requires host"))
		}
		if cfg.Port < 1 || cfg.Port > 65535 {
			errs = append(errs, fmt.Errorf("dbkit: port=%d out of range [1,65535]", cfg.Port))
		}
		if len(cfg.Nodes) > 0 {
			errs = append(errs, errors.New("dbkit: mode=standalone does not allow nodes"))
		}
	case ModeCluster:
		if len(cfg.Nodes) == 0 {
			errs = append(errs, errors.New("dbkit: mode=cluster requires nodes"))
		}
		for i, n := range cfg.Nodes {
			if n.Host == "" {
				errs = append(errs, fmt.Errorf("dbkit: nodes[%d].host must not be empty", i))
			}
			if n.Port < 1 || n.Port > 65535 {
				errs = append(errs, fmt.Errorf("dbkit: nodes[%d].port=%d out of range [1,65535]", i, n.Port))
			}
		}
	case "":
		errs = append(errs, errors.New("dbkit: mode must not be empty"))
	default:
		errs = append(errs, fmt.Errorf("dbkit: invalid mode %q, must be standalone or cluster", cfg.Mode))
	}

	if cfg.Username == "" {
		errs = append(errs, errors.New("dbkit: username must not be empty"))
	}
	if cfg.Password == "" {
		errs = append(errs, errors.New("dbkit: password must not be empty"))
	}
	if cfg.Database == "" {
		errs = append(errs, errors.New("dbkit: database must not be empty"))
	}

	return errors.Join(errs...)
}

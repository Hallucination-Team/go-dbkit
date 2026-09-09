package dbkit

import (
	"fmt"
	"sort"
	"strings"
)

// PostgresDriver adapts PostgreSQL, backed by pgx/v5 (stdlib registers "pgx").
//
// Cluster mode uses pgx's native multi-host DSN (host=h1,h2 port=p1,p2);
// node selection, target_session_attrs filtering and failover are handled by
// pgx itself. This library performs no cluster management.
type PostgresDriver struct{}

// Name returns the driver identifier used in configuration.
func (PostgresDriver) Name() string { return DriverPostgres }

// SQLDriverName returns the database/sql registered driver name.
func (PostgresDriver) SQLDriverName() string { return "pgx" }

// postgresReservedKeys lists keys not allowed in options (they clash with
// core fields or base DSN keys). libpq keywords are case-insensitive, so
// matching lower-cases the key first.
var postgresReservedKeys = map[string]bool{
	"host": true, "port": true, "user": true, "password": true,
	"dbname": true, "database": true,
}

// BuildDSN builds a libpq keyword=value DSN:
//
//	host=h1[,h2...] port=p1[,p2...] user=... password=... dbname=... <options sorted by key>
func (PostgresDriver) BuildDSN(cfg Config) (string, error) {
	for k := range cfg.Options {
		if postgresReservedKeys[strings.ToLower(k)] {
			return "", fmt.Errorf("dbkit: postgres: options must not contain reserved key %q", k)
		}
	}

	hosts := make([]string, 0, 2)
	ports := make([]string, 0, 2)
	if cfg.Mode == ModeCluster {
		for _, n := range cfg.Nodes {
			hosts = append(hosts, n.Host)
			ports = append(ports, fmt.Sprintf("%d", n.Port))
		}
	} else {
		hosts = append(hosts, cfg.Host)
		ports = append(ports, fmt.Sprintf("%d", cfg.Port))
	}

	parts := []string{
		"host=" + escapeLibpqValue(strings.Join(hosts, ",")),
		"port=" + strings.Join(ports, ","),
		"user=" + escapeLibpqValue(cfg.Username),
		"password=" + escapeLibpqValue(cfg.Password),
		"dbname=" + escapeLibpqValue(cfg.Database),
	}

	keys := make([]string, 0, len(cfg.Options))
	for k := range cfg.Options {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, k+"="+escapeLibpqValue(cfg.Options[k]))
	}
	return strings.Join(parts, " "), nil
}

// escapeLibpqValue escapes a value per libpq rules: values containing spaces,
// single quotes or backslashes are wrapped in single quotes, and internal
// backslashes and single quotes are escaped with a backslash.
func escapeLibpqValue(v string) string {
	if v == "" {
		return ""
	}
	if !strings.ContainsAny(v, " \t\n'\\") {
		return v
	}
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return `'` + v + `'`
}

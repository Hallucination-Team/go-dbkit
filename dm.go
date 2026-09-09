package dbkit

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// DMDriver adapts Dameng DM, backed by the official Go driver
// (third_party/dm, registered as "dm").
//
// Cluster mode uses the driver's native dynamic service name mechanism: the
// DSN host position holds a service-name placeholder and the node list is
// passed as a query parameter of the same name. Node traversal, loginMode
// primary/standby preference and switchTimes/switchInterval retries are all
// handled by the driver. This library performs no cluster management.
type DMDriver struct{}

// Name returns the driver identifier used in configuration.
func (DMDriver) Name() string { return DriverDM }

// SQLDriverName returns the database/sql registered driver name.
func (DMDriver) SQLDriverName() string { return "dm" }

// dmClusterServiceName is the service-name placeholder in the host position
// of cluster DSNs (a fixed constant, not configurable).
const dmClusterServiceName = "dbkit_svc"

// dmReservedKeys lists keys not allowed in options (they clash with core
// fields or the cluster service-name parameter; case-sensitive).
var dmReservedKeys = map[string]bool{
	"host": true, "port": true, "user": true, "password": true,
	dmClusterServiceName: true,
}

// BuildDSN builds a native DM DSN:
//
//	standalone: dm://user:pass@host:port?<params sorted by key>
//	cluster:    dm://user:pass@dbkit_svc?dbkit_svc=(h1:p1,h2:p2)&<params sorted by key>
//
// cfg.Database maps to the driver's schema connection parameter (a DM
// instance holds one database; schemas are the namespaces). The driver
// ignores a DSN path in cluster mode, while the schema parameter works in
// both modes. An explicit schema option wins over the injected value.
//
// DSN escaping limits (the driver's parser does not URL-decode, see
// third_party/dm n.go parseDSN): username/password must not contain '?';
// option keys must not contain '&', '=' or '?'; option values must not
// contain '&' or '?'. '@' ':' '&' '=' are safe in username/password, which
// the driver parses via LastIndex("@") + SplitN(":", 2).
func (DMDriver) BuildDSN(cfg Config) (string, error) {
	if strings.ContainsRune(cfg.Username, '?') {
		return "", fmt.Errorf("dbkit: dm: username must not contain '?' (DM DSN parsing limit)")
	}
	if strings.ContainsRune(cfg.Password, '?') {
		return "", fmt.Errorf("dbkit: dm: password must not contain '?' (DM DSN parsing limit)")
	}

	opts := make(map[string]string, len(cfg.Options)+1)
	for k, v := range cfg.Options {
		if dmReservedKeys[k] {
			return "", fmt.Errorf("dbkit: dm: options must not contain reserved key %q", k)
		}
		if strings.ContainsAny(k, "&=?") {
			return "", fmt.Errorf("dbkit: dm: options key %q must not contain '&', '=' or '?'", k)
		}
		if strings.ContainsAny(v, "&?") {
			return "", fmt.Errorf("dbkit: dm: options[%s] value must not contain '&' or '?'", k)
		}
		opts[k] = v
	}

	// map database to the schema parameter; an explicit schema option wins
	if _, ok := opts["schema"]; !ok {
		if strings.ContainsRune(cfg.Database, '?') {
			return "", fmt.Errorf("dbkit: dm: database must not contain '?' (DM DSN parsing limit)")
		}
		opts["schema"] = cfg.Database
	}

	var hostPart string
	if cfg.Mode == ModeCluster {
		nodes := make([]string, 0, len(cfg.Nodes))
		for _, n := range cfg.Nodes {
			nodes = append(nodes, formatDMNode(n.Host, n.Port))
		}
		hostPart = dmClusterServiceName
		opts[dmClusterServiceName] = "(" + strings.Join(nodes, ",") + ")"
	} else {
		hostPart = formatDMNode(cfg.Host, cfg.Port)
	}

	keys := make([]string, 0, len(opts))
	for k := range opts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+opts[k])
	}

	dsn := "dm://" + cfg.Username + ":" + cfg.Password + "@" + hostPart
	if len(pairs) > 0 {
		dsn += "?" + strings.Join(pairs, "&")
	}
	return dsn, nil
}

// formatDMNode formats a single host:port node; IPv6 addresses get brackets
// (required by the driver's SplitHostPort).
func formatDMNode(host string, port int) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return host + ":" + strconv.Itoa(port)
}

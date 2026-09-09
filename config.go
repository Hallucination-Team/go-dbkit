// Package dbkit provides unified database connection initialization for
// PostgreSQL and Dameng DM.
//
// The single entry point for applications:
//
//	db, err := dbkit.Open(config)
//
// It returns a standard *sql.DB; connection pooling is managed by
// database/sql. This library implements neither pooling nor node health
// checks nor failover - those are provided by the database drivers or the
// databases' own HA mechanisms.
package dbkit

// Valid mode values in configuration.
const (
	ModeStandalone = "standalone"
	ModeCluster    = "cluster"
)

// Driver identifiers in configuration (Registry keys).
const (
	DriverPostgres = "postgres"
	DriverDM       = "dm"
)

// Config describes a database connection. Fields map one-to-one to the YAML
// structure; YAML decoding is the application's responsibility (this library
// does not depend on yaml).
type Config struct {
	Driver string `yaml:"driver"` // postgres | dm
	Mode   string `yaml:"mode"`   // standalone | cluster

	Host  string       `yaml:"host"`  // required by standalone
	Port  int          `yaml:"port"`  // required by standalone, [1,65535]
	Nodes []NodeConfig `yaml:"nodes"` // required by cluster

	Options map[string]string `yaml:"options"` // driver-specific passthrough

	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// NodeConfig is a single node address in cluster mode.
type NodeConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

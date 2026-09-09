package dbkit

import (
	"database/sql"

	// official DM driver, registers "dm" via init()
	_ "github.com/Hallucination-Team/go-dbkit/third_party/dm"
	// pgx/v5 stdlib, registers "pgx" via init()
	_ "github.com/jackc/pgx/v5/stdlib"
)

func init() {
	// pre-register built-in drivers; duplicate registration is a programming
	// error, so panic to surface it
	for _, d := range []Driver{PostgresDriver{}, DMDriver{}} {
		if err := defaultRegistry.Register(d); err != nil {
			panic(err)
		}
	}
}

// Open validates the configuration, builds the DSN via the selected driver
// and returns a standard *sql.DB.
//
// Connections are established lazily (sql.Open semantics): configuration
// errors surface here, connectivity errors surface on first use or on
// db.Ping.
func Open(cfg Config) (*sql.DB, error) {
	if err := validateConfig(cfg, defaultRegistry); err != nil {
		return nil, err
	}
	d, err := defaultRegistry.Lookup(cfg.Driver)
	if err != nil {
		return nil, err
	}
	dsn, err := d.BuildDSN(cfg)
	if err != nil {
		return nil, err
	}
	return sql.Open(d.SQLDriverName(), dsn)
}

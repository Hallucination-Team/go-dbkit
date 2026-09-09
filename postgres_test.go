package dbkit

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPostgresDriverNames(t *testing.T) {
	d := PostgresDriver{}
	if d.Name() != DriverPostgres {
		t.Fatalf("Name() = %q, want %q", d.Name(), DriverPostgres)
	}
	if d.SQLDriverName() != "pgx" {
		t.Fatalf("SQLDriverName() = %q, want pgx", d.SQLDriverName())
	}
}

func TestPostgresBuildDSNStandalone(t *testing.T) {
	d := PostgresDriver{}
	got, err := d.BuildDSN(Config{
		Mode: ModeStandalone, Host: "10.10.1.101", Port: 5432,
		Username: "app", Password: "password", Database: "myapp",
		Options: map[string]string{"target_session_attrs": "read-write"},
	})
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	want := "host=10.10.1.101 port=5432 user=app password=password dbname=myapp target_session_attrs=read-write"
	if got != want {
		t.Fatalf("DSN mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestPostgresBuildDSNCluster(t *testing.T) {
	d := PostgresDriver{}
	got, err := d.BuildDSN(Config{
		Mode: ModeCluster,
		Nodes: []NodeConfig{
			{Host: "10.10.1.101", Port: 5432},
			{Host: "10.10.1.102", Port: 5433},
		},
		Username: "app_user", Password: "password", Database: "app",
		Options: map[string]string{
			"target_session_attrs": "read-write",
			"connect_timeout":      "10",
		},
	})
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	want := "host=10.10.1.101,10.10.1.102 port=5432,5433 user=app_user password=password dbname=app " +
		"connect_timeout=10 target_session_attrs=read-write"
	if got != want {
		t.Fatalf("DSN mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestPostgresOptionsSortedDeterministic(t *testing.T) {
	d := PostgresDriver{}
	base := Config{
		Mode: ModeStandalone, Host: "h", Port: 5432,
		Username: "u", Password: "p", Database: "d",
	}
	cfgA := base
	cfgA.Options = map[string]string{"b_opt": "1", "a_opt": "2"}
	cfgB := base
	cfgB.Options = map[string]string{"a_opt": "2", "b_opt": "1"}
	gotA, _ := d.BuildDSN(cfgA)
	gotB, _ := d.BuildDSN(cfgB)
	if gotA != gotB {
		t.Fatalf("same options in different insertion order must produce the same DSN:\n%s\n%s", gotA, gotB)
	}
	if !strings.Contains(gotA, "a_opt=2 b_opt=1") {
		t.Fatalf("options must be emitted sorted by key: %s", gotA)
	}
}

func TestPostgresReservedKeyRejected(t *testing.T) {
	d := PostgresDriver{}
	for _, key := range []string{"host", "port", "user", "password", "dbname", "database", "HOST", "DBNAME"} {
		cfg := validStandalone()
		cfg.Options = map[string]string{key: "x"}
		_, err := d.BuildDSN(cfg)
		if err == nil || !strings.Contains(err.Error(), "reserved key") {
			t.Fatalf("reserved key %q must be rejected, got: %v", key, err)
		}
	}
}

func TestPostgresEscapeLibpqValue(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "plain"},
		{"", ""},
		{"with space", `'with space'`},
		{"it's", `'it\'s'`},
		{`back\slash`, `'back\\slash'`},
		{"p@ss:word", "p@ss:word"}, // no quoting needed for these characters
	}
	for _, c := range cases {
		if got := escapeLibpqValue(c.in); got != c.want {
			t.Fatalf("escapeLibpqValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPostgresPasswordEscaped(t *testing.T) {
	d := PostgresDriver{}
	got, err := d.BuildDSN(Config{
		Mode: ModeStandalone, Host: "h", Port: 5432,
		Username: "u", Password: `pa ss'wo\rd`, Database: "d",
	})
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	if !strings.Contains(got, `password='pa ss\'wo\\rd'`) {
		t.Fatalf("password not escaped correctly: %s", got)
	}
}

// pgx.ParseConfig only parses and never connects; use it to prove DSN validity.
func TestPostgresDSNParseableByPgx(t *testing.T) {
	d := PostgresDriver{}
	cases := []struct {
		name      string
		cfg       Config
		wantHosts int
	}{
		{
			name: "standalone",
			cfg: Config{
				Mode: ModeStandalone, Host: "127.0.0.1", Port: 5432,
				Username: "u", Password: "p", Database: "d",
				Options: map[string]string{"sslmode": "disable"},
			},
			wantHosts: 1,
		},
		{
			name: "cluster",
			cfg: Config{
				Mode: ModeCluster,
				Nodes: []NodeConfig{
					{Host: "127.0.0.1", Port: 5432},
					{Host: "127.0.0.2", Port: 5433},
				},
				Username: "u", Password: "p", Database: "d",
			},
			wantHosts: 2,
		},
		{
			name: "special-password",
			cfg: Config{
				Mode: ModeStandalone, Host: "127.0.0.1", Port: 5432,
				Username: "u", Password: `pa ss'wo\rd`, Database: "d",
			},
			wantHosts: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dsn, err := d.BuildDSN(tc.cfg)
			if err != nil {
				t.Fatalf("BuildDSN: %v", err)
			}
			parsed, err := pgx.ParseConfig(dsn)
			if err != nil {
				t.Fatalf("pgx.ParseConfig(%q): %v", dsn, err)
			}
			// pgx v5 keeps only the first host in Config.Host (a string);
			// the remaining hosts live in Config.Fallbacks.
			gotHosts := map[string]bool{parsed.Host: true}
			for _, fb := range parsed.Fallbacks {
				gotHosts[fb.Host] = true
			}
			if len(gotHosts) != tc.wantHosts {
				t.Fatalf("parsed %d hosts, want %d", len(gotHosts), tc.wantHosts)
			}
		})
	}
}

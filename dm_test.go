package dbkit

import (
	"strings"
	"testing"
)

func TestDMDriverNames(t *testing.T) {
	d := DMDriver{}
	if d.Name() != DriverDM || d.SQLDriverName() != "dm" {
		t.Fatalf("Name/SQLDriverName mismatch: %q/%q", d.Name(), d.SQLDriverName())
	}
}

func TestDMBuildDSNStandalone(t *testing.T) {
	d := DMDriver{}
	got, err := d.BuildDSN(Config{
		Mode: ModeStandalone, Host: "10.10.1.101", Port: 5236,
		Username: "app", Password: "password", Database: "myapp",
	})
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	want := "dm://app:password@10.10.1.101:5236?schema=myapp"
	if got != want {
		t.Fatalf("DSN mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestDMBuildDSNStandaloneWithOptions(t *testing.T) {
	d := DMDriver{}
	got, err := d.BuildDSN(Config{
		Mode: ModeStandalone, Host: "10.10.1.101", Port: 5236,
		Username: "app", Password: "password", Database: "myapp",
		Options: map[string]string{
			"epSelector":     "1",
			"switchTimes":    "3",
			"switchInterval": "200",
		},
	})
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	want := "dm://app:password@10.10.1.101:5236?epSelector=1&schema=myapp&switchInterval=200&switchTimes=3"
	if got != want {
		t.Fatalf("DSN mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestDMBuildDSNCluster(t *testing.T) {
	d := DMDriver{}
	got, err := d.BuildDSN(Config{
		Mode: ModeCluster,
		Nodes: []NodeConfig{
			{Host: "10.10.1.101", Port: 5236},
			{Host: "10.10.1.102", Port: 5236},
		},
		Username: "app_user", Password: "password", Database: "app",
		Options: map[string]string{"epSelector": "1"},
	})
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	want := "dm://app_user:password@dbkit_svc?dbkit_svc=(10.10.1.101:5236,10.10.1.102:5236)&epSelector=1&schema=app"
	if got != want {
		t.Fatalf("DSN mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestDMBuildDSNClusterIPv6Node(t *testing.T) {
	d := DMDriver{}
	got, err := d.BuildDSN(Config{
		Mode: ModeCluster,
		Nodes: []NodeConfig{
			{Host: "::1", Port: 5236},
			{Host: "10.10.1.102", Port: 5236},
		},
		Username: "u", Password: "p", Database: "d",
	})
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	if !strings.Contains(got, "[::1]:5236") {
		t.Fatalf("IPv6 node must be bracketed: %s", got)
	}
}

func TestDMReservedKeyRejected(t *testing.T) {
	d := DMDriver{}
	for _, key := range []string{"host", "port", "user", "password", "dbkit_svc"} {
		cfg := Config{
			Mode: ModeStandalone, Host: "h", Port: 5236,
			Username: "u", Password: "p", Database: "d",
			Options: map[string]string{key: "x"},
		}
		_, err := d.BuildDSN(cfg)
		if err == nil || !strings.Contains(err.Error(), "reserved key") {
			t.Fatalf("reserved key %q must be rejected, got: %v", key, err)
		}
	}
}

func TestDMSpecialCharsRejected(t *testing.T) {
	d := DMDriver{}
	cases := []struct {
		name string
		cfg  Config
	}{
		{
			"password contains ?",
			Config{Mode: ModeStandalone, Host: "h", Port: 5236, Username: "u", Password: "p?w", Database: "d"},
		},
		{
			"username contains ?",
			Config{Mode: ModeStandalone, Host: "h", Port: 5236, Username: "u?x", Password: "p", Database: "d"},
		},
		{
			"option key contains &",
			Config{Mode: ModeStandalone, Host: "h", Port: 5236, Username: "u", Password: "p", Database: "d",
				Options: map[string]string{"a&b": "1"}},
		},
		{
			"option key contains =",
			Config{Mode: ModeStandalone, Host: "h", Port: 5236, Username: "u", Password: "p", Database: "d",
				Options: map[string]string{"a=b": "1"}},
		},
		{
			"option value contains &",
			Config{Mode: ModeStandalone, Host: "h", Port: 5236, Username: "u", Password: "p", Database: "d",
				Options: map[string]string{"k": "1&2"}},
		},
		{
			"option value contains ?",
			Config{Mode: ModeStandalone, Host: "h", Port: 5236, Username: "u", Password: "p", Database: "d",
				Options: map[string]string{"k": "1?2"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := d.BuildDSN(tc.cfg)
			if err == nil {
				t.Fatal("inputs with forbidden special characters must be rejected")
			}
		})
	}
}

func TestDMPasswordSpecialCharsAllowed(t *testing.T) {
	// @ : & = are safe in the userinfo position (the driver parses with
	// LastIndex("@") + SplitN(":", 2))
	d := DMDriver{}
	got, err := d.BuildDSN(Config{
		Mode: ModeStandalone, Host: "h", Port: 5236,
		Username: "u", Password: `p@a:ss&w=rd`, Database: "d",
	})
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	want := "dm://u:p@a:ss&w=rd@h:5236?schema=d"
	if got != want {
		t.Fatalf("DSN mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestDMSchemaOptionOverridesDatabase(t *testing.T) {
	d := DMDriver{}
	got, err := d.BuildDSN(Config{
		Mode: ModeStandalone, Host: "h", Port: 5236,
		Username: "u", Password: "p", Database: "d",
		Options: map[string]string{"schema": "OTHER"},
	})
	if err != nil {
		t.Fatalf("BuildDSN: %v", err)
	}
	if strings.Count(got, "schema=") != 1 || !strings.Contains(got, "schema=OTHER") {
		t.Fatalf("explicit schema must override the database injection and appear exactly once: %s", got)
	}
}

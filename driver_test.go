package dbkit

import (
	"errors"
	"strings"
	"testing"
)

// fakeDriver 仅用于测试 Registry 的最小实现。
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
		t.Fatalf("Lookup 返回 %q，期望 a", d.Name())
	}
}

func TestRegistryDuplicateRegister(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(fakeDriver{name: "a"}); err != nil {
		t.Fatalf("第一次 Register: %v", err)
	}
	err := r.Register(fakeDriver{name: "a"})
	if !errors.Is(err, ErrDuplicateDriver) {
		t.Fatalf("期望 ErrDuplicateDriver，得到 %v", err)
	}
}

func TestRegistryLookupUnknown(t *testing.T) {
	r := NewRegistry()
	_, err := r.Lookup("nope")
	if !errors.Is(err, ErrUnknownDriver) {
		t.Fatalf("期望 ErrUnknownDriver，得到 %v", err)
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Fatalf("错误信息应包含 driver 名，得到: %v", err)
	}
}

func TestRegistryRegisterNil(t *testing.T) {
	r := NewRegistry()
	var d Driver
	if err := r.Register(d); err == nil {
		t.Fatal("注册 nil driver 应返回错误")
	}
}

package dbkit

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Driver 数据库驱动适配器。实现方负责把 Config 转换为驱动原生 DSN。
// BuildDSN 是纯函数、不做 IO；DSN 的具体格式与参数语义以各驱动的官方文档为准。
type Driver interface {
	// Name 配置中的 driver 标识（Registry 键），如 postgres、dm。
	Name() string
	// SQLDriverName database/sql 驱动注册名，如 pgx、dm。
	// PG 的两者不同，因此需要独立方法。
	SQLDriverName() string
	// BuildDSN 构建 DSN；假定传入的 Config 已通过校验。
	BuildDSN(cfg Config) (string, error)
}

var (
	// ErrUnknownDriver 请求的 driver 未注册。
	ErrUnknownDriver = errors.New("dbkit: 未知 driver")
	// ErrDuplicateDriver 注册了同名 driver。
	ErrDuplicateDriver = errors.New("dbkit: driver 已注册")
)

// Registry driver 注册表，并发安全。
type Registry struct {
	mu      sync.RWMutex
	drivers map[string]Driver
}

// NewRegistry 创建空的注册表。
func NewRegistry() *Registry {
	return &Registry{drivers: make(map[string]Driver)}
}

// Register 注册 driver，名称重复返回 ErrDuplicateDriver。
func (r *Registry) Register(d Driver) error {
	if d == nil {
		return errors.New("dbkit: 不能注册 nil driver")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.drivers[d.Name()]; ok {
		return fmt.Errorf("%w: %q", ErrDuplicateDriver, d.Name())
	}
	r.drivers[d.Name()] = d
	return nil
}

// Lookup 按名称查找 driver，未注册返回 ErrUnknownDriver，
// 错误信息中列出当前已注册的全部 driver 名。
func (r *Registry) Lookup(name string) (Driver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.drivers[name]
	if !ok {
		return nil, fmt.Errorf("%w %q，已注册: %s", ErrUnknownDriver, name, strings.Join(r.namesLocked(), ", "))
	}
	return d, nil
}

// namesLocked 返回排序后的 driver 名列表，调用方需已持有锁。
func (r *Registry) namesLocked() []string {
	names := make([]string, 0, len(r.drivers))
	for n := range r.drivers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// defaultRegistry 包级默认注册表，Open 使用它；内置 driver 在 db.go 的 init 中预注册。
var defaultRegistry = NewRegistry()

// Register 向默认注册表注册自定义 driver（扩展点）。
func Register(d Driver) error {
	return defaultRegistry.Register(d)
}

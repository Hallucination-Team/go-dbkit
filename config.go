// Package dbkit 统一 PostgreSQL 与达梦 DM 的数据库连接初始化。
//
// 业务应用唯一入口：
//
//	db, err := dbkit.Open(config)
//
// 返回标准 *sql.DB，连接池由 database/sql 管理。本库不实现连接池、
// 节点探活、failover 等能力，这些由数据库驱动或数据库自身 HA 机制提供。
package dbkit

// 配置中 mode 的合法取值。
const (
	ModeStandalone = "standalone"
	ModeCluster    = "cluster"
)

// 配置中 driver 的合法取值（Registry 的注册键）。
const (
	DriverPostgres = "postgres"
	DriverDM       = "dm"
)

// Config 数据库连接配置。字段与 YAML 结构一一对应，
// YAML 解码由应用侧完成（本库不引入 yaml 依赖）。
type Config struct {
	Driver string `yaml:"driver"` // postgres | dm
	Mode   string `yaml:"mode"`   // standalone | cluster

	Host  string       `yaml:"host"`  // standalone 必填
	Port  int          `yaml:"port"`  // standalone 必填，[1,65535]
	Nodes []NodeConfig `yaml:"nodes"` // cluster 必填

	Options map[string]string `yaml:"options"` // Driver-specific 参数透传

	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// NodeConfig 集群模式下的单个节点地址。
type NodeConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

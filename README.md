# go-dbkit

统一的 Go 数据库连接公共库：一条 `dbkit.Open(config)` 打通 PostgreSQL 与达梦 DM，
支持单节点（standalone）与集群（cluster）模式，返回标准 `*sql.DB`。

## 特性

- 基于 `database/sql`，连接池由标准库管理
- PostgreSQL：pgx/v5（stdlib），集群使用 pgx 原生多主机 DSN + target_session_attrs
- 达梦 DM：官方 Go 驱动（v8.1.5.60，re-home 于 third_party/dm），集群使用驱动原生动态服务名机制
- `options` 原样透传，无白名单、无默认值、不做参数翻译
- 配置校验：errors.Join 聚合所有问题一次性报出

## 职责边界（本库不做）

节点探活、健康检查、Primary 判断、Failover、节点权重、自定义负载均衡、
自定义重试、连接迁移、ORM、SQL Builder、自定义连接池。
这些能力交给数据库驱动或数据库自身的 HA 机制。

## 安装

要求 Go 1.25+（pgx v5.11.0 的要求）。

```bash
go get github.com/Hallucination-Team/go-dbkit
```

安装即用：PG 驱动（pgx/v5）与 DM 驱动（third_party/dm）均已在库内，`dbkit.Open` 无需额外 import 驱动包。

## 快速开始

`config.yaml`（完整文件见 [examples/postgres-standalone](examples/postgres-standalone/config.yaml)，顶层扁平结构）：

```yaml
driver: postgres
mode: standalone

host: 127.0.0.1
port: 5432

options:
  sslmode: disable
  target_session_attrs: read-write

username: app_user
password: "your_password"
database: app_db
```

Go 代码：`yaml.Unmarshal` → `dbkit.Open` → `Ping`（完整可运行示例见 [examples/postgres-standalone](examples/postgres-standalone/main.go)）：

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	dbkit "github.com/Hallucination-Team/go-dbkit"
	"gopkg.in/yaml.v3"
)

func main() {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	var cfg dbkit.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatal(err)
	}

	db, err := dbkit.Open(cfg) // 校验配置并构建 DSN
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("connected")
}
```

四种场景完整 YAML 配置示例分别见：
examples/postgres-standalone、examples/postgres-cluster、examples/dm-standalone、examples/dm-cluster。

## 配置说明

`Config` 字段与 YAML 结构一一对应（YAML 解析由应用负责，本库不依赖 yaml）：

| 字段 | 类型 | 说明 |
|---|---|---|
| `driver` | string | `postgres` \| `dm`，必填 |
| `mode` | string | `standalone` \| `cluster`，必填 |
| `host` | string | standalone 必填 |
| `port` | int | standalone 必填，范围 [1,65535] |
| `nodes` | []NodeConfig | cluster 必填，每项为 host + port |
| `options` | map[string]string | 驱动参数，原样透传（见下节） |
| `username` | string | 必填 |
| `password` | string | 必填 |
| `database` | string | 必填 |

校验规则（`Open` 前置校验，所有问题经 errors.Join 一次性报出）：

- standalone：必填 host/port，且禁止出现 `nodes`
- cluster：必填非空 `nodes`，每个节点 host 必填、port ∈ [1,65535]
- `port` 与节点 `port` 均须 ∈ [1,65535]；`driver`、`mode`、`username`、`password`、`database` 均不得为空

保留键规则（`options` 不得覆盖核心字段，命中即报错）：

- postgres：`host`、`port`、`user`、`password`、`dbname`、`database`（键不区分大小写）
- dm：`host`、`port`、`user`、`password`、`dbkit_svc`（键区分大小写；`dbkit_svc` 为 cluster 模式内部服务名占位参数）

## options 透传

- 公共层不做任何解释，键值原样传给驱动，语义以驱动官方文档为准
- 输出按键排序，保证 DSN 确定性

### PostgreSQL 常用参数（来源：pgx/libpq 文档）

target_session_attrs（read-write/read-only/primary/standby/prefer-standby）、
sslmode、connect_timeout、search_path 等。

### DM 常用参数（来源：官方 Go 驱动源码 third_party/dm）

| 参数 | 说明 |
|---|---|
| epSelector | 节点选择：0 均衡轮转（默认），1 顺序（头）优先 |
| loginMode | 主备优先级：0=主库优先，1=仅主库，2=仅备库，3=备库优先，4=普通库优先（默认） |
| switchTimes / switchInterval | 连接重试轮数 / 轮间隔 ms（默认 200） |
| doSwitch | 连接失效行为（默认 1：返回 driver.ErrBadConn 由 database/sql 重试） |
| driverReconnect | 连接失效时驱动自愈重连 |
| rwSeparate / rwPercent | 读写分离 |
| connectTimeout / socketTimeout | 连接/套接字超时 |

注意：JDBC 的 `autoReconnect` 在 Go 驱动中不存在，对应能力是 `doSwitch` + `driverReconnect`。

## 已知限制

- DM DSN 不做 URL 反转义：username/password 不能含 `?`；option key 不能含 `&` `=` `?`；option value 不能含 `&` `?`
- DM 各模式下 `database` 字段映射为驱动的 `schema` 连接参数（DM 实例内"库"即模式；options 显式配置 schema 时以 options 为准）
- PG 密码按 libpq 规则自动转义（空格/引号/反斜杠），无需手动处理
- PG options key 无字符校验：含空格的 key 会产出破损 DSN
- `sql.Open` 的错误未包裹，裸露出且无 `dbkit:` 前缀

## 开发

    go build ./... && go test ./...                     # 单元测试（不连库）
    go vet $(go list ./... | grep -v /third_party/)     # 静态检查（third_party 豁免：官方源码固有告警）
    gofmt -l . | grep -v '^third_party/'    # 格式检查（third_party 豁免：官方源码固有格式）
    go test -tags integration ./...                     # 集成测试（需本机 PG/DM 容器）

### 升级 DM 驱动（re-home 脚本）

DM 官方驱动以 vendored 方式维护在 [third_party/dm](third_party/dm)（import 改写为
`github.com/Hallucination-Team/go-dbkit/third_party/dm/*`，并移除其自带 go.mod/go.sum）。
官方发布新版驱动后，按以下步骤升级：

1. **获取官方驱动 zip**：从达梦官方发布包（安装介质）中取 Go 驱动压缩包。注意发行包常为
   外壳 zip（内含 `go/dm-go-driver.zip`、gorm 方言包等），脚本需传入**内层**的
   `dm-go-driver.zip`（其顶层为 `dm/` 目录，含 `go.mod`(module dm)、源码、LICENSE）。
2. **执行脚本**（幂等：自动清理旧的 third_party/dm 后重建）：

   ```bash
   chmod +x tools/rehome_dm_driver.sh
   tools/rehome_dm_driver.sh /path/to/dm-go-driver.zip
   ```

   脚本会：解压 → 重写全部 `dm/*` import 为本库路径 → 移除其 go.mod/go.sum →
   保留官方 LICENSE/CHANGELOG.md/VERSION → 校验无残留 import，任一环节失败即报错退出。
3. **收尾验证与提交**：

   ```bash
   go mod tidy && go build ./... && go test ./...
   git add third_party tools/rehome_dm_driver.sh go.mod go.sum
   git commit   # 建议注明官方驱动版本号，见 third_party/dm/VERSION
   ```

升级后新驱动是否改变保留键/禁则行为，重跑 `go test ./...` 与集成测试确认。

集成测试读取环境变量 `DBKIT_TEST_PG_HOST` / `DBKIT_TEST_PG_USER` / `DBKIT_TEST_PG_PASSWORD` /
`DBKIT_TEST_PG_DATABASE` 与 `DBKIT_TEST_DM_HOST` / `DBKIT_TEST_DM_USER` / `DBKIT_TEST_DM_PASSWORD` /
`DBKIT_TEST_DM_DATABASE`，未设置时使用 integration_test.go 中的默认值
（容器网络场景 host 通常填宿主网关，即 Docker 默认网段网关，实际地址以部署环境为准）。

## License

MIT（third_party/dm 保留达梦官方 LICENSE）

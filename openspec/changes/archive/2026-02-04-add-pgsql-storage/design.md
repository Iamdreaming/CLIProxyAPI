## Context

当前系统通过 `sdk/cliproxy/usage.Plugin` 接口收集使用记录，由 `internal/usage.LoggerPlugin` 聚合到内存中的 `RequestStatistics` 结构。这种设计简单高效，但数据在服务重启后丢失。

项目已有 `github.com/jackc/pgx/v5` 依赖，可直接用于 PostgreSQL 连接。

现有插件架构支持注册多个 Plugin 实现，新增 PostgreSQL 存储只需实现 `usage.Plugin` 接口并注册即可，无需修改核心代码。

## Goals / Non-Goals

**Goals:**
- 实现 PostgreSQL 存储后端，持久化每条请求记录
- 支持聚合查询（按日/小时/API/模型维度）
- 提供配置选项控制 PostgreSQL 存储的启用和连接参数
- 保持与现有管理 API 的兼容性
- 支持数据库连接池和重连机制

**Non-Goals:**
- 不替换内存存储（保留作为默认选项）
- 不支持其他数据库（MySQL、SQLite 等）
- 不实现数据清理/归档策略（可后续添加）
- 不提供数据迁移工具（从内存导入到 PostgreSQL）

## Decisions

### 1. 使用 pgx/v5 作为 PostgreSQL 驱动

**选择**: `github.com/jackc/pgx/v5`

**原因**:
- 项目已有此依赖，无需新增
- pgx 是 Go 生态最成熟的 PostgreSQL 驱动，性能优秀
- 内置连接池 (`pgxpool`)，无需额外依赖

**备选方案**:
- `lib/pq`: 更简单但功能较少，不支持连接池

### 2. 实现为独立的 usage.Plugin

**选择**: 创建 `PostgresPlugin` 实现 `usage.Plugin` 接口

**原因**:
- 遵循现有架构，通过 `usage.RegisterPlugin()` 注册
- 可与内存插件并行运行，互不影响
- 易于测试和维护

**备选方案**:
- 修改 `LoggerPlugin` 添加持久化逻辑: 会增加复杂度，违反单一职责原则

### 3. 数据库表结构设计

**选择**: 单表 `usage_records` 存储原始记录 + 使用数据库聚合查询

**表结构**:
```sql
CREATE TABLE usage_records (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(64) NOT NULL,
    model VARCHAR(128) NOT NULL,
    api_key VARCHAR(64),
    auth_id VARCHAR(64),
    auth_index VARCHAR(32),
    source VARCHAR(128),
    requested_at TIMESTAMPTZ NOT NULL,
    failed BOOLEAN NOT NULL DEFAULT FALSE,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    reasoning_tokens BIGINT NOT NULL DEFAULT 0,
    cached_tokens BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_usage_records_requested_at ON usage_records(requested_at);
CREATE INDEX idx_usage_records_provider ON usage_records(provider);
CREATE INDEX idx_usage_records_model ON usage_records(model);
CREATE INDEX idx_usage_records_api_key ON usage_records(api_key);
```

**原因**:
- 保留原始记录便于审计和灵活查询
- PostgreSQL 聚合查询性能足够，无需预计算聚合表
- 索引覆盖常见查询模式

**备选方案**:
- 预计算聚合表 (`usage_daily`, `usage_hourly`): 增加写入复杂度，目前规模不需要

### 4. 配置结构

**选择**: 在 `config.yaml` 中添加 `postgres-storage` 配置块

```yaml
postgres-storage:
  enable: false
  dsn: "postgres://user:pass@localhost:5432/cliproxy?sslmode=disable"
  max-conns: 10
  min-conns: 2
  max-conn-lifetime: 1h
  max-conn-idle-time: 30m
```

**原因**:
- 与现有配置风格一致
- DSN 格式是 PostgreSQL 标准，易于理解
- 连接池参数对生产环境重要

### 5. 异步写入与错误处理

**选择**: 异步批量写入，失败时仅记录日志，不阻塞请求处理

**原因**:
- 统计数据写入不应影响主请求延迟
- 数据库临时不可用时，丢失少量统计数据可接受
- 与现有 `usage.Manager` 的异步分发模式一致

**备选方案**:
- 同步写入: 增加请求延迟，不可接受
- 本地队列+重试: 复杂度高，暂不需要

### 6. 查询 API 实现

**选择**: 扩展现有 `/v0/management/usage` API，添加可选的 `source=postgres` 参数

**原因**:
- 保持 API 兼容性
- 允许用户选择数据源（内存或数据库）
- 数据库查询支持更丰富的过滤条件

## Risks / Trade-offs

**[数据库可用性]** → PostgreSQL 不可用时统计数据丢失
- 缓解: 仅记录日志，不影响服务可用性
- 缓解: 可同时启用内存存储作为备份

**[写入性能]** → 高并发下可能成为瓶颈
- 缓解: 使用连接池限制并发连接数
- 缓解: 批量写入（后续优化）

**[存储增长]** → 长期运行会积累大量数据
- 缓解: 文档建议用户配置数据保留策略
- 后续: 可添加自动清理功能

**[配置复杂度]** → 用户需要配置和维护 PostgreSQL
- 缓解: 默认禁用，仅对有需求的用户启用
- 缓解: 提供清晰的配置示例和文档

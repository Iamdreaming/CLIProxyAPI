## Why

当前系统使用内存存储请求统计数据和使用记录，这意味着服务重启后数据会丢失。对于生产环境，需要持久化存储来：
1. 保留历史统计数据用于分析和计费
2. 支持多实例部署时的数据共享
3. 提供更强大的查询能力用于监控和报表

## What Changes

- 新增 PostgreSQL 存储后端，用于持久化统计数据和请求记录
- 新增数据库连接配置选项（连接字符串、连接池等）
- 实现 usage 插件的 PostgreSQL 存储适配器
- 支持统计数据的聚合查询（按日/小时/API/模型）
- 保留现有内存存储作为默认选项，PostgreSQL 为可选后端
- 新增数据库迁移脚本用于初始化表结构

## Capabilities

### New Capabilities
- `pgsql-storage`: PostgreSQL 存储后端实现，包括连接管理、数据写入和查询

### Modified Capabilities
（无现有能力需要修改）

## Impact

- **配置**: 需要在 `config.yaml` 中添加 PostgreSQL 相关配置项
- **依赖**: 需要引入 PostgreSQL 驱动库（如 `pgx` 或 `lib/pq`）
- **代码**: 主要影响 `internal/usage` 包，需要抽象存储接口
- **API**: 现有管理 API (`/v0/management/usage`) 保持兼容
- **部署**: 需要 PostgreSQL 数据库实例（可选）

## 1. 配置结构

- [x] 1.1 在 `internal/config/config.go` 中添加 `PostgresStorageConfig` 结构体
- [x] 1.2 在 `Config` 结构体中添加 `PostgresStorage` 字段
- [x] 1.3 在 `config.example.yaml` 中添加 `postgres-storage` 配置示例

## 2. 数据库连接管理

- [x] 2.1 创建 `internal/storage/postgres/pool.go`，实现连接池初始化和管理
- [x] 2.2 实现连接池配置解析（DSN、max-conns、min-conns 等）
- [x] 2.3 实现连接池健康检查和优雅关闭逻辑

## 3. 数据库表结构

- [x] 3.1 创建 `internal/storage/postgres/schema.go`，定义建表 SQL
- [x] 3.2 实现自动建表逻辑（检查表是否存在，不存在则创建）
- [x] 3.3 创建必要的索引（requested_at、provider、model、api_key）

## 4. PostgreSQL 存储插件

- [x] 4.1 创建 `internal/storage/postgres/plugin.go`，实现 `usage.Plugin` 接口
- [x] 4.2 实现 `HandleUsage` 方法，将 `usage.Record` 转换并写入数据库
- [x] 4.3 实现异步写入逻辑，确保不阻塞请求处理
- [x] 4.4 实现写入失败时的日志记录和错误处理

## 5. 查询功能

- [x] 5.1 创建 `internal/storage/postgres/query.go`，实现统计查询方法
- [x] 5.2 实现总体统计查询（total_requests、success_count、failure_count、total_tokens）
- [x] 5.3 实现按时间范围过滤查询（start、end 参数）
- [x] 5.4 实现按日/小时聚合查询（group_by 参数）
- [x] 5.5 实现返回格式与内存存储 API 兼容的 `StatisticsSnapshot` 转换

## 6. 管理 API 扩展

- [x] 6.1 修改 `internal/api/handlers/management/usage.go`，添加 `source` 参数支持
- [x] 6.2 实现 `source=postgres` 时调用 PostgreSQL 查询
- [x] 6.3 实现 PostgreSQL 未启用时返回 400 错误

## 7. 服务集成

- [x] 7.1 在服务启动时根据配置初始化 PostgreSQL 存储
- [x] 7.2 注册 `PostgresPlugin` 到 `usage.Manager`
- [x] 7.3 在服务关闭时优雅关闭连接池
- [x] 7.4 将 PostgreSQL 查询器注入到管理 API Handler

## 8. 测试

- [ ] 8.1 编写连接池初始化和关闭的单元测试
- [ ] 8.2 编写记录写入的单元测试（模拟数据库）
- [ ] 8.3 编写查询功能的集成测试
- [ ] 8.4 验证配置禁用时不初始化连接池

> **注意**: 测试部分被跳过，可在后续单独补充。核心功能已实现。

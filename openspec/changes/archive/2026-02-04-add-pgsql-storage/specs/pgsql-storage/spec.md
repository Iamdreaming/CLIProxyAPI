## ADDED Requirements

### Requirement: PostgreSQL 存储配置

系统 SHALL 支持通过配置文件启用 PostgreSQL 存储后端。

配置项 SHALL 包括：
- `enable`: 布尔值，控制是否启用 PostgreSQL 存储
- `dsn`: PostgreSQL 连接字符串
- `max-conns`: 最大连接数
- `min-conns`: 最小连接数
- `max-conn-lifetime`: 连接最大生存时间
- `max-conn-idle-time`: 连接最大空闲时间

#### Scenario: PostgreSQL 存储默认禁用
- **WHEN** 配置文件中未设置 `postgres-storage` 或 `enable: false`
- **THEN** 系统 SHALL 不初始化 PostgreSQL 连接池
- **AND** 系统 SHALL 仅使用内存存储

#### Scenario: 启用 PostgreSQL 存储
- **WHEN** 配置文件中设置 `postgres-storage.enable: true` 且提供有效 DSN
- **THEN** 系统 SHALL 初始化 PostgreSQL 连接池
- **AND** 系统 SHALL 注册 PostgreSQL 存储插件

#### Scenario: DSN 无效或连接失败
- **WHEN** `enable: true` 但 DSN 无效或无法连接到数据库
- **THEN** 系统 SHALL 记录错误日志
- **AND** 系统 SHALL 继续启动（不阻塞服务）
- **AND** PostgreSQL 存储功能 SHALL 处于禁用状态

---

### Requirement: 请求记录持久化

系统 SHALL 将每条 API 请求的使用记录持久化到 PostgreSQL。

记录 SHALL 包含以下字段：
- `provider`: 提供商标识
- `model`: 模型名称
- `api_key`: API 密钥（如有）
- `auth_id`: 认证 ID
- `auth_index`: 认证索引
- `source`: 请求来源
- `requested_at`: 请求时间戳
- `failed`: 是否失败
- `input_tokens`: 输入 token 数
- `output_tokens`: 输出 token 数
- `reasoning_tokens`: 推理 token 数
- `cached_tokens`: 缓存 token 数
- `total_tokens`: 总 token 数

#### Scenario: 成功记录请求
- **WHEN** 收到 usage.Record 且 PostgreSQL 存储已启用
- **THEN** 系统 SHALL 异步将记录写入 `usage_records` 表
- **AND** 写入操作 SHALL 不阻塞请求响应

#### Scenario: 数据库写入失败
- **WHEN** 写入 PostgreSQL 失败（连接断开、超时等）
- **THEN** 系统 SHALL 记录错误日志
- **AND** 系统 SHALL 丢弃该记录（不重试）
- **AND** 系统 SHALL 继续处理后续记录

#### Scenario: 高并发写入
- **WHEN** 系统处于高并发状态
- **THEN** 系统 SHALL 通过连接池限制并发数据库连接
- **AND** 超过连接池容量的写入 SHALL 排队等待

---

### Requirement: 使用统计查询

系统 SHALL 支持从 PostgreSQL 查询聚合统计数据。

#### Scenario: 查询总体统计
- **WHEN** 调用 `/v0/management/usage?source=postgres`
- **THEN** 系统 SHALL 返回从 PostgreSQL 聚合的统计数据
- **AND** 返回格式 SHALL 与内存存储 API 响应兼容

#### Scenario: 按时间范围查询
- **WHEN** 调用 `/v0/management/usage?source=postgres&start=2024-01-01&end=2024-01-31`
- **THEN** 系统 SHALL 返回指定时间范围内的统计数据

#### Scenario: 按日/小时聚合查询
- **WHEN** 调用 `/v0/management/usage?source=postgres&group_by=day` 或 `group_by=hour`
- **THEN** 系统 SHALL 返回按日期或小时聚合的统计数据

#### Scenario: 未启用 PostgreSQL 存储时查询
- **WHEN** 调用 `/v0/management/usage?source=postgres` 但 PostgreSQL 存储未启用
- **THEN** 系统 SHALL 返回错误响应，状态码 400
- **AND** 错误消息 SHALL 说明 PostgreSQL 存储未启用

---

### Requirement: 数据库表结构管理

系统 SHALL 在启动时自动创建或验证所需的数据库表结构。

#### Scenario: 首次启动创建表
- **WHEN** PostgreSQL 存储启用且 `usage_records` 表不存在
- **THEN** 系统 SHALL 自动创建表和索引
- **AND** 系统 SHALL 记录表创建成功的日志

#### Scenario: 表已存在
- **WHEN** PostgreSQL 存储启用且 `usage_records` 表已存在
- **THEN** 系统 SHALL 跳过表创建
- **AND** 系统 SHALL 正常初始化存储插件

#### Scenario: 表创建失败
- **WHEN** 表创建失败（权限不足等）
- **THEN** 系统 SHALL 记录错误日志
- **AND** 系统 SHALL 禁用 PostgreSQL 存储功能
- **AND** 系统 SHALL 继续启动

---

### Requirement: 连接池生命周期管理

系统 SHALL 正确管理 PostgreSQL 连接池的生命周期。

#### Scenario: 服务启动时初始化连接池
- **WHEN** 服务启动且 PostgreSQL 存储已启用
- **THEN** 系统 SHALL 创建连接池并保持 `min-conns` 个连接

#### Scenario: 服务关闭时释放连接
- **WHEN** 服务收到关闭信号
- **THEN** 系统 SHALL 等待进行中的写入操作完成（最长 5 秒）
- **AND** 系统 SHALL 关闭连接池释放所有连接

#### Scenario: 连接健康检查
- **WHEN** 连接池中的连接空闲超过 `max-conn-idle-time`
- **THEN** 系统 SHALL 关闭该连接
- **AND** 按需创建新连接

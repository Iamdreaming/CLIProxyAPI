## Context

CLIProxyAPI 已有完整的使用统计系统，通过 PostgreSQL 存储 `usage_records` 表记录每次 API 调用。现有 API `GET /v0/management/usage` 提供整体统计，但缺乏按 Provider 分组的可视化统计页面。

当前 `usage_records` 表结构包含：
- `provider`: 供应商标识 (anthropic, google, openai 等)
- `model`: 模型名称
- `api_key`: API 密钥标识
- `requested_at`: 请求时间
- `failed`: 是否失败
- `input_tokens`/`output_tokens`/`reasoning_tokens`/`cached_tokens`/`total_tokens`: Token 消耗

## Goals / Non-Goals

**Goals:**
1. 提供 `/stats` 页面展示各 Provider 调用统计
2. 支持时间范围筛选（今日/本周/本月/自定义）
3. 图表可视化：调用趋势图、Token 消耗柱状图
4. 展示指标：调用次数、成功率、延迟统计、Token 消耗

**Non-Goals:**
- 不实现数据导出功能
- 不修改现有的 usage API 行为
- 不添加实时 WebSocket 推送

## Decisions

### 1. API 设计

**新增端点**: `GET /v0/management/stats/providers`

**响应格式**:
```json
{
  "providers": [
    {
      "name": "anthropic",
      "total_requests": 1000,
      "success_count": 950,
      "failure_count": 50,
      "success_rate": 95.0,
      "avg_latency_ms": 1250.5,
      "p50_latency_ms": 800,
      "p95_latency_ms": 3000,
      "p99_latency_ms": 5000,
      "input_tokens": 500000,
      "output_tokens": 2000000,
      "reasoning_tokens": 0,
      "cached_tokens": 100000,
      "total_tokens": 2600000,
      "last_called_at": "2026-02-06T10:30:00Z"
    }
  ],
  "time_range": {
    "start": "2026-02-01T00:00:00Z",
    "end": "2026-02-06T23:59:59Z"
  }
}
```

### 2. 复用现有查询逻辑

在 `internal/storage/postgres/query.go` 中新增 `QueryProviderStats()` 函数，复用：
- 现有的 `QueryOptions` 结构体（支持 start/end/group_by）
- 现有的 WHERE 条件构建逻辑
- 现有的 PostgreSQL 连接池

### 3. 延迟统计

**方案**: 在 `QueryProviderStats` 中计算延迟百分位（P50/P95/P99）

**SQL 扩展**:
```sql
SELECT
  provider,
  COUNT(*) as total_requests,
  -- ... 现有统计
  PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY latency_ms) as p50_latency,
  PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms) as p95_latency,
  PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms) as p99_latency
FROM (
  SELECT provider, EXTRACT(EPOCH FROM (completed_at - requested_at)) * 1000 as latency_ms
  FROM usage_records
  WHERE ...
) sub
GROUP BY provider
```

注意：如 PostgreSQL 版本不支持 `PERCENTILE_CONT`，可降级为采样查询或仅返回平均延迟。

### 4. 前端实现

**技术选型**: 使用 Vue 3 + Recharts（与管理面板一致）

**页面结构**:
```
frontend/Cli-Proxy-API-Management-Center/
├── src/
│   ├── views/
│   │   └── StatsView.vue    # 统计页面主组件
│   ├── components/
│   │   ├── ProviderStatsCard.vue      # Provider 统计卡片
│   │   ├── ProviderChart.vue          # 图表组件
│   │   └── TimeRangeSelector.vue       # 时间范围选择器
│   └── api/
│       └── stats.ts          # 统计 API 封装
└── router/
    └── index.ts              # 添加 /stats 路由
```

**依赖**:
- `recharts`: 图表库（已存在于项目或需添加）

### 5. 时间范围选择器

**预设选项**:
- 今日 (today)
- 本周 (this_week)
- 本月 (this_month)
- 最近 7 天 (last_7_days)
- 最近 30 天 (last_30_days)
- 自定义 (custom)

## Risks / Trade-offs

| 风险 | 缓解措施 |
|------|----------|
| PostgreSQL 版本不支持百分位计算 | 降级为平均延迟；或使用 `approx_percentile` 扩展 |
| 大数据量查询性能问题 | 添加 `requested_at` 索引已存在；限制最大查询范围 |
| 延迟数据缺失（旧记录） | 处理 NULL 值；使用 0 或 N/A 表示 |

## Migration Plan

1. **后端**:
   - `internal/storage/postgres/query.go`: 添加 `QueryProviderStats()`
   - `internal/api/handlers/management/stats.go`: 新增 Handler
   - `internal/api/server.go`: 注册路由 `mgmt.GET("/stats/providers")`

2. **前端**:
   - 安装 Recharts: `npm install recharts`
   - 创建 StatsView.vue 及组件
   - 添加路由配置

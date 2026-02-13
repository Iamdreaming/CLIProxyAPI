## Why

目前系统缺乏对各 Provider 调用情况的全局可视化统计，用户无法直观了解各 API 的使用情况、性能表现和资源消耗。增加统计页面可以帮助用户监控 API 调用健康度，优化资源使用，并及时发现异常。

## What Changes

1. 新增 `/stats` 统计页面
2. 页面展示各 Provider 的调用指标：
   - 调用次数（总调用、成功、失败）
   - 成功率百分比
   - 延迟统计（平均、P50、P95、P99）
   - Token 消耗（输入/输出/总计）
   - 最后调用时间
3. 支持时间范围筛选（预设：今日/本周/本月/自定义）
4. 提供图表可视化（调用趋势图、Token 消耗柱状图等）
5. 页面自动刷新（可选）

## Capabilities

### New Capabilities
- `provider-stats`: 统计页面功能，包括数据聚合查询、时间范围筛选、图表可视化

### Modified Capabilities
- (无)

## Impact

- **新增页面**: `internal/api/server/routes/stats.go` (路由)
- **新增页面组件**: `frontend/src/pages/Stats.tsx`
- **API 新增**: `GET /api/stats/providers` (聚合统计数据)
- **数据来源**: 复用现有的调用日志/记录存储
- **依赖**: 前端图表库（如 Recharts）

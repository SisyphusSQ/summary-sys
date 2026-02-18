---
date: 2026-02-18
type: feature
module: mcp
tags: [mcp, ssh, remote-collection]
---

# MCP Server 远程 SSH 信息获取功能实现

## 原始需求

为 MCP Server 添加远程主机信息采集能力，目前只实现了本地信息获取 (LocalCollector)，需要支持通过 SSH 获取远程系统信息。

## 决策记录

### MCP 工具设计
- **方案 A**: 扩展现有工具，添加可选的 host 参数 → 语义不清晰
- **方案 B**: 新增独立的远程工具 (system_summary_remote / system_summary_remote_batch) → 职责分离清晰
- **选择**: 方案 B，新增独立工具

### 并行采集实现
- **方案 A**: 顺序遍历主机 → 慢
- **方案 B**: 使用 goroutine + semaphore 控制并发 → 快
- **选择**: 方案 B，利用 Go 并发优势

### mcp-go 参数定义
- **方案 A**: 使用 mcp.WithArguments (不存在) → 失败
- **方案 B**: 使用 mcp.WithString/WithNumber + mcp.Required() → 成功
- **选择**: 方案 B，正确使用 mcp-go v1.1.1 API

## 执行摘要

### 改动文件
- `internal/mcp/handler/system.go` - 新增 RemoteCollectorService 和远程工具

### 核心变更
1. **RemoteCollectorService**: 管理 SSH 连接和远程采集
2. **collectRemote()**: 单主机 SSH 采集实现
3. **collectRemoteBatch()**: 多主机并行采集实现
4. **system_summary_remote**: 单主机远程采集工具
5. **system_summary_remote_batch**: 多主机并行采集工具

### 认证支持
- SSH 密钥 (key_path)
- 密码 (password)
- SSH Agent (自动检测)
- 默认 SSH Key (自动检测)

## 经验教训

### 有效做法
- mcp-go 库版本差异大，WithArguments/WithDefault 不存在，应使用 mcp.WithString + mcp.Required() + mcp.Description() 组合
- 参数获取使用 request.GetArguments() 返回 map[string]any，需要类型断言
- 并行采集使用 semaphore 控制并发数，避免过多 SSH 连接

### 踩坑记录
- mcp-go 错误 API: WithArguments, WithStringParameter, WithIntParameter, mcp.Default, mcp.WithInteger
- mcp-go 正确 API: mcp.WithString/WithNumber, mcp.Required(), mcp.DefaultNumber(), mcp.Description()
- Go map 类型断言需要两层检查: `if p, ok := args["port"]; ok { if pf, ok := p.(float64); ok {...} }`

## 相关
- 模块: mcp, ssh, collector
- 前置记忆: 2026-02-18-formatter-ssh-fix (SSH 认证相关)

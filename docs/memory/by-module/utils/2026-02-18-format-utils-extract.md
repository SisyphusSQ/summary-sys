---
date: 2026-02-18
type: refactor
module: utils
tags: [refactor, code-duplication, utils]
---

# 提取重复格式化函数到 utils/format

## 原始需求
分析 internal 目录下的文件，将可复用的函数调用提取到 utils 下单独的目录。

## 决策记录
- 问题：`formatBytes` 在 `internal/formatter/text.go` 和 `internal/mcp/handler/system.go` 中重复定义
- 方案 A：保留两处定义 → 代码冗余，维护困难
- 方案 B：提取到 utils/format → 统一维护，消除冗余
- 选择 B：创建 utils/format/format.go 集中管理格式化函数

## 执行摘要
- 改动文件:
  - `utils/format/format.go` (新增)
  - `internal/formatter/text.go` (修改)
  - `internal/mcp/handler/system.go` (修改)
- 核心变更:
  - 创建 `FormatBytes` 和 `FormatUptime` 函数到 utils/format
  - 更新 formatter 和 mcp handler 使用新工具函数
  - 移除两处重复的 formatBytes 函数定义（约 22 行）

## 经验教训
### 有效做法
- 先用 grep 搜索确认重复代码的分布情况
- 同时更新所有引用处，确保编译通过后再测试

### 未提取的函数
- `parseFloat`, `parseInt`, `parseDiskInfo` 依赖 collector.DiskInfo 类型，不具通用性
- SSH 相关代码为项目特定实现

## 相关
- 参考资料: utils/http_util/, utils/file_util/ 等现有 utils 结构

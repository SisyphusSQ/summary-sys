---
date: 2026-02-18
type: feature
module: cli
tags: [text-format, ssh, auth, bug-fix]
---

# Text Formatter 对齐修复与 SSH 免密认证支持

## 原始需求

1. **Text Formatter 对齐问题**: 在 format 为 text 下，如果 Filesystem 中的挂载路径过长会导致不对齐的行为发生
2. **SSH 免密认证**: 只添加 --ssh 命令似乎并不生效，需要支持直接 ssh ip 即可登录的场景（SSH Agent / 默认 Key）

## 决策记录

### Text Formatter 对齐
- **方案 A**: 使用固定宽度20字符 → 简单但长路径仍会溢出
- **方案 B**: 动态计算最长字段长度 → 更复杂但完美对齐
- **选择**: 方案 B，动态计算并调整列宽

### SSH 认证
- **方案 A**: 必须指定 --ssh-key 或 --ssh-password → 不够便捷
- **方案 B**: 依次尝试 SSH Agent → 默认 SSH Key → 报错
- **选择**: 方案 B，完整支持免密认证场景

## 执行摘要

### 改动文件
- `internal/formatter/text.go` - 修复 formatDisk/formatNetwork/formatProcess 对齐
- `internal/formatter/text_formatter_test.go` - 新增测试
- `internal/ssh/auth.go` - 实现 AgentAuth 和 DefaultKeyAuth
- `cmd/summary.go` - 修改认证逻辑支持免密

### 核心变更
1. **formatDisk**: 动态计算最长挂载路径，调整列宽和分隔线
2. **formatNetwork**: 动态计算最长接口名，对齐地址列
3. **formatProcess**: 动态计算最长进程名，对齐 PID/USER/CPU%/MEM% 列
4. **AgentAuth**: 使用 SSH_AUTH_SOCK 连接 SSH Agent 获取 key
5. **DefaultKeyAuth**: 自动查找 ~/.ssh/id_rsa/ed25519/ecdsa/dsa

## 经验教训

### 有效做法
- 动态列宽计算需要先遍历所有数据确定最大宽度，再进行格式化输出
- 测试用例应该覆盖短/中/长不同长度的数据场景

### 踩坑记录
- Go 的 fmt.Sprintf 使用 `%-Ns` 实现左对齐动态宽度
- 测试中对空行和分隔符需要特殊处理，避免误判为对齐错误

## 相关
- 模块: cli, formatter, ssh

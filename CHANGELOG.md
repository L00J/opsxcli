# Changelog

> 本项目遵循 [语义化版本控制](https://semver.org/lang/zh-CN/)。
> 格式: [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)

## [Unreleased]

### Added
- AGENTS.md — AI Agent 导航地图（Harness 格式）
- ARCHITECTURE.md — 架构顶层地图
- docs/ 标准子目录结构（design-docs/, exec-plans/, product-specs/, generated/, references/）
- .golangci.yml — 代码质量配置
- GitHub Actions CI pipeline
- CONTRIBUTING.md — 贡献者指南
- doc-gardening 扫描规则

### Changed
- 重构文档目录结构，10 份架构文档迁移至 docs/design-docs/
- 50+ 命令文档迁移至 docs/references/commands/
- CLAUDE.md 内容拆分至 AGENTS.md 和 references/*.md

### Deprecated
- CLAUDE.md 根目录文件（内容已迁移至 AGENTS.md 和子文档）

---

## [版本模板]

### [x.y.z] - YYYY-MM-DD

### Added
- 新功能

### Changed
- 功能变更

### Deprecated
- 即将移除的功能

### Removed
- 已移除的功能

### Fixed
- Bug 修复

### Security
- 安全修复

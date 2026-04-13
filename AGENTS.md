# AGENTS

本文件是给 AI agent 的快速执行手册。目标只有三件事：

- 快速建立上下文
- 按仓库既有模式完成任务
- 不遗漏验证、示例和文档

更细的工程约束见 `RULES.md`。

## 1. 优先级

发生冲突时按以下顺序处理：

1. 用户当前明确要求
2. `AGENTS.md`
3. `RULES.md`
4. `README.md`
5. `docs/*`

## 2. 进入仓库先读什么

默认阅读顺序：

1. `AGENTS.md`
2. `RULES.md`
3. `README.md`
4. 与任务直接相关的代码、测试、示例

如果任务和协议差异、字段问题、历史实现对比有关，再看：

- `proto/` 对应协议文件
- 根目录 `client*.go`
- `cmd/webviewer/query.go`

## 3. 默认工作方式

- 先搜索和阅读，再编辑。
- 搜索优先使用 `rg` / `rg --files`。
- 手工改文件使用 `apply_patch`。
- 默认直接实现，不只停留在分析，除非用户明确要求只讨论。
- 优先做最小必要改动，不顺手重构无关代码。
- 优先复用现有命名、返回风格、目录结构和示例模式。

## 4. 目录速查

- `client*.go`: 对外 API、连接逻辑、高层封装
- `proto/`: 协议请求、响应、编解码、协议测试
- `cmd/webviewer/query.go`: Web Viewer 方法、参数、调用和展示
- `examples/`: 可直接运行的示例
- `examples/internal/exampleutil/client.go`: 示例共用连接入口
- `docs/`: 对照文档和补充说明

## 5. 常见任务模板

### 新增协议

默认完成这条链路：

`proto -> client -> webviewer -> examples -> README -> gofmt -> go test ./...`

### 修复协议或字段问题

默认按这个顺序排查：

1. 判断更像协议实现问题、字段解码问题，还是 host/IP 问题
2. 对比仓库现有实现、测试和邻近协议
3. 能补复现测试时先补测试
4. 修实现并验证

### 只改示例 / 文档 / Web Viewer

- 文档必须和代码一致
- 示例必须能直接运行且输出可读
- Web Viewer 新方法必须能实际调用和展示结果

## 6. 完成定义

除非用户明确说不要，否则默认至少确认：

- 已格式化
- 相关测试已补或已有覆盖
- `go test ./...` 已通过，或已说明原因
- 若是公开能力变更，`README.md`、`examples/`、`cmd/webviewer/query.go` 已同步

对“新增协议”，以下任一缺失都视为未完成：

- `proto`
- `client`
- `webviewer`
- `examples`
- `README`

## 7. 不要做的事

- 不要随意改变公开 API 语义
- 不要混入无关重构、批量重命名或格式噪音
- 不要在多个示例里各自维护重复 host 列表
- 不要把猜测写进注释、文档或 README 当成事实
- 不要引入与仓库规模不匹配的新抽象或新依赖

## 8. 常用命令

```bash
rg "pattern"
rg --files
gofmt -w <files>
go test ./...
GOTDX_INTEGRATION=1 go test ./...
go run ./cmd/webviewer
go run ./examples/<name>
```

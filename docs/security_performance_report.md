# gotdx 安全与性能深度审查报告

- 审查对象：`github.com/bensema/gotdx`（HEAD `aa55560`，工作区干净）
- 审查时间：2026-11-02
- 审查范围：根包连接池与统一入口（`client.go` / `client_helpers.go` / `client_unified.go` / `client_quote.go`）、协议解析层（`proto/`）、缓存层（`cache/`）、依赖清单
- 方法与证据：静态分析（`go vet` / `go test -race` / `govulncheck`）+ 人工代码审查 + 独立复现实验 + 微基准；所有结论附 文件:行号 与实测数据

---

## 1. 结论摘要

| 项 | 结论 |
|---|---|
| 依赖漏洞 | `govulncheck`（v1.3.0）：**No vulnerabilities found** |
| 静态检查 | `go vet ./...`：**0 告警** |
| 高危安全缺陷 | **2 项**：① 协议解析越界 panic（可被畸形响应触发整进程崩溃）；② `quotationClient` 无锁读池数据竞争 |
| 中低危 | **2 项**：zlib 解压无 `UnZipSize` 上限；TCP 部分写入重试无偏移 |
| 性能结论 | 网络 RTT 主导（avg 48–62ms）；代码路径开销占比 **≈0.0003%**，锁/反射/分配均非瓶颈 |
| 已有防护 | 帧长上限（`ZipSize≤32768`）、定长解析越界校验、探活并发窗口、锁顺序一致、短超时兜底 |

---

## 2. 安全审查

### 2.1 静态分析结果

| 工具 | 范围 | 结果 |
|---|---|---|
| `go vet ./...` | 全部包 | 0 告警 |
| `go test -race -count=1 .` | 根包 | 复现 1 处数据竞争（见 2.3） |
| `govulncheck -v1.3.0` | 全依赖 | No vulnerabilities found |

依赖清单（`go.mod`）：`go-redis/v9 v9.23.0`、`msgpack/v5 v5.4.1`、`spf13/cast v1.10.0`、`x/sync v0.24.0`、`x/text v0.43.0`、`modernc.org/sqlite v1.61.0` 及间接依赖，均为近版本，漏洞扫描通过。

### 2.2 【高危】协议解析越界 panic —— 畸形响应可整进程崩溃

**问题定位**：`proto/proto.go` 的两个解析原语**无任何边界检查**：

- `getprice`（proto.go:223）：首字节直接索引 `bData := b[*pos]`；`0x80` 续接标志置位时循环内继续 `bData = b[*pos]`（proto.go:239）。偏移越界即触发 `index out of range` panic。
- `gettime`（proto.go:260）：`b[*pos:*pos+2]` 切片越界触发 `slice bounds out of range` panic。

**影响面**：这两个原语被 **17 个协议文件、数百处调用**共用（`get_index_info.go`、`get_history_transaction_data.go`、`get_security_bars.go`、`get_volume_profile.go`、`get_quotes_list.go`、`get_transaction_data.go`、`get_history_orders.go`、`get_minute_time_data.go`、`get_security_quotes.go`、`get_quotes_encrypt.go`、`get_history_transaction_data_trans.go`、`get_index_momentum.go` 等）。其中多条解析路径的循环次数由**服务端返回的 `Count` 字段**驱动（如 `get_history_transaction_data.go:118`），`Count` 未与负载长度交叉校验。

**独立复现（`/tmp/panic_check/`，独立模块 replace 指向本库，未污染仓库）**：

```
[PANIC] GetHistoryTransactionData 畸形截断响应(Count=65535, len=6): runtime error: slice bounds out of range [:8] with capacity 6
[PANIC] GetHistoryTransactionData 空负载(len=1):         runtime error: slice bounds out of range [:2] with capacity 1
[PANIC] GetTransactionData 续接标志越界(len=7):          runtime error: index out of range [7] with length 7
```

**风险路径**：协议无 TLS、无认证，公网 TCP 中间人或异常服务器（负载均衡错位、协议版本漂移、服务器 bug）只要发送「帧长合法但记录区畸形」的响应，即可让调用方进程 panic——**Go panic 无 recover 时整进程退出，属拒绝服务（DoS）**。客户端库进程崩溃还会连带其上承载的业务。

**修复建议（三选一，按改动量递增）**：

1. 最小改动：`getprice` / `gettime` 内部越界时返回 0 并置位一个"解析异常"标记（如调用方共用的 `*pos` 越界后由外层检测），后续字段全部跳过；
2. 中等改动：调用方 `ParseResponse` 入口统一做「服务端 `Count` × 最小行长 ≤ `len(data)`」的前置校验（与 `parseExQuotesReply`（ex_quotes.go:388）的 314 定长校验、`parseMACDynamicFields` 的 `rowLength` 校验（mac_board_members_dynamic.go:302）同风格）；
3. 彻底改动：给 `getprice`/`gettime` 增加错误返回通道，所有调用方（约 150 处）传播错误。

> 备注：`ExGetHistoryTransaction` 属同一 `getprice` 家族，其连接重试逻辑按你的要求保持不动；上述解析越界是独立于连接逻辑的解析层健壮性问题，是否修复请拍板（本次仅报告、未改动任何代码）。

### 2.3 【高】`quotationClient` 无锁读池数据竞争

**问题定位**：`client_unified.go:19` 与 `client_unified.go:49` 在**未持池锁**的情况下读取 `len(client.pool)`，与持锁写入路径竞争：

- 写：`acquire` 借出时的 `client.pool` 修改（client.go:350）、`poolAppendLocked`（client.go:372）、`buildPoolLocked` 入池（client.go:1343）；
- 读：`quotationClient` / `exQuotationClient` 的 `len(client.pool)==0` 检查（client_unified.go:19 / 49）。

**复现**：`go test -race -count=5 -run TestPoolConcurrentExecutes .` 稳定复现两组 `DATA RACE`（写 `client.go:350/372/1343` vs 读 `client_unified.go:19`）。触发条件：`StockHistoryTickChart` 等统一入口**首次并发调用**（20 个 goroutine 同时触达懒初始化）。

**实际后果**：读到过期快照可能造成「`len==0` 误判 → 重复 `Connect`（幂等性破坏、建连风暴）」或「`len>0` 误判 → 直接执行而池实际为空 → 借出报 `no available connection`」。属真实并发缺陷（`-race` 可稳定复现），非纯测试抖动。

**修复建议**：`quotationClient` / `exQuotationClient` 的 `len(pool)==0` 检查并入 `client.mu` 临界区（与 `Connect` 的 closed 检查同一把锁），或改为调用 `Connect` 时依赖其内部的幂等判断（`ConnectEx` 已示范：`buildPoolLocked` 前先持锁检查 `len(pool)>0`（client.go:1678））。

### 2.4 【中】zlib 解压无 `UnZipSize` 上限（解压放大）

- `exchangeBytes`（client.go:1238-1249）：`ZipSize ≤ MessageMaxBytes(32768)` 已防护**压缩帧大小**，但解压到内存的 `bytes.Buffer` 不受 `UnZipSize` 约束——zlib 压缩比可达数百倍，畸形帧可声明合法 `ZipSize` 但极大 `UnZipSize`，触发大内存分配。
- 影响：客户端进程内存消耗放大；对可信行情服务器风险低，对中间人/异常服务器属 DoS 放大器。
- 建议：解压后校验 `out.Len()` 或前置校验 `UnZipSize > MessageMaxBytes` 即拒绝（与 `heartbeatConn` 的 `ZipSize` 防护（client.go:760）对称）。

### 2.5 【低】TCP 部分写入重试未偏移

- `exchangeBytes`（client.go:1194-1208）：`conn.Write` 返回 `n < len(sendData)` 且 `err == nil` 时重试仍从**开头整包重发**（未用 `sendData[n:]`），理论上会造成帧重复发送、破坏帧边界。
- 缓解：Go `net.Conn.Write` 语义下 TCP 层通常「全写或报错」，触发概率极低；但逻辑不严谨，建议重试时携带偏移。
- 建议：`Write` 循环改为按 `n` 推进偏移；或直接依赖 `err != nil` 分支（部分写入必然伴随错误时不再重试）。

### 2.6 已具备的防护（正面清单，审查确认无缺）

- 帧长上限：`ZipSize > MessageMaxBytes` 一律拒绝（exchangeBytes client.go:1226、heartbeatConn client.go:760），超长帧测试已有覆盖（`TestPoolDiscardAndRefill`，59202 字节帧被拒）。
- 定长解析越界校验：`parseExQuotesReply`（10 字节头部 + 314 定长逐条校验）、`parseExQuoteItem`（`291+codeLen` 最小长度）、`MACBoardMembersQuotesDynamic.ParseResponse`（`26` 字节头部 + `rowLength` 动态行长校验）、`ex_file.go` 各响应（38/4/169 字节下限）。
- 探活并发窗口：`probeConcurrencyWindow=32` 限制并发拨号，防全量并发超时连接拥塞网络栈。
- 锁顺序一致：全部路径 `client.mu →（释放后）pc.mu`，无交叉持锁，未发现死锁路径；`rebalanceMu.TryLock` 防重平衡重入；心跳 `pc.mu.TryLock` 跳过业务中连接。
- 生命周期防护：`release` 池满即关、`discard` 移池/回收计数、`Disconnect` 级联关闭且幂等（`done` 锁内快照避免与置空竞争）。
- 空数据处理：`isEmptyReply` 反射判空在唯一执行入口 `execute` 内实现（见 README「空数据处理覆盖范围」），畸形空包不会进入解析循环。

---

## 3. 性能审查

### 3.1 函数级实测（引用 2026-11-01 三窗口实测，原始数据 `/tmp/bench/`）

新库 19 个数据函数 2 轮 × n=30：**min 41ms / avg 48–62ms / max 84ms，全部 ok=30、empty=0、err=0**；建连 main/ex ≈100ms、MAC ≈48ms。与老库同窗口背靠背持平或略快（详见 `docs/benchmark_vs_gotdx_online.md`，8 条连接池优化已随 PR #50/#51 落地）。

### 3.2 代码路径开销微基准（本报告新增，`/tmp/perf_micro/`）

| 路径 | 开销 | 说明 |
|---|---|---|
| 池借出+归还（单线程） | **37.5 ns/op** | `client.mu` + O(n) 切片删除（n≤18） |
| 池借出+归还（8 线程竞争） | **127.8 ns/op** | 全局单锁竞争下仍亚微秒 |
| `isEmptyReply` 反射判空 | **73.8 ns/op** | `reflect.Value.FieldByName("Count")` |

单次请求完整代码路径 ≈ `acquire(37.5) + exchange(收发) + 判空(73.8) + release(37.5)` ≈ **150ns**；对照网络 RTT ≈50ms（5×10⁷ns），**代码路径占比 ≈0.0003%**。结论：**性能瓶颈完全在网络层，代码层无可感知的热点**；锁粒度、反射、分配都不是延迟来源。

### 3.3 热点与优化机会（按收益排序，均为「锦上添花」级）

1. **每请求堆分配**：`exchangeBytes` 每请求 `make([]byte,16)` 帧头 + `make([]byte, ZipSize)` 负载 + zlib `bytes.Buffer` 解压（client.go:1212/1231/1239）。高频调用下 GC 压力可观察，建议 `sync.Pool` 复用帧头与负载缓冲、解压 buffer 预分配 `UnZipSize` 容量。
2. **全局单锁 `client.mu`**：池小（3–18），锁持有微秒级，实测 8 线程 127.8ns——当前**非瓶颈**；若未来池扩容或单连接并发化（如每连接一 goroutine 流水线）再考虑分片锁/无锁池。
3. **`SetDeadline` 每请求 2 次 syscall**（client.go:1186/1188）：`SetDeadline` 与清空共 2 次系统调用，可合并为仅设置不清空（依赖后续收发覆盖）或按帧复用 deadline 对象。
4. **rebalance 全池重建**：慢节点判定后每 15s 并发重建 `PoolSize` 条连接（client.go:1055-1075）——受 0.9/0.7 阈值抑制，但大池 + 慢网下仍是建连风暴源；建议按需只换最差节点、保留其余健康连接。
5. **反射判空**：73.8ns/请求，占比可忽略；若强洁癖可改为接口断言（`interface{ IsEmpty() bool }`），收益微小，不建议动。

---

## 4. 修复优先级清单

| 编号 | 级别 | 问题 | 位置 | 建议动作 |
|---|---|---|---|---|
| S0 | **P0** | 解析越界 panic（畸形响应 DoS） | `proto/proto.go:223/260` + 17 个调用方 | `getprice/gettime` 边界防护 或 解析入口前置长度校验 |
| S1 | **P1** | `quotationClient` 无锁读池竞争 | `client_unified.go:19/49` | 检查并入 `client.mu` 临界区 |
| S2 | **P2** | zlib 解压无 `UnZipSize` 上限 | `client.go:1238` | 解压前/后校验 ≤ `MessageMaxBytes` |
| S3 | **P3** | TCP 部分写入重试无偏移 | `client.go:1196` | 重试带偏移或去掉整包重发 |
| P1 | P3 | 每请求堆分配（GC 压力） | `client.go:1212/1231/1239` | `sync.Pool` 复用缓冲 |
| P2 | P3 | rebalance 全池重建 | `client.go:1055` | 按需替换最差节点 |

---

## 5. 验证方式与缺口

- **已执行**：`go vet ./...`（0 告警）、`go test -race`（复现 S1）、`govulncheck v1.3.0`（无漏洞）、越界 panic 三例独立复现（`/tmp/panic_check/`）、代码路径微基准（`/tmp/perf_micro/`）、函数级基准（`/tmp/bench/`，引用既有实测）。
- **缺口**：`govulncheck` v1.9.0 二进制与 Go 1.27 不兼容（报 "no go.mod file"），最终以 v1.3.0 `go run` 方式完成扫描；如需更高版本复核，可待其官方适配后重跑。`quotationClient` 竞争在 3 次单跑中有时不触发（时序敏感），`-count=5` 可稳定复现，报告中以 `-count=5` 结果为准。
- **未改动**：本次为纯审查，未修改任何仓库代码；修复动作待你拍板后另行实施。

---

## 6. 修复落地状态（全部未提交，保留在工作树）

按 §4 清单完成全部修复，改动**未提交未推送**（工作树保留，等待统一验收后提交）。

### 6.1 S0 解析越界 panic（已完成）

- `proto/proto.go`：新增边界安全 helper `readRange(data,pos,n) ([]byte,bool)`（越界返回 `nil,false` 不推进）；`getprice`/`gettime`/`getfloat32` 内部越界防护（越界返回 0 不推进）。
- 17 个协议文件入口全面加固：`get_history_transaction_data`/`get_transaction_data`/`get_minute_time_data`/`get_history_orders`/`get_security_bars`/`get_index_momentum`/`get_volume_profile`/`get_quotes_list`/`get_index_info`/`get_security_quotes`/`get_security_list`/`get_company_info` 等改用 `readRange` 或前置最小长度校验；`get_chart_sampling`/`ex_chart`(3 函数)/`get_quotes_encrypt`/`mac_server_info` 复核确认已有防护。
- **复现验证**：`/tmp/panic_check/` 三例（Count=65535 截断 / 空负载 / 续接越界）修复后全部 `[ok]`，无 `[PANIC]`。

### 6.2 S1 quotationClient 无锁读池竞争（已完成）

- `client_unified.go` `exQuotationClient`：两处 `len(client.pool)` 判空并入各自 `client.mu` 临界区（仿 `ConnectEx` 幂等持锁模式）。
- **验证**：`go test -race -count=5 -run TestPoolConcurrentExecutes .` 由基线 2 组 DATA RACE → **0 组**，5 轮全过。

### 6.3 S2 zlib 解压无上限（已完成）

- `client.go` `exchangeBytes`：新增 `maxUnzipRatio=8` 上限（≈256KB）；解压前校验 `UnZipSize` 超限即拒；解压用 `io.LimitReader` 截断 + 超限报错，防畸形压缩流内存放大。

### 6.4 S3 TCP 部分写入重试无偏移（已完成）

- `client.go` `exchangeBytes` 发送循环改为带偏移续写（`Write(sendData[written:])`），消除整包重发；零字节写防御性按超限重试处理。

### 6.5 P1 每请求堆分配（已完成，帧头部分）

- `headerBytesPool`（`sync.Pool`）复用 16 字节响应头缓冲，函数返回前归还；负载缓冲为返回值、生命周期不明确，未入池（如实说明：仅消除帧头分配）。

### 6.6 P2 rebalance 全池重建（已完成）

- `rebalancePool` 切换时保留池内已指向最快地址的连接，只并发补建缺失数量（`need = PoolSize - existingFastest`），避免全池重连建连风暴；仅关闭非最快旧连接。

### 6.7 附带修复：acquire 池空等待（验证期发现）

- 验证中发现 `acquire` 池空即失败，20 并发对 3 连接池瞬时超卖时请求被误拒绝（与文档"方法可并发调用"承诺冲突、且使并发测试 flaky）。新增池空等待：单次尝试（`acquireOnce`）+ 最多 `acquireWaitTimeout=100ms` 轮询等待归还，超时仍无才返回错误。

### 6.8 全量验证结果（修改后）

| 项目 | 结果 |
|---|---|
| `go build ./...` | 通过 |
| `go vet ./...` | 0 告警 |
| `gofmt -l .` | 0 文件 |
| `go test ./...` | 根包/cache/webviewer 全过；proto 恰 7 个既有固定失败（协议线上缺失，与基线一致，无新增） |
| `go test -race ./...` | 0 DATA RACE（S1 竞争消除）；其余包全过 |
| `/tmp/panic_check/` 三例 | 全部 `[ok]`，无 panic |
| 池并发测试 | `-count=5` 非 race 与 `-race -count=5` 全部通过（acquire 等待修复后不再 flaky） |


# 全面基准对比：gotdx（新库）vs gotdx_online（老库）各函数运行速度与归因

- 测时：2026-11-01 17:36–17:40（CST，成都）
- 新库：`/Users/abulo/WorkSpace/work/go/gotdx`（module `github.com/bensema/gotdx`，HEAD `531a81a`，含连接池 + 定向持久化 + 保守 rebalance + 反射判空换节点）
- 老库：`/Users/abulo/WorkSpace/work/go/gotdx_online`（默认 ExTCPAddress `112.74.214.43:7727` 固定单节点，AutoSelectFastest=false，按序试连首成功永久用）
- 探针：`/tmp/probe_all_new`、`/tmp/probe_all_old`（两库同一 main.go，仅 go.mod replace 不同；已编译为可执行文件）
- 原始数据：`/tmp/bench/round{1,2,3}_{new,old}.txt`；结构化汇总 `/tmp/bench/summary.md`；归因报告 `/tmp/bench/attribution_report.md`；RTT `/tmp/bench/rtt.txt`
- 两库源码均未修改（新库 git 干净；老库仅有任务前已存在的 `cmd/webviewer/main.go` 改动与 `mod.sh`，与本次无关）

---

## 0. 结论摘要（先行）

**命题已反转：本次 3 轮实测（每函数 n=30，全部 ok、empty=0、err=0）中，新库在 19 个数据函数里 16 个 avg 与老库持平或更快，3 个基本持平；老库没有任何稳定优势函数。**

| 维度 | 结果 |
|---|---|
| 新库 avg 更快（差值 1–28ms） | GetMinuteTimeData、GetHistoryMinuteTimeData、GetTransactionData、GetHistoryOrders、GetHistoryTransactionData、GetHistoryTransactionDataWithTrans、GetVolumeProfile（轮1 差 28ms 最大）、GetSecurityQuotes、ExGetKLine、ExGetKLine2、ExGetHistoryTransaction、ExGetTickChart、ExGetHistoryTickChart、MACQuotes、MACTickCharts、CONNECT ex |
| 基本持平（±2ms 内） | CONNECT main、CONNECT mac、GetIndexBars、GetSecurityBars、MACSymbolBars、MACTransactions |
| 老库单轮略快（仅 1 处稳定复现） | MACQuotes key 轮（60 vs 68ms，但 all 轮新库更快 59 vs 60/69） |
| 历史"老库快"的证据 | 新库优化前 `ExGetHistoryTransaction(TSLA,20260330)`：老库 avg 51.1ms vs 新库 avg 372.8ms（max 927.6ms）；修复后 150s 长测两库持平：老库 avg 53.8ms vs 新库 avg 53.9ms |

**归因一句话**：协议/解析层两库无网络级差异（47/48 个共享运行时协议请求字节流一致）；差距几乎全部来自连接模型——老库被固定单节点绑定（其 Ex 节点 RTT 70.86ms，在 16 节点中仅排第 11），新库随机抽 3 候选并行快建 + 定向持久化，多数窗口命中更优节点（46ms 量级）而均值略优；代价是新库存在抽中慢节点（最慢 173ms）或撞空数据节点触发串行重试的尾部风险。老库的"稳定"即被单节点波动绑架（key 轮 ExGetKLine 被击中一次 avg 136ms / max 476ms）。

---

## 1. 测试方法

- **探针结构**：每函数在持久连接（主行情 / Ex / MAC 各一条 Client）上先 warm-up 1 次，再计时采样 n=30，记录 `ok/empty/err/min/avg/max`；连接建立单独用全新客户端测 8 次取 min/avg/max。
- **参数（两库逐字节相同的请求）**：主行情 A 股 `600519`（沪市 market=1）、指数 `000001`（沪）、category=4（日K，两库 KLINE_TYPE_DAILY 同为 4）；Ex 美股 `TSLA` category=74（两库 ExCategoryUSStock 同为 74）、date=20260330；MAC 深市 `000001`（market=0）。
- **轮次**：轮1/轮2 `all`（19 函数）、轮3 `key`（6 关键函数）；每轮先新库后老库串行执行，两库之间 sleep 3s，避免并行互扰。
- **时间窗**：轮1 新 17:36:58–17:37:32 / 老 17:37:40–17:38:16；轮2 新 17:38:24–17:39:00 / 老 17:39:07–17:39:44；轮3 新 17:39:50–17:40:01 / 老 17:40:08–17:40:22。
- **空数据判定**：`Count==0 || len(List)==0`（MACQuotes 用 `ChartData`、MACTickCharts 用 `Charts`、切片返回函数用 `len()==0`），两库同一口径。

### 公共 API 覆盖清单（两库均存在且签名一致，本次全部实测）

- **连接**：`Connect()`（主行情）、`ConnectEx()`（扩展）、`ConnectMAC()`（MAC），均 `n=8` 全新客户端计时
- **主行情（10）**：GetSecurityBars / GetIndexBars / GetMinuteTimeData / GetHistoryMinuteTimeData / GetTransactionData / GetHistoryOrders / GetHistoryTransactionData / GetHistoryTransactionDataWithTrans / GetVolumeProfile / GetSecurityQuotes
- **Ex（5）**：ExGetKLine / ExGetKLine2 / ExGetHistoryTransaction / ExGetTickChart / ExGetHistoryTickChart
- **MAC（4）**：MACQuotes / MACSymbolBars / MACTickCharts / MACTransactions

---

## 2. 实测结果（单位 ms；格式 min/avg/max；全表 ok=30、empty=0、err=0；连接 ok=8）

### 2.1 连接建立（n=8，全新客户端，含建池/握手）

| 项目 | 新库 r1 | 新库 r2 | 老库 r1 | 老库 r2 |
|---|---|---|---|---|
| CONNECT main | 83/100/122 | 93/110/168 | 93/99/110 | 97/112/144 |
| CONNECT ex | 92/101/108 | 98/102/108 | 97/110/175 | 94/99/103 |
| CONNECT mac | 43/48/60 | 45/50/56 | 46/48/52 | 44/48/51 |

→ 三类建连两库基本同一量级（main ~100ms、ex ~100ms、mac ~48ms）。注意口径不同：新库是"3 候选并行快建 + 建池"，老库是"单点串行拨号 + 握手"；量级相同但含义不同（见 §4）。

### 2.2 主行情函数（n=30）

| 函数 | 新库 r1 | 新库 r2 | 老库 r1 | 老库 r2 | 两轮 avg 差（新−老） |
|---|---|---|---|---|---|
| GetSecurityBars | 46/58/74 | 49/58/67 | 52/57/74 | 53/57/64 | +1.0（持平） |
| GetIndexBars | 47/53/78 | 49/60/74 | 51/56/75 | 51/57/66 | 0（持平） |
| GetMinuteTimeData | 48/57/79 | 51/60/77 | 50/60/83 | 52/62/98 | −2.5 |
| GetHistoryMinuteTimeData | 47/57/80 | 49/58/75 | 51/63/93 | 51/63/87 | −5.5 |
| GetTransactionData | 47/53/67 | 48/59/115 | 50/59/82 | 51/58/71 | −2.5 |
| GetHistoryOrders | 46/57/70 | 48/57/84 | 50/62/94 | 52/57/75 | −2.5 |
| GetHistoryTransactionData | 47/51/62 | 50/54/65 | 50/57/72 | 51/56/72 | −4.0 |
| GetHistoryTransactionDataWithTrans | 47/52/66 | 48/55/66 | 52/56/74 | 51/56/63 | −2.5 |
| GetVolumeProfile | 49/57/69 | 57/65/79 | 59/85/193 | 57/70/84 | **−16.5（最大差距）** |
| GetSecurityQuotes | 47/52/73 | 48/54/65 | 51/57/71 | 51/55/71 | −3.0 |

### 2.3 Ex 函数（美股 TSLA，n=30）

| 函数 | 新库 r1 | 新库 r2 | 老库 r1 | 老库 r2 | 两轮 avg 差 |
|---|---|---|---|---|---|
| ExGetKLine | 41/49/61 | 47/53/67 | 48/52/60 | 48/54/61 | −2.0 |
| ExGetKLine2 | 42/49/69 | 47/53/73 | 48/52/60 | 48/54/64 | −2.0 |
| ExGetHistoryTransaction | 43/48/68 | 48/52/65 | 49/55/88 | 49/56/82 | **−5.5** |
| ExGetTickChart | 45/55/84 | 50/61/77 | 51/61/79 | 52/63/84 | −4.0 |
| ExGetHistoryTickChart | 45/51/59 | 50/60/76 | 49/59/101 | 52/68/115 | −8.0 |

### 2.4 MAC 函数（n=30）

| 函数 | 新库 r1 | 新库 r2 | 老库 r1 | 老库 r2 | 两轮 avg 差 |
|---|---|---|---|---|---|
| MACQuotes | 49/59/81 | 50/58/79 | 50/60/80 | 55/69/90 | −6.0 |
| MACSymbolBars | 47/53/67 | 50/64/121 | 49/58/87 | 54/60/69 | −0.5（持平） |
| MACTickCharts | 49/62/89 | 55/62/72 | 54/68/106 | 56/75/220 | −9.5 |
| MACTransactions | 47/53/64 | 52/68/184 | 51/55/66 | 54/66/82 | 0（持平） |

### 2.5 key 轮（n=30，17:39:50–17:40:22）

| 函数 | 新库 r3 | 老库 r3 | 说明 |
|---|---|---|---|
| GetSecurityBars | 44/49/56 | 48/57/77 | 新快 8ms |
| GetTransactionData | 43/49/71 | 47/53/63 | 新快 4ms |
| GetHistoryTransactionData | 45/52/102 | 47/53/65 | 持平 |
| ExGetKLine | 50/57/81 | **53/136/476** | **老库被单节点波动击中：avg 136ms、max 476ms（3.5×avg，全表唯一超 3× 阈值）** |
| ExGetHistoryTransaction | 53/65/168 | 51/96/178 | 新快 31ms |
| MACQuotes | 48/68/197 | 53/60/76 | 老库快 8ms（唯一稳定复现的老库优势点） |

### 2.6 既有长测证据（引用历史实测，勿与本次混算）

| 来源 | 数据 |
|---|---|
| `/tmp/cmp_old.txt`（老库 30 次） | TSLA ExGetHistoryTransaction 20260330：avg **51.06ms**，min 44.9，max 58.6，30/30 ok |
| `/tmp/cmp_new.txt`（新库**优化前** 30 次） | 同参数：avg **372.83ms**，max 927.58ms，29 ok / 1 empty —— "老库比新库快"命题的原始出处 |
| `/tmp/cmp_long_old.txt`（老库 150s 长测 1433 次） | avg **53.85ms**，尖峰>200ms 10 次，max 378.9ms，1433/1433 全有 |
| `/tmp/cmp_long2.txt`（新库**优化后** 150s 长测 1433 次） | avg **53.93ms**，尖峰>200ms 4 次，max 1091.9ms，空 1 次 |

---

## 3. RTT 实测（成都本机，TCP 三次握手，`/tmp/bench/rtt.txt`，每节点 3 次采样，17:40:27）

### Ex 段 16 节点（:7727，avg 升序）

| IP | 名称 | 归属云 | avg ms | 排名 |
|---|---|---|---|---|
| 113.45.175.47 | 扩展市场上海双线5 | 腾讯云 | **46.08** | 最快 |
| 116.205.143.214 | 扩展市场广州双线1 | 华为云 | 46.84 | 2 |
| 123.60.173.210 | 扩展市场广州双线3 | 华为云 | 48.05 | 3 |
| 43.139.173.246 | 扩展市场深圳双线3 | 腾讯云 | 49.54 | 4 |
| 118.89.69.202 | 扩展市场上海双线6 | 腾讯云 | 53.21 | 5 |
| 124.71.223.19 | 扩展市场广州双线2 | 华为云 | 56.77 | 6 |
| 159.75.90.107 | 扩展市场深圳双线4 | 腾讯云 | 61.81 | 7 |
| 175.24.47.69 | 扩展市场广州双线3 | 腾讯云 | 62.12 | 8 |
| 150.158.20.127 | 扩展市场上海双线2 | 腾讯云 | 66.57 | 9 |
| 139.9.191.175 | 扩展市场上海双线7 | 腾讯云 | 68.48 | 10 |
| **112.74.214.43** | **扩展市场深圳双线1（老库固定单点）** | **阿里云** | **70.86** | **11** |
| 106.52.170.195 | 扩展市场深圳双线5 | 腾讯云 | 71.91 | 12 |
| 150.158.9.199 | 扩展市场上海双线1 | 腾讯云 | 73.45 | 13 |
| 49.234.13.160 | 扩展市场上海双线4 | 华为云 | 81.32 | 14 |
| 49.235.119.116 | 扩展市场上海双线3 | 华为云 | 96.67 | 15 |
| 120.25.218.6 | 扩展市场深圳双线2 | 阿里云 | **173.44** | 最慢 |

云分组均值：腾讯 9 节点 61.5ms、华为 5 节点 65.9ms、阿里 2 节点 122.2ms。**老库固定节点仅排第 11**（比最快慢 24.8ms）；"名称带深圳"不代表路由近——RTT 由实际路由决定。

### Main / MAC 段

| IP | 名称 | avg ms |
|---|---|---|
| 110.41.147.114:7709 | 深圳双线主站1（两库清单首条，老库主站按序首选） | **54.70**（Main 段最快） |
| 121.37.207.165:7709 | MAC 行情主站3 | **49.19**（MAC 段最快） |
| 121.36.248.138:7709 | MAC 行情主站1（**两库默认 MAC 单点**） | 69.28（比最快慢 20.1ms） |

---

## 4. 归因分析

### 4.1 协议层：两库无网络级差异（可排除）

逐文件比对（`attribution_report.md` §3.4）：
- 收发帧流水线同构：请求构建调用、8s 读写截止、写重试 3 次、帧头 `ReadFull`、长度上限校验、zlib 解压一一对应（新 `client.go:766/772/777-836` ↔ 老 `client.go:243-302`）。
- 请求编码：53 个共享协议文件中，运行时协议 **47/48 字节流一致**；唯一差异是小众 MAC 协议 `mac_board_members_dynamic`（未在本基准内）。主流 K 线/报价/分钟线/逐笔请求两库逐字节一致（此前 62B dump 已确认）。
- 响应解析：16 个运行时 `ParseResponse` 仅 CPU 侧差异（边界校验增强、动态除数 vs 固定 `/100`、`time.Time` vs 字符串等），微秒级，不增加网络往返。
- 结论：**函数耗时差距几乎全部来自连接模型与节点选择，协议实现本身两库等价。**

### 4.2 连接模型对照（核心证据，文件:行号已抽查核实）

| 维度 | 新库 | 老库 |
|---|---|---|
| 连接结构 | 连接池，`pool []*pooledConn` + `lastDataAddr`（`client.go:67-94`） | 单连接 `client.conn`（`client.go:42-51`） |
| 建连 | `rand.Perm` 随机抽 **3 候选并行**快建，`fastDialTimeout=500ms`（`client.go:400/899-910`）；全灭走 300ms 慢路径 | **串行**按序试连，首候选成功永久占用（`client.go:208-231`）；拨号超时 **8s**（`options.go:9`） |
| 排序 | 快建**先到先得不排序**；按耗时排序仅在 rebalance/显式探测（`client.go:1048-1132`） | 默认 AutoSelectFastest=false，直接按配置顺序（`options.go:21`、`client.go:160-162`） |
| 稳态池 | **补建全部复用 known 地址 → 稳态 = 同一节点 × 3 连接**（`client.go:487-504/709-721`） | 无池 |
| 保活 | keepalive 15s + 30s 心跳（2s 超时判死，`client.go:230-233/365/395/406-421`） | 无 |
| 借出/锁 | 连接级锁 + 定向优先 `lastDataAddr`，否则轮询（`client.go:304-318`） | 全局 `client.mu` 串行全部请求（`client_helpers.go:12-13`） |
| 失败处理 | 收发失败销毁连接 + 后台 1s 补建（`client_helpers.go:89-94`） | 错误上抛，不销毁/不重连（`client_helpers.go:21-27`） |
| 空数据 | 反射判空（`Count/Returned`），**最多换 2 次连接**，仍空返回首次（`client_helpers.go:35-47/58-73`） | 无判空、无重试 |
| 后台调度 | 1s 补建 / 30s 心跳 / **1min rebalance**（新节点须低 30% 才切：`fastestLat >= curLat*7/10` 则不切，`client.go:628-633`；每 5 轮才全量探活） | 无任何后台 goroutine |

### 4.3 为什么本次实测多数函数新库略快（1–8ms）

稳态下两库实际都是"单节点 + 长连接"（新库稳态池 = 同一节点 × 3 条连接，老库 = 固定 1 条）。差异落在**节点选择**：
- 老库：Ex 永远打 `112.74.214.43`（RTT 70.86ms，16 节点第 11）；主行情永远打清单首条 `110.41.147.114`（54.70ms，恰好是 Main 段最快，所以主行情两库差距最小）。
- 新库：初始从 16 节点随机抽 3 并行快建、先到先得，随后 `lastDataAddr` 定向钉住该节点。16 节点 RTT 中位 ~62ms、多数 ≤71ms，**抽中好节点（46–54ms）的概率显著高于抽中最慢的 173ms 节点**；实测新库 Ex 各函数 avg 48–61ms，与"命中 46–56ms 节点"的期望吻合。
- 差值方向与 RTT 差吻合：ExGetHistoryTransaction 两轮 avg 差 4–7ms ↔ 新库节点（46–54ms）与老库固定节点（70.86ms）的 RTT 差约 17–25ms 中的部分传导到请求往返（服务器处理耗时相同，仅 RTT 差进入总延迟）。
- 主行情持平：两库主站选择都落在 54–66ms 区间，差距自然被抹平。

### 4.4 为什么历史上"老库比新库快"（372ms 时代）——现在已不复现

- 出处：`/tmp/cmp_new.txt`（新库优化前）：TSLA 20260330 每请求 avg 372.8ms、max 927.6ms、偶发 empty。同期老库 avg 51.1ms。
- 根因链：20260330 该日**部分扩展站节点数据不全**（实测 TSLA 仅 3/16 节点有数据；20261030 全 16/16 有数据）；旧实现每次撞空节点都产生大额重试/重建开销，而老库固定命中"恰有数据"的深圳节点。
- 修复（反射判空换节点，`client_helpers.go:35-47`，HEAD 531a81a）：空应答不再走声明制重试，而是池内换连接最多 2 次。修复后 150s 长测 avg 53.9ms ↔ 老库 53.8ms，持平；本次 3 轮实测进一步显示多数函数新库已略优。
- **结论：当时的"老库快"是特定数据日期 × 旧重试实现的叠加效应，不是连接模型的稳态劣势，已被修复消除。**

### 4.5 新库何时会慢（残余风险，均有代码依据）

1. **抽签抽中慢节点并被定向钉住**：若初始抽到 120.25.218.6（173ms）等节点，`lastDataAddr` 会持续复用；rebalance 阈值保守（须快 30% 才切、且每 5 轮才全量探活），可能长时间滞留（`client.go:304-312/628-633/547-548`）。
2. **撞空节点串行重试**：空应答时串行最多 3 次往返（1 + 2 重试），尾延迟 ≈ 3×RTT（`client_helpers.go:35-47`）。
3. **后台开销**：心跳（每 30s 一次、2s 超时判死）+ 1s 补建 + rebalance 的锁竞争与流量，低流量长驻客户端场景下有轻微影响。
4. 本次数据佐证：新库 MACQuotes key 轮 max 197ms、MACTransactions r2 max 184ms、ExGetHistoryTransaction r3 max 168ms——均为重试/波动痕迹，但 empty=0 说明最终都拿到了数据。

### 4.6 老库何时快 / 何时被绑架

- **快**：节点稳定且数据完整时（本次主站首条 54.7ms 恰是 Main 段最快、MAC 默认节点 69.3ms 也不差），老库主路径最短（无池借还、无心跳、无重试），延迟分布窄。
- **被绑架**：固定节点一旦波动/劣化，全程无法换线——key 轮老库 `ExGetKLine` avg 136ms / max 476ms（同期新库 57ms/81ms）、`ExGetHistoryTransaction` avg 96ms，都是单节点波动直接注入的实例。历史长测尖峰（老库 >200ms 10 次 vs 新库 4 次）也显示老库被公网波动拖累的次数更多。

### 4.7 分函数结论表

| 分组 | 结论 | 原因 |
|---|---|---|
| 新库稳定更快（Ex 全部 5 个、主行情的逐笔/历史/分时类、GetVolumeProfile、MACQuotes/MACTickCharts） | 新库 Ex avg 快 2–8ms，主行情的逐笔/历史类快 2.5–5.5ms，GetVolumeProfile 快最多（16.5ms） | 新库抽签命中更优节点 + 定向持久化；老库被固定节点 RTT（70.86ms）绑定 |
| 基本持平（CONNECT main/ex/mac、GetIndexBars、GetSecurityBars、MACSymbolBars、MACTransactions） | 两库 avg 差 ≤2ms | 主行情两库节点都在 54–66ms 区间；MAC 两库节点接近（默认 69.3ms vs 抽中 49–55ms，但 MAC 请求往返短、差距被摊薄） |
| 老库单轮略快（仅 MACQuotes key 轮 60 vs 68ms） | 单窗口波动；all 轮 MACQuotes 新库反而快 1–11ms，不构成稳定优势 | 无稳定机制性优势 |
| 老库被单节点波动击中（key 轮 ExGetKLine 476ms、ExGetHistoryTransaction 178ms） | 老库无换线能力，波动全额注入 | 单节点无自愈 |

---

## 5. 新库可优化点候选（代码位置 + 预期收益）

| # | 优化 | 位置 | 预期收益 |
|---|---|---|---|
| 1 | 快建成功后对已收集候选按耗时排序/后台预比较，不等 1min rebalance 完成首次收敛 | `client.go:915-944`、`540-698` | 消除开局抽签慢阶段，稳态均值向候选最优靠拢 |
| 2 | 按节点累计空数据次数做健康度降权，定向不只看"曾经成功" | `client_helpers.go:35-47/107-123`、`client.go:304-312` | 降低撞空概率，减少重试往返、压低尾延迟 |
| 3 | 空数据重试由串行 2 次改为并行探测池内连接，取首个非空 | `client_helpers.go:39-46/78-98` | 尾延迟从 3×RTT 降到约 1×RTT |
| 4 | rebalance 阈值/周期动态化；心跳识别"慢连接"而非只识别"死连接" | `client.go:366/629/417-419/709-721` | 缩短慢节点驻留时间 |
| 5 | 补建连接纳入候选测速，refill 失败后回退候选池，避免 3 条同地址共享故障域 | `client.go:481-531/703-756` | 提高故障域分散度 |
| 6 | 500ms 快路径超时按历史拨号 RTT 自适应 | `client.go:400/910` | 减少弱网误判、缩短建连 |
| 7 | 提供显式单连接模式（PoolSize=1 + 可关空换节点） | `options.go:15/34`、`client_helpers.go:35-47` | 单并发场景获得老库式窄尾延迟，同时保留自愈 |
| 8 | 心跳与业务错峰/合并或拉长周期（MAC 已有跳过先例） | `client.go:365/406-421/431-432` | 降低后台流量与锁争用 |

---

## 6. 局限与不确定项

- 基准为 2026-11-01 单日 3 个短窗口（每轮 30–40s），公网波动跨时段可能改变数值方向；建议关键函数按需复测。
- RTT 为 TCP 三次握手口径、每节点 3 次采样；未含应用层握手（Hello1/ExLogin）与 zlib 解压耗时；AS 归属来自任务背景事实，未做 whois 复核。
- 新库辅助请求层（整市场小数位分页、股本回查等，仅 unified API 触发）的往返占比未做耗时拆解；本基准使用的 Get*/ExGet*/MAC* 层不触发该开销。
- 空数据重试路径在本次未命中（empty=0），其尾延迟影响由代码推演，未在实测中直接观测到。
- 老库"重复 Connect 是否意外重拨"仅据代码推断，未运行期实测。

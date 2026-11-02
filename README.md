# gotdx

<p align="center">通达信行情协议的 Go 客户端，覆盖主行情、扩展市场、MAC 板块/统一行情、F10/文件接口，以及一个可直接运行的 Web Viewer。</p>

<p align="center">
  <a href="#quickstart">快速开始</a> ·
  <a href="#capabilities">能力概览</a> ·
  <a href="#examples">示例</a> ·
  <a href="#viewer">Web Viewer</a> ·
  <a href="#api">API 速览</a> ·
  <a href="#testing">测试</a> ·
  <a href="TdxProtocol.md">协议文档</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.27%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.27+">
  <img src="https://img.shields.io/badge/Markets-Main%20%2F%20Extended-123B67" alt="Markets">
  <img src="https://img.shields.io/badge/MAC-Supported-0F766E" alt="MAC">
  <img src="https://img.shields.io/badge/WebViewer-Built--in-C86B36" alt="Web Viewer">
</p>

`gotdx` 把通达信协议里常用的查询能力整理成更偏 Go 的调用方式。你可以直接查主行情和扩展市场的快照、K 线、分时、逐笔，也可以继续往下钻到 F10、财务、板块文件、扩展表格和 MAC 板块协议；如果只是想先看接口返回长什么样，仓库里还带了一个可以直接打开的网页查看器。

## 为什么用 gotdx

- 一个 `Client` 同时管理主站和扩展市场，适合做统一监控和跨市场抓取。
- 提供两套入口：高阶统一接口 `Stock* / Ex* / MAC* / Goods*`，以及面向协议细节的底层 `Get* / ExGet*`。
- 地址池、超时、重试都可配，适合对接不稳定的真实行情站点。
- 内置主站、扩展、MAC、券商 host/IP 列表（主站 51 / 券商 23 / 扩展 16 / MAC 3 / MAC 扩展 18 / ICFQS 12，共 107 个唯一 IP），并支持 TCP 测速选最快节点；主站与券商池在官方公开清单、本地通达信 `connect.cfg` 基础上合并了 pytdx、injoyai/tdx 社区列表，经 500ms 并发拨号验证剔除失效节点后收录；MAC 扩展池按 MAC 协议握手实测收录 16 个扩展行情站（与 Ex 扩展池共享 7727 节点，协议握手不同）。
- 自带 30+ 个示例，覆盖从列表、快照、K 线、分时到 F10、板块、扩展表格。
- 自带 `cmd/webviewer`，不写代码也能直接调方法、填参数、看返回字段。
- 协议实现、示例和 Web Viewer 已整理为统一风格，便于继续补协议、核字段和排查差异。

<a id="quickstart"></a>
## 快速开始

### 安装

```bash
go get github.com/bensema/gotdx
```

### 最小示例

```go
package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/types"
)

func main() {
	mainHosts := gotdx.MainHostAddresses()
	exHosts := gotdx.ExHostAddresses()

	client := gotdx.New(
		gotdx.WithTCPAddress(mainHosts[0]),
		gotdx.WithTCPAddressPool(mainHosts[1:]...),
		gotdx.WithExTCPAddress(exHosts[0]),
		gotdx.WithExTCPAddressPool(exHosts[1:]...),
		gotdx.WithTimeoutSec(6),
	)
	defer client.Disconnect()

	stocks, err := client.StockQuotesDetail(
		[]uint8{types.MarketSZ.Uint8(), types.MarketSH.Uint8()},
		[]string{"000001", "600519"},
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, item := range stocks {
		log.Printf("stock code=%s price=%.2f turnover=%.2f%%", item.Code, item.Price, item.Turnover)
	}

	exQuotes, err := client.ExQuotes(
		[]uint8{types.ExCategoryUSStock},
		[]string{"TSLA"},
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, item := range exQuotes {
		log.Printf("ex: %+v", item)
	}
}
```

### 从哪套 API 开始

- 想尽快拿到业务数据：优先用 `Stock* / Ex* / MAC*` 统一高阶入口。
- 想逐个协议排查字段：用 `Get* / ExGet* / GetMAC*` 底层接口。
- 想先确认请求参数和返回格式：直接运行 `go run ./cmd/webviewer`。

### 内置 Host 列表与测速

```go
results := gotdx.ProbeHosts(gotdx.MainHosts(), time.Second)
for _, item := range results[:3] {
	log.Printf("main host=%s addr=%s reachable=%v latency=%s",
		item.Name, item.Address, item.Reachable, item.Latency)
}

fastest, err := gotdx.FastestHost(gotdx.ExHosts(), time.Second)
if err != nil {
	log.Fatal(err)
}
log.Printf("fastest ex host: %s %s", fastest.Name, fastest.Address)

client := gotdx.New(
	gotdx.WithTimeoutSec(6),
)
```

可用列表包括：`MainHosts`、`BrokerHosts`、`ExHosts`、`MACHosts`、`MACExHosts`，以及对应的 `*HostAddresses` 便捷函数。

### 连接池

`gotdx.Client` 以**动态连接池**与行情服务器通信：`Connect`（或 `ConnectEx` / `ConnectMAC`）按 `WithPoolSize` 配置池容量，建池走**并行快路径**——并发拨号配置前 3 个候选、收到第一个成功即返回（波动网络无需等待首地址超时即可命中可达服务器），其余候选已建成的连接入池复用；快路径拨号超时在样本不足时使用 **500ms 默认短超时**，样本充足后按历史建连耗时中位数自适应（在 200ms～2s 之间取值），服务器偶发秒级慢响应被直接排除而非踩中样本，全候选超时失败时走**流水线探活**（拨号成功即握手、首个成功即复用）兜底。池内余量由后台补建循环异步补齐；每条连接内部串行执行请求、连接之间并行，吞吐不受单连接往返限制。运行期间健康状态下**每分钟**重新探活并切换到当前最快的服务器，识别出慢节点后缩短到 **15 秒**一轮（候选收敛为池内地址与最近可达地址，每 5 次全量刷新一次）；另有**应用层心跳**——活跃连接 **30 秒**一轮、空闲连接 **60 秒**一轮，近 2 分钟内有业务活动的连接本轮直接跳过，2 秒短超时判定僵死连接并丢弃补建——与 **TCP keepalive**（15 秒周期，兜底无握手协议的 MAC 模式），网络波动或服务器迁移时连接池自动跟随。

连接池完整工作流如下（请求路径与后台维护并行运行）：

```mermaid
flowchart TD
    Start["构造 New / NewEx / NewMAC / NewMACEx"] --> Trig{"首次查询<br/>或显式 Connect*"}
    Trig -->|"是（建池）"| Build["建池：随机抽 3 候选并行拨号+握手"]
    Build -->|"首个成功即返回"| Pool[("连接池<br/>PoolSize 条")]
    Build -->|"全部失败"| Probe["流水线探活兜底"]
    Probe --> Pool
    Trig -->|"否（复用池）"| Req
    Pool --> Req["查询请求"]
    Req --> Acq{"定向持久化<br/>lastDataAddr 存在?"}
    Acq -->|"是"| Take["优先借最近成功节点连接"]
    Acq -->|"否"| Take2["轮询借出池内连接"]
    Take --> Exec["借出连接执行协议请求"]
    Take2 --> Exec
    Exec --> Empty{"应答 Count==0?"}
    Empty -->|"否"| Ok["成功：setDataAddr 记录节点<br/>release 归还"]
    Empty -->|"是"| Retry["并行探测池内其他连接"]
    Retry -->|"首个非空"| Ok
    Retry -->|"仍全空"| GiveUp["返回首次应答<br/>该日期确实无数据"]
    Ok --> Req
    GiveUp --> Req

    subgraph BG["后台维护"]
        Reb["重平衡：健康 1min / 慢节点 15s"] -->|"新节点延迟低于当前 90%"| Sw["切换更快节点"]
        HB["心跳：活跃 30s / 空闲 60s"] -->|"2s 超时判死"| Drop["丢弃僵死连接"]
        Drop --> Refill["补建循环：每秒补齐<br/>失败回退候选池"]
    end

    Down["Disconnect"] --> Close["关闭全部连接并停止后台循环"]
```

**工作机制**（调用方只需要构造客户端、必要时显式建连、用完 `Disconnect`，查询方法内部自动完成池内连接的借出、归还与换线）：

1. **建连**：`New*` 不拨号，首次查询或显式 `Connect*` 时才建池。建池走**并行快路径**：随机抽 3 个候选并发拨号并握手，第一个成功即返回（波动网络无需等待首地址超时即可命中可达服务器），其余已建成的连接入池复用；快路径超时在样本不足时用 500ms 默认值，样本充足后按历史建连耗时中位数自适应（200ms～2s），偶发秒级慢响应被直接排除；全部候选失败时走流水线探活兜底。
2. **首次收敛**：建池后立即按拨号+握手总耗时对已收候选排序，后台预比较提前切换到最快候选，不等每分钟重平衡完成首次收敛，开局即落在较优节点。
3. **定向持久化**：每次查询成功后记录"最近成功节点"，下次请求优先借出该节点的连接——请求稳定落在一个已确认可达的节点上，不会每请求重新测速；节点一旦劣化，由重平衡或心跳换线。
4. **空数据处理**：应答无记录（`Count==0`）时判定为空，**并行**探测池内其他连接、取首个非空应答（尾延迟从串行 3×RTT 降到约 1×RTT）；连续多次返回空的节点健康度降权，定向与轮询都会跳过。仍全空时返回首次应答（该日期确实无数据）。
5. **重平衡**：后台周期性探活切换更快节点——健康时每 1 分钟一轮，识别到慢节点后缩短到 15 秒一轮（候选收敛为池内地址与最近可达地址，每 5 次全量刷新）；新节点必须明显更优（默认阈值 0.9，即新节点延迟低于当前 90%）才切换，避免频繁换线。
6. **保活与自愈**：应用层心跳活跃连接 30 秒一轮、空闲连接 60 秒一轮（近 2 分钟有业务活动则跳过），2 秒短超时判定僵死连接并丢弃；TCP keepalive 15 秒兜底无握手协议的 MAC 模式。任一条连接被丢弃后，后台补建循环每秒检查池容量自动补齐，补建失败回退其他候选。
7. **释放**：`Disconnect` 关闭池内全部连接并停止后台循环，可重复调用。

### 空数据处理覆盖范围

判空换节点在唯一执行入口 `execute` 内实现，四个入口的全部行情/数据查询方法最终都汇聚到该入口（上层方法只做幂等建连与转包），因此**天然全覆盖**，无需逐函数接入：

```mermaid
flowchart TD
    M["主行情 New"] --> S["Stock* / Get* 组合层"]
    E["扩展市场 NewEx"] --> S2["Ex* / ExGet* 组合层"]
    MAC["MAC 主站 NewMAC"] --> S3["MAC* 上层<br/>幂等 ConnectMAC + 转包"]
    MEX["MAC 扩展站 NewMACEx"] --> S4["Goods*<br/>全部转包 Ex*/MAC*"]
    S --> B["底层协议方法<br/>Get* / ExGet* / GetMAC*"]
    S2 --> B
    S3 --> B
    S4 --> B
    B --> EX["execute 唯一入口"]
    EX -->|"应答空<br/>Count==0"| R["并行探测换节点 ≤2 次<br/>健康度降权 + 强制重平衡"]
    EX -->|"非空"| OK["setDataAddr 定向<br/>健康度清零"]
```

判空规则（`isEmptyReply`）：应答结构带 `Count uint16` 字段且为零（历史、分时、逐笔类）或带 `Returned uint32` 字段且为零（MAC 偏移 K 线）即判定为空；反射只读单个字段、开销可忽略，新增函数只要走 `execute` 就自动获得换节点能力。

| 类别 | 覆盖 | 说明 |
| --- | --- | --- |
| 主行情查询（`Stock*` / `Get*`） | 覆盖 | 直接或转包后 `return client.execute(obj)` |
| 扩展市场查询（`Ex*` / `ExGet*`） | 覆盖 | 含 `ExHistoryTransaction` |
| MAC 查询（`MAC*` / `GetMAC*`） | 覆盖 | 上层 = 幂等 `ConnectMAC` + 转包底层 |
| MAC 扩展站（`Goods*`） | 覆盖 | 全部转包 `Ex*` / `MAC*` 查询 |
| 文件下载（`DownloadFullFile` 等） | 不适用 | 空块是"文件下载完毕"标志（`Size==0` 结束），换节点破坏下载流 |
| 表格拉取（`ExGetTable` 等） | 不适用 | `Count==0` 表示表格已读完，是自身的终止条件 |
| 原始调试（`RawDataReply` 类） | 自动免疫 | 应答无 `Count` / `Returned` 字段，判空跳过，不误触发 |
| 订阅（`Subscriber`） | 不适用 | 长连接流式模型，不走 `execute` |
| 非结构体返回（`uint16` / `uint32` / `[]float64`） | 自动免疫 | 判空反射对非结构体返回 false |
| 连接与管理（`Connect` / `Disconnect` 等） | — | 无"数据为空"概念 |

单连接模式（`WithSingleConnectionMode`）或关闭换节点（`DisableEmptyNodeSwitch`）时，空应答直接返回首次结果，行为对齐老库固定单节点。

### 四个入口

四个入口的构造与建连方式一致，差异在握手机制与能力域：

| 入口 | 构造 | 显式建连 | 握手 | 能力域 |
| --- | --- | --- | --- | --- |
| 主行情 | `gotdx.New` | `Connect()` | Hello1 | A 股/指数快照、K 线、分时、逐笔、F10、文件 |
| 扩展市场 | `gotdx.NewEx` | `ConnectEx()` | ExLogin | 美股/港股/期货等扩展标的列表、报价、K 线、历史成交、表格 |
| MAC 主站 | `gotdx.NewMAC` | `ConnectMAC()` | 无（拨号即连） | 板块行情、成分报价、单只快照、逐笔、竞价、资金流向 |
| MAC 扩展站 | `gotdx.NewMACEx` | `ConnectMAC()` | MAC 协议握手 | MAC 扩展行情，查询方法同 MAC 协议族 |

```go
// 主行情入口
client := gotdx.New(gotdx.WithTimeoutSec(6))
if _, err := client.Connect(); err != nil {
	log.Fatal(err)
}
defer client.Disconnect()

stocks, err := client.StockQuotesDetail(
	[]uint8{types.MarketSZ.Uint8(), types.MarketSH.Uint8()},
	[]string{"000001", "600519"},
)
if err != nil {
	log.Fatal(err)
}
for _, item := range stocks {
	log.Printf("code=%s price=%.2f turnover=%.2f%%", item.Code, item.Price, item.Turnover)
}
```

```go
// 扩展市场入口（美股历史成交）
exClient := gotdx.NewEx(gotdx.WithTimeoutSec(6))
if _, err := exClient.ConnectEx(); err != nil {
	log.Fatal(err)
}
defer exClient.Disconnect()

items, err := exClient.ExHistoryTransaction(20260330, types.ExCategoryUSStock, "TSLA")
if err != nil {
	log.Fatal(err)
}
log.Printf("history items=%d", len(items))
```

```go
// MAC 主站入口（深市单只快照）
macClient := gotdx.NewMAC(gotdx.WithTimeoutSec(6))
if err := macClient.ConnectMAC(); err != nil {
	log.Fatal(err)
}
defer macClient.Disconnect()

quotes, err := macClient.MACQuotes(types.MarketSZ.Uint8(), "000001")
if err != nil {
	log.Fatal(err)
}
log.Printf("code=%s name=%s price=%.2f", quotes.Code, quotes.Name, quotes.Price)
```

```go
// MAC 扩展站入口（深市日 K，查询方法同 MAC 协议族；需 import "github.com/bensema/gotdx/types"）
macExClient := gotdx.NewMACEx(gotdx.WithTimeoutSec(6))
if err := macExClient.ConnectMAC(); err != nil {
	log.Fatal(err)
}
defer macExClient.Disconnect()

bars, err := macExClient.MACSymbolBars(
	types.MarketSZ.Uint8(), "000001",
	types.KLINE_TYPE_DAILY, 1, 0, 10, types.AdjustNone,
)
if err != nil {
	log.Fatal(err)
}
if len(bars) > 0 {
	log.Printf("bars=%d lastClose=%.2f", len(bars), bars[len(bars)-1].Close)
}
```

### 选项与场景配置

不传任何选项即用内置默认（内置地址池 + `PoolSize=3` + 超时 8 秒）；按场景调整：

```go
// 场景一：默认多连接池，并发抓取多市场；无需选项或只调超时
client := gotdx.NewEx(gotdx.WithTimeoutSec(6))

// 场景二：单并发、追求窄尾延迟 —— 单连接模式（池=1 + 关闭空数据换节点）
client := gotdx.NewEx(gotdx.WithSingleConnectionMode(), gotdx.WithTimeoutSec(6))

// 场景三：自定义地址池 + 预测速，把最快节点设为首选
fastest, err := gotdx.FastestHost(gotdx.ExHosts(), 3*time.Second)
if err != nil {
	log.Fatal(err)
}
client := gotdx.NewEx(
	gotdx.WithExTCPAddress(fastest.Address),
	gotdx.WithExTCPAddressPool(gotdx.ExHostAddresses()...),
	gotdx.WithTimeoutSec(6),
)
```

| 选项 | 默认 | 说明 |
| --- | --- | --- |
| `WithPoolSize(n)` | 3 | 池内并行 TCP 连接条数，越大并发吞吐越高 |
| `WithSingleConnectionMode()` | 关闭 | 池大小置 1 并关闭空数据换节点，见下方「单连接模式」 |
| `WithEmptyNodeSwitchDisabled(true)` | 关闭 | 单独关闭空数据换节点，不改变池大小 |
| `WithTCPAddress` / `WithTCPAddressPool` | 内置池 | 主行情单点地址 / 地址池 |
| `WithExTCPAddress` / `WithExTCPAddressPool` | 内置池 | 扩展行情单点地址 / 地址池 |
| `WithMacTCPAddress` / `WithMacTCPAddressPool` | 内置池 | MAC 主站单点地址 / 地址池 |
| `WithMacExTCPAddress` / `WithMacExTCPAddressPool` | 内置池 | MAC 扩展站单点地址 / 地址池 |
| `WithTimeoutSec(n)` | 8 秒 | 连接与读写共用超时，非正数忽略并回退默认 |

### 连接语义与并发

- 每条池内连接**内部串行**执行请求，**连接之间并行**，吞吐随 `PoolSize` 增长（默认 3 条）；
- 借出与归还由查询方法内部完成，所有公开方法**并发安全**，可直接多 goroutine 同时调用；
- 显式 `Connect*` 幂等：已有池直接复用，不重复拨号；
- `Disconnect` 幂等：关闭全部连接并停止补建、心跳、重平衡循环；
- `CurrentAddress` 返回池内首条连接实际连通的服务器地址，用于观测与日志。

### 单连接模式

单并发、追求窄尾延迟的调用方可以开启显式单连接模式：`WithSingleConnectionMode()` 同时把池大小置为 1 并关闭空数据换节点，全程复用一条长连接，获得接近老库固定单连接的稳定延迟分布；断连自愈（心跳判死、后台补建、重平衡）仍然保留。只想关闭空数据换节点、保留多连接池时，使用 `WithEmptyNodeSwitchDisabled(true)`，两个选项相互独立；默认关闭，不传时保持三连接池与空数据并行换节点行为。

```go
client := gotdx.NewEx(
	gotdx.WithSingleConnectionMode(), // 池大小 1 + 关闭空数据换节点
)
if _, err := client.ConnectEx(); err != nil {
	log.Fatal(err)
}
defer client.Disconnect()
```

可运行示例见 `examples/ex_single_connection`。

### 行情订阅

`Client.Subscriber()` 返回的 `Subscriber` 通用订阅工厂，对任意数据源建立"轮询 diff 推送"订阅：内部按间隔轮询拉取快照（复用各查询方法请求协议，无需新增推送协议），快照变化时回调处理函数。首次订阅立即推送一次建立基线，此后仅在变化时推送；处理函数返回 `false` 或调用返回的取消函数即停止订阅。

```go
sub := client.Subscriber()

// 报价订阅：复用 GetSecurityQuotes 拉取链路。
cancel, err := sub.Subscribe(
	gotdx.SubscribeSource[*proto.GetSecurityQuotesReply]{
		Fetch: func() (*proto.GetSecurityQuotesReply, error) {
			return client.GetSecurityQuotes([]uint8{types.MarketSH}, []string{"600000"})
		},
	},
	3*time.Second, // 轮询间隔，非正数使用默认 3 秒
	func(q *proto.GetSecurityQuotesReply) bool {
		log.Printf("%s 最新价 %.2f", q.List[0].Code, q.List[0].Close)
		return true // 返回 false 取消订阅
	},
)
if err != nil {
	log.Fatal(err)
}
defer cancel()
```

K 线、分时、逐笔、指数、扩展市场等所有查询能力用同一套订阅语义接入：`SubscribeSource[T]` 中 `Fetch` 提供拉取函数（任意查询方法），`Changed` 可选自定义变化判定（如"仅新 K 线出现时推送""价格变动超过阈值才推送"），省略时默认深比较：

```go
sub.Subscribe(
	gotdx.SubscribeSource[*proto.GetSecurityBarsReply]{
		Fetch: func() (*proto.GetSecurityBarsReply, error) {
			return client.GetKLine(0, types.MarketSH, "600000", 0, 10, 0, 0)
		},
		Changed: func(old, new *proto.GetSecurityBarsReply) bool {
			return len(new.List) > len(old.List) // 仅新 K 线出现时推送
		},
	},
	5*time.Second,
	func(k *proto.GetSecurityBarsReply) bool {
		log.Printf("最新收盘 %.2f", k.List[0].Close)
		return true
	},
)
```

工厂基座能力：相同 `Key` 的订阅合并到同一轮询任务、一次拉取服务多个处理函数（避免重复轮询），未指定 `Key` 时每次订阅独立轮询；合并限定在同一 `Subscriber` 实例内，且追加订阅会立即收到当前快照建立基线（不等到数据变化）；轮询间隔每轮实时生效，缩小间隔即时提速。拉取失败自动跳过本轮、下轮重试，订阅不中断；拉取函数与处理函数的 panic 均被隔离，不影响其他订阅与进程。停止路径：返回的取消函数、处理函数返回 `false`、工厂 `Close` 或客户端 `Disconnect` 均可收敛轮询循环，不泄漏 goroutine。

<a id="capabilities"></a>
## 能力概览

| 模块 | 典型能力 | 代表接口 |
| --- | --- | --- |
| 主行情 | 股票/指数列表、快照、K 线、分时、逐笔、指数工具、异动、集合竞价 | `StockQuotesDetail`, `StockKLine`, `StockIndexInfo`, `StockUnusual`, `StockAuction` |
| 扩展市场 | 美股/港股/期货等扩展标的列表、报价、K 线、历史成交、表格 | `ExQuotes`, `ExKLine`, `ExHistoryTransaction`, `ExTable` |
| TDX 商品语义 | 对齐 `goods_*` 命名的商品总览、列表、报价、K 线、分时、历史成交 | `GoodsCount`, `GoodsList`, `GoodsVarieties`, `GoodsQuotes`, `GoodsKLine` |
| F10 与文件 | 公司信息分类、正文、财务、除权除息、文件下载、板块文件 | `GetCompanyInfo`, `GetFinanceInfo`, `GetXDXRInfo`, `DownloadFullFile` |
| MAC 协议 | 板块列表、成分股、成分报价、动态成分报价、批量股票报价、单只快照、逐笔成交、竞价、多日分时、股票摘要、资金流向、服务端交易时段、K线偏移信息、文件查询/下载、市场监控、所属板块、统一 K 线 | `MACBoardList`, `MACBoardMembers`, `MACBoardMembersWithSort`, `MACBoardMembersQuotes`, `MACBoardMembersQuotesWithSort`, `MACBoardMembersQuotesDynamic`, `MACBoardMembersQuotesDynamicWithFilter`, `MACSymbolQuotes`, `MACQuotes`, `MACTransactions`, `MACAuction`, `MACTickCharts`, `MACSymbolInfo`, `MACCapitalFlow`, `MACServerInfo`, `MACKLineOffset`, `MACMarketMonitor`, `MACSymbolBars` |
| ICFQS HTTP | 主题投资、主题行情、批量行情、龙虎榜、每日复盘等 TQLEX/JSON 原始接口 | `NewICFQS`, `PostTQL`, `ICFQSTopicListRaw`, `ICFQSQuotesBatchRaw`, `ICFQSMRFPRaw` |
| 协议调试 | 原始协议响应、扩展实验接口、网页查看器 | `MainTodoB`, `MainClient26AD`, `ExExperiment2487`, `cmd/webviewer` |

## 项目结构

- `proto/`: 各协议的请求/响应序列化与反序列化实现。
- `cache/`: 基于 Redis 的缓存层基座（机制底座），按数据形态分两类存储：时间序列能力（K 线/分时/逐笔/竞价/MAC 时序）采用「ZSet 闭合历史 + String 实时」双结构并走 Lua 原子流转，ZSet 永久保存不自动淘汰；行情快照/列表/服务器/公司资料/文件等采用整段键值缓存；两类均含空标记防穿透、singleflight+分布式锁防击穿、熔断防雪崩，详见「缓存层」一节。
- `client_quote.go`: 主行情底层接口。
- `client_exquote.go`: 扩展市场底层接口。
- `client_mac.go`: MAC 协议接口。
- `client_unified.go` / `client_goods.go`: `Stock* / Ex* / MAC* / Goods*` 高阶统一入口。
- `cmd/webviewer/`: 浏览器调试界面。
- `examples/`: 可直接运行的示例。

<a id="examples"></a>
## 缓存层

`cache` 包为 gotdx 提供基于 Redis 的缓存层**基座**：当前已实现存储机制与防护底座，
对外查询能力（与 `gotdx.Client` 同名方法）在此基座上逐能力族构建，每个能力族一个
源文件。所有机制并发安全。

**时间序列能力（K 线、分时、逐笔、历史成交、竞价、MAC 时序等）**——凡属
「最新一根未闭合 + 历史已闭合」形态的数据，采用 **ZSet 闭合历史 + String 实时**
双结构存储：

- 未闭合的最新一根存入普通 String（键为 ZSet 键加 `:live` 后缀），值为
  「分数|更新时刻|base64 数据」三段拼接文本；tick 到达时由单条 Lua 脚本原子完成
  「同根覆盖 / 前一根闭合流转进 ZSet / 续写 TTL」，TTL 仅作行情源断流兜底。
- 已闭合的根按 score 存入 ZSet：score 统一为 Unix 毫秒（时间戳换算），member 为
  单根应答 MessagePack 编码后再 base64 的文本，与实时槽编码一致、可被 Lua 原样搬移。
- **ZSet 永久保存，不自动淘汰**（无后台任务、无 `Stop()`）；如需清理只能外部显式
  `DEL` 或 `ZREMRANGEBYSCORE`。
- 历史修正写入由 Lua 脚本原子完成：先按 score 删除同 score 旧成员再 ZADD（修正覆盖
  而不堆积）；逐笔、成交等「同分可多笔共存」的族以追加写入绕过同分唯一约束。
- 查询用一条 pipeline 同时完成空标记检查、实时槽读取与 ZSet 最新段读取（单次往返），
  合并后按分数降序（时间从新到旧）排序，按 start/count 窗口切片返回；实时槽超过
  刷新间隔未更新时拉一次最新一根做 tick 更新（同样受熔断保护）。

**普通键值能力（行情快照、证券列表、榜单、服务器信息、公司资料、文件等）**——整段
应答 MessagePack 编码后存入 Redis String（信封带版本号便于格式迁移），附带空标记，
不设体积上限。

**防护机制（两类通用）**：

- 穿透防护：回源失败或结果为空时写入短 TTL 空标记（默认 30 秒），存活期内不再回源，
  防恶意穿透打垮行情源；失败与空结果用不同标记值区分。
- 击穿防护：同进程 `singleflight` 合并同键并发；跨进程抢 Redis 分布式锁
  （SET NX EX，5 秒超时防死锁），持锁方二次确认后才回源，以 Lua 校验 token 解锁；
  抢锁失败直连回源兜底，同键并发只打一次行情服务器。
- 熔断防护：回源连续失败达到阈值（默认 5 次）打开熔断，冷却期内（默认 10 秒）未命中
  查询快速失败返回 `ErrCircuitOpen`，不再打行情源；冷却期满放行单个试探请求，成功即
  关断、失败重新打开。
- 降级：Redis 故障时自动退化为直连回源（查询、锁、熔断三层全部兜底），不因缓存层
  故障阻断行情。

```go
import (
    "github.com/bensema/gotdx"
    "github.com/bensema/gotdx/cache"
    "github.com/redis/go-redis/v9"
)

cli := gotdx.New()                    // 行情客户端，按需传 gotdx.WithTCPAddress 等选项，使用前 Connect
rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
cached := cache.NewCacheClient(rdb, cli) // 可传 cache.WithKeyPrefix、cache.WithEmptyTTL、
                                         // cache.WithLockTimeout、cache.WithLiveTTL、
                                         // cache.WithCircuitFailThreshold、cache.WithCircuitOpenTimeout
// 非法选项值（阈值 0、负 TTL、空前缀）自动回退默认值。
// 能力层在此基座上按族构建（每个能力族一个源文件，覆盖 gotdx 全部查询能力）；
// 构建后调用方式与 gotdx.Client 同名方法一致，仅首位新增 context.Context。

// 运行时观测：Stats() 返回命中/空标记命中/回源/回源失败/熔断拒绝五项原子计数快照，
// 可据此计算缓存命中率并判断行情源健康度。
stats := cached.Stats()
fmt.Println(stats.Hits, stats.EmptyHits, stats.Fetches, stats.SourceErrors, stats.CircuitRejects)
```

基座文件：`client.go`（入口与配置）、`kv.go`（键值缓存机制）、`ts.go`（时序双结构
机制）、`lua.go`（原子脚本）、`breaker.go`（回源熔断器）。

## 示例

下面这些命令可以直接运行：

| 场景 | 命令 |
| --- | --- |
| 主行情快照 | `go run ./examples/stock_quotes` |
| 批量主行情 | `go run ./examples/stock_batch_quotes` |
| 列表与分页遍历 | `go run ./examples/stock_list` / `go run ./examples/stock_paged_list` |
| K 线与指数工具 | `go run ./examples/stock_kline` / `go run ./examples/stock_index_tools` |
| 分时、历史分时、逐笔 | `go run ./examples/stock_tick` / `go run ./examples/stock_history` / `go run ./examples/stock_transaction` |
| F10、公司资料、板块文件 | `go run ./examples/stock_f10_block` / `go run ./examples/stock_company_raw` / `go run ./examples/stock_block_raw` |
| 市场监控 | `go run ./examples/stock_market_watch` |
| 主机测速与地址池 | `go run ./examples/host_probe` |
| 主站服务与试验协议 | `go run ./examples/stock_server_info` / `go run ./examples/main_experimental` |
| 主行情兼容协议 | `go run ./examples/stock_list_old` / `go run ./examples/stock_feature_452` / `go run ./examples/stock_quotes_encrypt` / `go run ./examples/stock_kline_offset` / `go run ./examples/stock_history_transaction_with_trans` |
| 扩展市场单只/批量报价 | `go run ./examples/ex_quote` / `go run ./examples/ex_quotes` / `go run ./examples/ex_quotes2` |
| 扩展市场列表与分类 | `go run ./examples/ex_count` / `go run ./examples/ex_list` / `go run ./examples/ex_paged_list` / `go run ./examples/ex_category_list` |
| 扩展市场 K 线、分时、历史成交 | `go run ./examples/ex_kline` / `go run ./examples/ex_tick` / `go run ./examples/ex_history` |
| 扩展市场单连接模式 | `go run ./examples/ex_single_connection` |
| 扩展试验与补充协议 | `go run ./examples/ex_list_extra` / `go run ./examples/ex_board_list` / `go run ./examples/ex_experiment_2487` / `go run ./examples/ex_experiment_2488` / `go run ./examples/ex_kline2` / `go run ./examples/ex_mapping_2562` |
| 扩展市场表格 | `go run ./examples/ex_table` / `go run ./examples/ex_table_detail` |
| TDX 商品语义入口 | `go run ./examples/goods_overview` |
| MAC 协议 | `go run ./examples/mac_board_list` / `go run ./examples/mac_board_members` / `go run ./examples/mac_board_members_quotes` / `go run ./examples/mac_board_members_quotes_dynamic` / `go run ./examples/mac_symbol_quotes` / `go run ./examples/mac_market_monitor` / `go run ./examples/mac_quotes` / `go run ./examples/mac_transactions` / `go run ./examples/mac_auction` / `go run ./examples/mac_tick_charts` / `go run ./examples/mac_symbol_info` / `go run ./examples/mac_capital_flow` / `go run ./examples/mac_server_info` / `go run ./examples/mac_kline_offset` / `go run ./examples/mac_file_query` / `go run ./examples/mac_symbol_belong_board` / `go run ./examples/mac_symbol_bars` |
| ICFQS HTTP 原始接口 | `go run ./examples/icfqs_raw` |
| 统一监控示例 | `go run ./examples/unified_watchlist` |

<details>
<summary>查看完整示例目录</summary>

- `examples/stock_count`
- `examples/stock_server_info`
- `examples/stock_list`
- `examples/stock_list_old`
- `examples/stock_paged_list`
- `examples/stock_batch_quotes`
- `examples/stock_quotes`
- `examples/stock_quotes_encrypt`
- `examples/stock_lowlevel_quote`
- `examples/stock_quotes_list`
- `examples/stock_kline`
- `examples/stock_kline_offset`
- `examples/stock_feature_452`
- `examples/stock_index_tools`
- `examples/stock_tick`
- `examples/stock_history`
- `examples/stock_history_transaction_with_trans`
- `examples/stock_market_watch`
- `examples/host_probe`
- `examples/icfqs_raw`
- `examples/stock_transaction`
- `examples/stock_f10_block`
- `examples/stock_company_raw`
- `examples/stock_block_raw`
- `examples/main_experimental`
- `examples/ex_count`
- `examples/ex_quote`
- `examples/ex_list`
- `examples/ex_list_extra`
- `examples/ex_paged_list`
- `examples/ex_quotes`
- `examples/ex_quotes2`
- `examples/ex_quotes_list`
- `examples/ex_kline`
- `examples/ex_kline2`
- `examples/ex_board_list`
- `examples/ex_experiment_2487`
- `examples/ex_experiment_2488`
- `examples/ex_mapping_2562`
- `examples/goods_overview`
- `examples/ex_history`
- `examples/ex_single_connection`
- `examples/ex_tick`
- `examples/ex_server_info`
- `examples/ex_sampling`
- `examples/ex_category_list`
- `examples/ex_table`
- `examples/ex_table_detail`
- `examples/mac_board_list`
- `examples/mac_board_members`
- `examples/mac_board_members_quotes`
- `examples/mac_board_members_quotes_dynamic`
- `examples/mac_symbol_quotes`
- `examples/mac_market_monitor`
- `examples/mac_quotes`
- `examples/mac_transactions`
- `examples/mac_auction`
- `examples/mac_tick_charts`
- `examples/mac_symbol_info`
- `examples/mac_capital_flow`
- `examples/mac_server_info`
- `examples/mac_kline_offset`
- `examples/mac_file_query`
- `examples/mac_symbol_belong_board`
- `examples/mac_symbol_bars`
- `examples/unified_watchlist`

</details>

<a id="viewer"></a>
## Web Viewer

仓库内置了一个轻量的网页查看器，适合在这些场景下使用：

- 先确认某个方法应该填哪些参数。
- 快速查看返回字段，而不是先写一段测试代码。
- 对照 MAC 动态位图字段，直接看 `bit/name/format/description`，也可以用 TDX 商品语义分组调 `goods_*` 入口。
- 直接检查主站文件、表格文件和 CSV 文件的结构化结果。
- 对比不同主机返回的数据差异。
- 调试实验协议或原始接口。

![gotdx Web Viewer 截图](docs/images/webviewer-screenshot.png)

启动：

```bash
go run ./cmd/webviewer
```

默认地址：

```text
http://127.0.0.1:8080
```

Web Viewer 中的 ICFQS HTTP 原始接口已按 `主题投资`、`行情`、`龙虎榜`、`每日复盘` 分组；多表响应可用 `table_index` 选择展示的 `ResultSets`，完整响应保留在 Raw 面板。

<a id="api"></a>
## API 速览

### 高阶统一入口

- `StockQuotesDetail`、`StockQuotesList`、`StockQuotes` 会按证券列表里的 `DecimalPoint` 尽力修正非 2 位小数价格，并在可获取到流通股本时补齐 `Turnover`。
- 主行情：`StockCount`, `StockList`, `StockQuotesDetail`, `StockKLine`, `StockTickChart`, `StockIndexInfo`, `StockIndexMomentum`, `StockChartSampling`, `StockAuction`, `StockTopBoard`, `StockUnusual`, `StockVolumeProfile`, `StockHistoryOrders`, `StockHistoryTransaction`, `StockF10`
- 扩展市场：`ExCount`, `ExList`, `ExQuote`, `ExQuotes`, `ExKLine`, `ExTickChart`, `ExHistoryTransaction`, `ExTable`
- 商品语义：`GoodsCount`, `GoodsCategoryList`, `GoodsList`, `GoodsVarieties`, `GoodsQuote`, `GoodsQuotes`, `GoodsQuotesList`, `GoodsKLine`, `GoodsTickChart`, `GoodsChartSampling`, `GoodsHistoryTransaction`
- MAC：`MACBoardList`, `MACBoardMembers`, `MACBoardMembersWithSort`, `MACBoardMembersQuotes`, `MACBoardMembersQuotesWithSort`, `MACBoardMembersQuotesDynamic`, `MACBoardMembersQuotesDynamicWithFilter`, `MACSymbolQuotes`, `MACQuotes`, `MACQuotesWithDate`, `MACTransactions`, `MACTransactionsWithDate`, `MACAuction`, `MACTickCharts`, `MACSymbolInfo`, `MACCapitalFlow`, `MACFileList`, `MACDownloadFullFile`, `MACMarketMonitor`, `MACSymbolBelongBoard`, `MACSymbolBars`；`MACSymbolBars` 会基于返回的流通股本字段尽力补齐 `Turnover`。

### 常用底层接口

- 主行情：`GetSecurityCount`, `GetSecurityListRange`, `GetQuotesDetail`, `GetIndexInfo`, `GetIndexMomentum`, `GetVolumeProfile`, `GetMinuteTimeData`, `GetAuction`, `GetTopBoard`, `GetUnusual`, `GetTransactionData`, `GetHistoryOrders`
- F10/文件：`GetCompanyCategories`, `GetCompanyContent`, `GetFinanceInfo`, `GetXDXRInfo`, `DownloadFile`, `GetBlockFile`
- 扩展市场：`ExGetCategoryList`, `ExGetQuotesList`, `ExGetQuote`, `ExGetChartSampling`, `ExGetFileMeta`, `ExDownloadFile`
- MAC：`GetMACBoardCount`, `GetMACBoardList`, `GetMACBoardMembers`, `GetMACBoardMembersQuotes`, `GetMACBoardMembersQuotesDynamic`, `GetMACBoardMembersQuotesDynamicWithFilter`, `GetMACSymbolQuotes`, `GetMACQuotes`, `GetMACQuotesWithDate`, `GetMACTransactions`, `GetMACTransactionsWithDate`, `GetMACAuction`, `GetMACTickCharts`, `GetMACSymbolInfo`, `GetMACCapitalFlow`, `GetMACFileList`, `GetMACFileDownload`, `GetMACMarketMonitor`, `GetMACSymbolBelongBoard`, `GetMACSymbolBars`

### 地址池与测速

- 内置列表：`MainHosts`, `BrokerHosts`, `ExHosts`, `MACHosts`, `MACExHosts`
- 地址便捷函数：`MainHostAddresses`, `BrokerHostAddresses`, `ExHostAddresses`, `MACHostAddresses`, `MACExHostAddresses`
- 测速函数：`ProbeHosts`, `ProbeAddresses`, `FastestHost`, `FastestAddress`
- 客户端能力：`Client.ProbeHosts`, `Client.FastestHost`

### 试验与兼容协议

- 主站试验：`MainTodoB`, `MainTodoFDE`, `MainClient264B`, `MainClient26AC`, `MainClient26AD`, `MainClient26AE`, `MainClient26B1`
- 主站兼容：`StockListOld`, `StockFeature452`, `StockQuotesEncrypt`, `StockKLineOffset`, `StockHistoryTransactionWithTrans`
- 扩展试验：`ExListExtra`, `ExExperiment2487`, `ExExperiment2488`, `ExKLine2`, `ExMapping2562`
- TDX 商品语义：`GoodsCount`, `GoodsCategoryList`, `GoodsList`, `GoodsVarieties`, `GoodsQuote`, `GoodsQuotes`, `GoodsQuotesList`, `GoodsKLine`, `GoodsTickChart`, `GoodsChartSampling`, `GoodsHistoryTransaction`

<details>
<summary>查看更完整的接口分组</summary>

#### 主行情与通用接口

- `Connect`, `Disconnect`
- `GetExchangeAnnouncement`, `GetServerHeartbeat`, `GetAnnouncement`, `GetServerInfo`
- `GetSecurityCount`, `GetSecurityQuotes`, `GetQuotesDetail`, `GetQuotesEncrypt`
- `GetSecurityList`, `GetSecurityListOld`, `GetSecurityListRange`
- `GetKLine`, `GetSecurityBars`, `GetSecurityBarsOffset`, `GetIndexBars`
- `GetIndexMomentum`, `GetIndexInfo`, `GetMinuteTimeData`, `GetTickChart`, `GetHistoryMinuteTimeData`, `GetHistoryTickChart`
- `GetChartSampling`, `GetAuction`, `GetTopBoard`, `GetUnusual`
- `GetTransactionData`, `GetHistoryOrders`, `GetHistoryTransactionData`, `GetHistoryTransactionDataWithTrans`

#### F10、公司资料与文件

- `GetCompanyCategories`, `GetCompanyContent`, `GetFinanceInfo`, `GetXDXRInfo`, `GetCompanyInfo`
- `GetFileMeta`, `DownloadFile`, `DownloadFullFile`
- `GetBlockFile`, `GetTableFile`, `GetCSVFile`, `GetParsedBlockFile`, `GetGroupedBlockFile`

#### 扩展市场

- `ConnectEx`, `GetExServerInfo`
- `ExGetCount`, `ExGetCategoryList`, `ExGetList`, `ExGetListExtra`
- `ExGetQuotesList`, `ExGetQuote`, `ExGetQuotes`, `ExGetQuotes2`
- `ExGetKLine`, `ExGetKLine2`
- `ExGetHistoryTransaction`, `ExGetTickChart`, `ExGetHistoryTickChart`, `ExGetChartSampling`
- `ExGetBoardList`, `ExGetMapping2562`
- `ExGetFileMeta`, `ExDownloadFile`, `ExDownloadFullFile`
- `ExGetTable`, `ExGetTableDetail`

#### MAC 协议

- `NewMAC`, `NewMACEx`, `ConnectMAC`
- `GetMACBoardCount`, `GetMACBoardList`, `GetMACBoardMembers`, `GetMACBoardMembersQuotes`, `GetMACBoardMembersQuotesDynamic`, `GetMACBoardMembersQuotesDynamicWithFilter`, `GetMACSymbolQuotes`, `GetMACQuotes`, `GetMACQuotesWithDate`, `GetMACTransactions`, `GetMACTransactionsWithDate`, `GetMACAuction`, `GetMACTickCharts`, `GetMACSymbolInfo`, `GetMACCapitalFlow`, `GetMACServerInfo`, `GetMACKLineOffset`, `GetMACFileList`, `GetMACFileDownload`, `GetMACMarketMonitor`
- `GetMACSymbolBelongBoard`, `GetMACSymbolBars`
- `MACBoardCount`, `MACBoardList`, `MACBoardMembers`, `MACBoardMembersWithSort`, `MACBoardMembersQuotes`, `MACBoardMembersQuotesWithSort`, `MACBoardMembersQuotesDynamic`, `MACBoardMembersQuotesDynamicWithFilter`, `MACSymbolQuotes`, `MACQuotes`, `MACQuotesWithDate`, `MACTransactions`, `MACTransactionsWithDate`, `MACAuction`, `MACTickCharts`, `MACSymbolInfo`, `MACCapitalFlow`, `MACServerInfo`, `MACKLineOffset`, `MACFileList`, `MACDownloadFullFile`, `MACMarketMonitor`
- `MACSymbolBelongBoard`, `MACSymbolBars`
- 位图辅助：`MACFieldBit`, `MACPresetField`, `MACSortType`, `MACSortOrder`, `MACFilterType`, `MACFieldBitmap`, `MACFilterFlags`, `MACBoardMembersQuotesRequestBitmap`, `DefaultMACBoardMembersQuotesFieldBitmap`, `FullMACBoardMembersQuotesFieldBitmap`, `MACBoardMembersQuotesFieldBitmapFromBits`, `DefaultMACSymbolQuotesFieldBitmap`, `FullMACSymbolQuotesFieldBitmap`, `MACSymbolQuotesFieldBitmapFromBits`

#### ICFQS HTTP

- `NewICFQS`, `NewICFQSHot`, `WithICFQSAddress`, `WithICFQSTimeout`, `ICFQSHosts`, `ICFQSHostAddresses`
- `PostTQL`, `PostJSON`, `ICFQSFormatTables`
- `ICFQSTopicListRaw`, `ICFQSSearchTopicsRaw`, `ICFQSNewTopicsRaw`, `ICFQSHotTopicsRaw`, `ICFQSEventsRaw`, `ICFQSTopTopicsRaw`, `ICFQSTopicDetailRaw`, `ICFQSTopicKLineRaw`, `ICFQSTopicStocksRaw`
- `ICFQSTopicQuotesRaw`, `ICFQSTopicRotationRaw`, `ICFQSQuotesBatchRaw`
- `ICFQSLHBDetailRaw`, `ICFQSYYBDetailRaw`, `ICFQSYZDetailRaw`, `ICFQSMRFPRaw`, `ICFQSMRFPLatestDateRaw`

</details>

<a id="testing"></a>
## 测试

单元测试：

```bash
go test ./...
```

连接真实站点的集成测试默认跳过；如需开启：

```bash
GOTDX_INTEGRATION=1 go test ./...
```

## 相关文档

- [TdxProtocol.md](TdxProtocol.md): 协议分析笔记
- [docs/images/webviewer-screenshot.png](docs/images/webviewer-screenshot.png): Web Viewer 截图

## License

本项目使用 MIT License 开源，详见 [LICENSE](LICENSE)。

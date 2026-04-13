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
  <img src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.26+">
  <img src="https://img.shields.io/badge/Markets-Main%20%2F%20Extended-123B67" alt="Markets">
  <img src="https://img.shields.io/badge/MAC-Supported-0F766E" alt="MAC">
  <img src="https://img.shields.io/badge/WebViewer-Built--in-C86B36" alt="Web Viewer">
</p>

`gotdx` 把通达信协议里常用的查询能力整理成更偏 Go 的调用方式。你可以直接查主行情和扩展市场的快照、K 线、分时、逐笔，也可以继续往下钻到 F10、财务、板块文件、扩展表格和 MAC 板块协议；如果只是想先看接口返回长什么样，仓库里还带了一个可以直接打开的网页查看器。

## 为什么用 gotdx

- 一个 `Client` 同时管理主站和扩展市场，适合做统一监控和跨市场抓取。
- 提供两套入口：高阶统一接口 `Stock* / Ex* / MAC*`，以及面向协议细节的底层 `Get* / ExGet*`。
- 地址池、超时、重试都可配，适合对接不稳定的真实行情站点。
- 内置主站、扩展、MAC、券商 host/IP 列表，并支持 TCP 测速选最快节点。
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
)

func main() {
	mainHosts := gotdx.MainHostAddresses()
	exHosts := gotdx.ExHostAddresses()

	client := gotdx.New(
		gotdx.WithTCPAddress(mainHosts[0]),
		gotdx.WithTCPAddressPool(mainHosts[1:]...),
		gotdx.WithExTCPAddress(exHosts[0]),
		gotdx.WithExTCPAddressPool(exHosts[1:]...),
		gotdx.WithAutoSelectFastest(true),
		gotdx.WithTimeoutSec(6),
	)
	defer client.Disconnect()

	stocks, err := client.StockQuotesDetail(
		[]uint8{gotdx.MarketSZ, gotdx.MarketSH},
		[]string{"000001", "600519"},
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, item := range stocks {
		log.Printf("stock code=%s price=%.2f turnover=%.2f%%", item.Code, item.Price, item.Turnover)
	}

	exQuotes, err := client.ExQuotes(
		[]uint8{gotdx.ExCategoryUSStock},
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
	gotdx.WithAutoSelectFastest(true),
	gotdx.WithTimeoutSec(6),
)
```

可用列表包括：`MainHosts`、`BrokerHosts`、`ExHosts`、`MACHosts`、`MACExHosts`，以及对应的 `*HostAddresses` 便捷函数。

<a id="capabilities"></a>
## 能力概览

| 模块 | 典型能力 | 代表接口 |
| --- | --- | --- |
| 主行情 | 股票/指数列表、快照、K 线、分时、逐笔、指数工具、异动、集合竞价 | `StockQuotesDetail`, `StockKLine`, `StockIndexInfo`, `StockUnusual`, `StockAuction` |
| 扩展市场 | 美股/港股/期货等扩展标的列表、报价、K 线、历史成交、表格 | `ExQuotes`, `ExKLine`, `ExHistoryTransaction`, `ExTable` |
| F10 与文件 | 公司信息分类、正文、财务、除权除息、文件下载、板块文件 | `GetCompanyInfo`, `GetFinanceInfo`, `GetXDXRInfo`, `DownloadFullFile` |
| MAC 协议 | 板块列表、成分股、成分报价、所属板块、统一 K 线 | `MACBoardList`, `MACBoardMembers`, `MACBoardMembersQuotes`, `MACSymbolBars` |
| 协议调试 | 原始协议响应、扩展实验接口、网页查看器 | `MainTodoB`, `MainClient26AD`, `ExExperiment2487`, `cmd/webviewer` |

## 项目结构

- `proto/`: 各协议的请求/响应序列化与反序列化实现。
- `client_quote.go`: 主行情底层接口。
- `client_exquote.go`: 扩展市场底层接口。
- `client_mac.go`: MAC 协议接口。
- `client_unified.go`: `Stock* / Ex* / MAC*` 高阶统一入口。
- `cmd/webviewer/`: 浏览器调试界面。
- `examples/`: 可直接运行的示例。

<a id="examples"></a>
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
| 扩展试验与补充协议 | `go run ./examples/ex_list_extra` / `go run ./examples/ex_board_list` / `go run ./examples/ex_experiment_2487` / `go run ./examples/ex_experiment_2488` / `go run ./examples/ex_kline2` / `go run ./examples/ex_mapping_2562` |
| 扩展市场表格 | `go run ./examples/ex_table` / `go run ./examples/ex_table_detail` |
| MAC 协议 | `go run ./examples/mac_board_list` / `go run ./examples/mac_board_members` / `go run ./examples/mac_board_members_quotes` / `go run ./examples/mac_symbol_belong_board` / `go run ./examples/mac_symbol_bars` |
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
- `examples/ex_history`
- `examples/ex_tick`
- `examples/ex_server_info`
- `examples/ex_sampling`
- `examples/ex_category_list`
- `examples/ex_table`
- `examples/ex_table_detail`
- `examples/mac_board_list`
- `examples/mac_board_members`
- `examples/mac_board_members_quotes`
- `examples/mac_symbol_belong_board`
- `examples/mac_symbol_bars`
- `examples/unified_watchlist`

</details>

<a id="viewer"></a>
## Web Viewer

仓库内置了一个轻量的网页查看器，适合在这些场景下使用：

- 先确认某个方法应该填哪些参数。
- 快速查看返回字段，而不是先写一段测试代码。
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

<a id="api"></a>
## API 速览

### 高阶统一入口

- `StockQuotesDetail`、`StockQuotesList`、`StockQuotes`、`StockKLine`、`StockKLineOffset`、`StockVolumeProfile` 会在可获取到流通股本时尽力补齐 `Turnover`。
- 主行情：`StockCount`, `StockList`, `StockQuotesDetail`, `StockKLine`, `StockTickChart`, `StockIndexInfo`, `StockIndexMomentum`, `StockChartSampling`, `StockAuction`, `StockTopBoard`, `StockUnusual`, `StockVolumeProfile`, `StockHistoryOrders`, `StockHistoryTransaction`, `StockF10`
- 扩展市场：`ExCount`, `ExList`, `ExQuote`, `ExQuotes`, `ExKLine`, `ExTickChart`, `ExHistoryTransaction`, `ExTable`
- MAC：`MACBoardList`, `MACBoardMembers`, `MACBoardMembersQuotes`, `MACSymbolBelongBoard`, `MACSymbolBars`

### 常用底层接口

- 主行情：`GetSecurityCount`, `GetSecurityListRange`, `GetQuotesDetail`, `GetIndexInfo`, `GetIndexMomentum`, `GetVolumeProfile`, `GetMinuteTimeData`, `GetAuction`, `GetTopBoard`, `GetUnusual`, `GetTransactionData`, `GetHistoryOrders`
- F10/文件：`GetCompanyCategories`, `GetCompanyContent`, `GetFinanceInfo`, `GetXDXRInfo`, `DownloadFile`, `GetBlockFile`
- 扩展市场：`ExGetCategoryList`, `ExGetQuotesList`, `ExGetQuote`, `ExGetChartSampling`, `ExGetFileMeta`, `ExDownloadFile`

### 地址池与测速

- 内置列表：`MainHosts`, `BrokerHosts`, `ExHosts`, `MACHosts`, `MACExHosts`
- 地址便捷函数：`MainHostAddresses`, `BrokerHostAddresses`, `ExHostAddresses`, `MACHostAddresses`, `MACExHostAddresses`
- 测速函数：`ProbeHosts`, `ProbeAddresses`, `FastestHost`, `FastestAddress`
- 客户端能力：`Client.ProbeHosts`, `Client.FastestHost`, `WithAutoSelectFastest`

### 试验与兼容协议

- 主站试验：`MainTodoB`, `MainTodoFDE`, `MainClient264B`, `MainClient26AC`, `MainClient26AD`, `MainClient26AE`, `MainClient26B1`
- 主站兼容：`StockListOld`, `StockFeature452`, `StockQuotesEncrypt`, `StockKLineOffset`, `StockHistoryTransactionWithTrans`
- 扩展试验：`ExListExtra`, `ExExperiment2487`, `ExExperiment2488`, `ExKLine2`, `ExMapping2562`

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
- `GetMACBoardCount`, `GetMACBoardList`, `GetMACBoardMembers`, `GetMACBoardMembersQuotes`
- `GetMACSymbolBelongBoard`, `GetMACSymbolBars`
- `MACBoardCount`, `MACBoardList`, `MACBoardMembers`, `MACBoardMembersQuotes`
- `MACSymbolBelongBoard`, `MACSymbolBars`

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

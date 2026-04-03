<h1 align="center">🐍 gotdx</h1>

<p align="center">
  📈 通达信行情协议的 Go 客户端。<br>
  覆盖主行情、扩展市场、F10 / 文件接口、板块文件解析，以及一个可直接运行的 Web Viewer。
</p>

<p align="center">
  <a href="#quickstart">🚀 快速开始</a> ·
  <a href="#examples">🧪 示例</a> ·
  <a href="#viewer">🖥️ Web Viewer</a> ·
  <a href="#api-surface">📚 API 范围</a> ·
  <a href="TdxProtocol.md">📄 协议文档</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.26+">
  <img src="https://img.shields.io/badge/Markets-Main%20%2F%20Extended-123B67" alt="Markets">
  <img src="https://img.shields.io/badge/Examples-30%2B-1D1F23" alt="Examples">
  <img src="https://img.shields.io/badge/Web%20Viewer-Built--in-C86B36" alt="Web Viewer">
</p>



gotdx 把通达信协议里常用的查询能力整理成一个偏 Go 风格的包：你可以直接做主行情和扩展市场的实时查询，也可以往下钻到 F10、财务、板块、文件下载和表格接口；如果只是想先把协议跑起来，仓库还自带了一个能直接浏览方法和参数的网页查看器。

## ✨ 为什么用 gotdx

- 一个 `Client` 同时管理主行情与扩展市场，适合做组合监控和跨市场抓取。
- 提供地址池与超时配置，适合对接不稳定的真实行情站点。
- 既有 `Stock* / Ex*` 高阶统一入口，也保留底层原始接口，方便封装和排查协议细节。
- 仓库内置 30+ 可直接运行的示例，覆盖从证券列表、K 线、分时到 F10、板块、扩展市场表格。
- 附带 `cmd/webviewer`，可以不写代码先看协议返回什么。

<a id="quickstart"></a>
## 🚀 快速开始

安装：

```bash
go get github.com/bensema/gotdx
```

最小可运行示例：

```go
package main

import (
	"log"

	"github.com/bensema/gotdx"
)

func main() {
	client := gotdx.New(
		gotdx.WithTCPAddress("124.71.187.122:7709"),
		gotdx.WithTCPAddressPool(
			"124.71.187.72:7709",
			"124.70.133.119:7709",
		),
		gotdx.WithExTCPAddress("112.74.214.43:7727"),
		gotdx.WithExTCPAddressPool(
			"120.25.218.6:7727",
			"43.139.173.246:7727",
		),
		gotdx.WithTimeoutSec(6),
	)
	defer client.Disconnect()

	stocks, err := client.StockQuotesDetail(
		[]uint8{gotdx.MarketSZ, gotdx.MarketSH},
		[]string{"000001", "600519"},
	)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range stocks {
		log.Printf("stock: %+v", item)
	}

	exQuotes, err := client.ExQuotes(
		[]uint8{gotdx.ExCategoryUSStock},
		[]string{"TSLA"},
	)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range exQuotes {
		log.Printf("ex: %+v", item)
	}
}
```

## 🧭 能力分层

| 能力层 | 你会用它做什么 | 代表接口 |
| --- | --- | --- |
| 连接与容错 | 指定主站、扩展站、地址池、超时和重试 | `New`, `WithTCPAddressPool`, `WithExTCPAddressPool`, `WithTimeoutSec` |
| 主行情 | A 股/指数列表、快照、K 线、分时、逐笔、异动 | `StockQuotesDetail`, `GetSecurityList`, `GetSecurityBars`, `GetMinuteTimeData`, `GetTransactionData` |
| 扩展市场 | 美股/港股/期货等扩展市场列表、报价、K 线、表格 | `ExQuotes`, `ExQuotes2`, `ExGetKLine`, `ExGetTable`, `ExGetTableDetail` |
| 文件与 F10 | 公司资料、财务、除权除息、文件下载 | `GetCompanyInfo`, `GetFinanceInfo`, `GetXDXRInfo`, `DownloadFullFile` |
| 板块与聚合 | 板块文件、分组板块、统一高阶入口 | `GetBlockFile`, `GetParsedBlockFile`, `GetGroupedBlockFile`, `ExBoardList` |
| 浏览与调试 | 直接在浏览器里调方法、填参数、看表格结果 | `go run ./cmd/webviewer` |

<a id="examples"></a>
## 🧪 示例

下面这些示例都可以直接运行：

| 场景 | 命令 |
| --- | --- |
| 主行情快照 | `go run ./examples/stock_quotes` |
| 批量主行情 | `go run ./examples/stock_batch_quotes` |
| K 线与指数工具 | `go run ./examples/stock_kline` / `go run ./examples/stock_index_tools` |
| 分时、历史分时、逐笔 | `go run ./examples/stock_tick` / `go run ./examples/stock_history` / `go run ./examples/stock_transaction` |
| F10、公司资料、板块文件 | `go run ./examples/stock_f10_block` / `go run ./examples/stock_company_raw` / `go run ./examples/stock_block_raw` |
| 扩展市场报价 | `go run ./examples/ex_quote` / `go run ./examples/ex_quotes` / `go run ./examples/ex_quotes2` |
| 扩展市场列表与分类 | `go run ./examples/ex_list` / `go run ./examples/ex_paged_list` / `go run ./examples/ex_category_list` |
| 扩展市场 K 线、分时、历史成交 | `go run ./examples/ex_kline` / `go run ./examples/ex_tick` / `go run ./examples/ex_history` |
| 扩展市场表格 | `go run ./examples/ex_table` / `go run ./examples/ex_table_detail` |
| 统一监控示例 | `go run ./examples/unified_watchlist` |

<details>
<summary>查看完整示例目录</summary>

- `examples/stock_count` 市场证券数量
- `examples/stock_list` 股票列表
- `examples/stock_paged_list` 股票列表分页遍历
- `examples/stock_batch_quotes` 批量快照行情
- `examples/stock_quotes` 主行情报价
- `examples/stock_lowlevel_quote` 直连主行情低层报价接口
- `examples/stock_quotes_list` 排序行情
- `examples/stock_kline` K 线
- `examples/stock_index_tools` 指数和抽样图接口
- `examples/stock_tick` 分时
- `examples/stock_history` 历史分时和历史成交
- `examples/stock_market_watch` 集合竞价、异动、成交分布
- `examples/stock_transaction` 当日逐笔成交
- `examples/stock_f10_block` F10 和板块文件
- `examples/stock_company_raw` 公司/F10 原始接口
- `examples/stock_block_raw` 板块文件原始接口
- `examples/ex_count` 扩展市场数量
- `examples/ex_quote` 单个扩展市场报价
- `examples/ex_list` 扩展市场列表
- `examples/ex_paged_list` 扩展市场列表分页遍历
- `examples/ex_quotes` 扩展市场报价
- `examples/ex_quotes2` 扩展市场批量行情兼容接口
- `examples/ex_quotes_list` 扩展市场排序行情
- `examples/ex_kline` 扩展市场 K 线
- `examples/ex_history` 扩展市场历史成交
- `examples/ex_tick` 扩展市场分时
- `examples/ex_server_info` 扩展市场连接和服务信息
- `examples/ex_sampling` 扩展市场抽样图
- `examples/ex_category_list` 扩展市场分类列表
- `examples/ex_table` 扩展市场表格
- `examples/ex_table_detail` 扩展市场详细表格
- `examples/unified_watchlist` 统一 Client 组合监控示例

</details>

<a id="viewer"></a>
## 🖥️ Web Viewer

仓库内置了一个轻量 viewer，可以直接浏览 method、填写参数并以表格查看结果，适合快速验证协议字段和接口行为。

![gotdx Web Viewer 截图](docs/images/webviewer-screenshot.png)

运行：

```bash
go run ./cmd/webviewer
```

默认地址：

```text
http://127.0.0.1:8080
```

<a id="api-surface"></a>
## 📚 API 范围

<details>
<summary>查看完整 API 清单</summary>

### 主行情与通用接口

- `Connect` 连接券商行情服务器
- `Disconnect` 断开服务器
- `GetSecurityCount` 获取指定市场内的证券数目
- `GetSecurityQuotes` 获取盘口五档报价
- `GetQuotesDetail` 获取详细行情报价
- `GetSecurityList` 获取市场内指定范围内的所有证券代码
- `GetSecurityListRange` 获取市场内指定范围内的证券代码
- `GetKLine` 获取 K 线
- `GetSecurityBars` 获取股票 K 线
- `GetIndexBars` 获取指数 K 线
- `GetIndexMomentum` 获取指数动量
- `GetIndexInfo` 获取指数概况
- `GetMinuteTimeData` 获取分时图数据
- `GetTickChart` 获取当日分时图数据
- `GetHistoryMinuteTimeData` 获取历史分时图数据
- `GetHistoryTickChart` 获取历史分时图数据
- `GetChartSampling` 获取抽样图数据
- `GetAuction` 获取集合竞价
- `GetTopBoard` 获取排行榜
- `GetUnusual` 获取主力监控
- `GetTransactionData` 获取分时成交
- `GetHistoryOrders` 获取历史委托
- `GetHistoryTransactionData` 获取历史分时成交

### F10、公司资料与文件

- `GetCompanyCategories` 获取公司信息分类
- `GetCompanyContent` 获取公司信息内容
- `GetFinanceInfo` 获取财务信息
- `GetXDXRInfo` 获取除权除息信息
- `GetCompanyInfo` 获取公司信息聚合结果
- `GetFileMeta` 获取文件元信息
- `DownloadFile` 下载文件片段
- `DownloadFullFile` 下载完整文件
- `GetBlockFile` 获取完整板块文件
- `GetTableFile` 获取表格文件
- `GetCSVFile` 获取 CSV 文件
- `GetParsedBlockFile` 获取解析后的板块文件
- `GetGroupedBlockFile` 获取分组板块文件

### 扩展市场

- `ConnectEx` 连接扩展市场服务器并完成登录
- `GetExServerInfo` 获取扩展市场服务信息
- `ExGetCount` 获取扩展市场标的数量
- `ExGetCategoryList` 获取扩展市场分类列表
- `ExGetList` 获取扩展市场标的列表
- `ExGetQuotesList` 获取扩展市场行情列表
- `ExGetQuote` 获取单个扩展市场行情
- `ExGetQuotes` 获取批量扩展市场行情
- `ExGetQuotes2` 获取批量扩展市场行情兼容接口
- `ExGetKLine` 获取扩展市场 K 线
- `ExGetHistoryTransaction` 获取扩展市场历史成交
- `ExGetTickChart` 获取扩展市场当日分时图
- `ExGetHistoryTickChart` 获取扩展市场历史分时图
- `ExGetChartSampling` 获取扩展市场抽样图
- `ExGetBoardList` 获取扩展市场板块榜单
- `ExGetFileMeta` 获取扩展市场文件元信息
- `ExDownloadFile` 下载扩展市场文件片段
- `ExDownloadFullFile` 下载完整扩展市场文件
- `ExGetTable` 获取扩展市场表格
- `ExGetTableDetail` 获取扩展市场详细表格

### 统一高阶入口

- `ExQuotes2` 统一 Client 扩展市场批量行情兼容入口
- `ExBoardList` 统一 Client 扩展市场板块榜单入口
- `Stock* / Ex*` 统一 Client 高阶入口

</details>

## ✅ 测试

单元测试：

```bash
go test ./...
```

集成测试默认跳过；如需连接真实站点：

```bash
GOTDX_INTEGRATION=1 go test ./...
```

## 🔗 相关文档

- [TdxProtocol.md](TdxProtocol.md): 协议分析笔记
- [docs/images/webviewer-screenshot.png](docs/images/webviewer-screenshot.png): Web Viewer 截图

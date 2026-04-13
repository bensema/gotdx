package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/proto"
)

type methodParam struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Default     string `json:"default"`
	Placeholder string `json:"placeholder,omitempty"`
	Help        string `json:"help,omitempty"`
}

type methodDef struct {
	Key         string        `json:"key"`
	Label       string        `json:"label"`
	Group       string        `json:"group"`
	Target      string        `json:"target"`
	Description string        `json:"description"`
	Params      []methodParam `json:"params"`
}

type queryRequest struct {
	Method string            `json:"method"`
	Params map[string]string `json:"params"`
}

type queryResponse struct {
	Method        string         `json:"method"`
	Label         string         `json:"label"`
	Group         string         `json:"group"`
	Target        string         `json:"target"`
	Description   string         `json:"description"`
	Request       map[string]any `json:"request"`
	Columns       []string       `json:"columns"`
	Rows          [][]string     `json:"rows"`
	TotalRows     int            `json:"total_rows"`
	DisplayedRows int            `json:"displayed_rows"`
	DurationMS    int64          `json:"duration_ms"`
	Warning       string         `json:"warning,omitempty"`
	Raw           any            `json:"raw,omitempty"`
}

type queryPayload struct {
	columns []string
	rows    [][]string
	raw     any
	warning string
}

var methodDefs = []methodDef{
	{
		Key:         "stock_count",
		Label:       "股票数量",
		Group:       "股票快照",
		Target:      "main",
		Description: "获取指定市场证券总数。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0", Help: "0=SZ 1=SH 2=BJ"},
		},
	},
	{
		Key:         "stock_quotes",
		Label:       "股票批量行情",
		Group:       "股票快照",
		Target:      "main",
		Description: "批量盘口行情。",
		Params: []methodParam{
			{Key: "markets", Label: "市场列表", Type: "text", Default: "0,1", Help: "可只填一个市场，多个代码时会自动复用"},
			{Key: "codes", Label: "代码列表", Type: "textarea", Default: "000001,600000", Help: "逗号或换行分隔证券代码"},
		},
	},
	{
		Key:         "stock_quotes_detail",
		Label:       "股票详细行情",
		Group:       "股票快照",
		Target:      "main",
		Description: "多代码详细行情快照。",
		Params: []methodParam{
			{Key: "markets", Label: "市场列表", Type: "text", Default: "0,1", Help: "逗号分隔，0=SZ 1=SH 2=BJ"},
			{Key: "codes", Label: "代码列表", Type: "textarea", Default: "000001,600000", Help: "逗号或换行分隔证券代码"},
		},
	},
	{
		Key:         "stock_quotes_list",
		Label:       "股票排序行情",
		Group:       "股票快照",
		Target:      "main",
		Description: "按分类和排序方式拉取主行情列表。",
		Params: []methodParam{
			{Key: "category", Label: "分类", Type: "number", Default: "6", Help: "6=A股 8=科创板 14=创业板"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "30"},
			{Key: "sort_type", Label: "排序类型", Type: "number", Default: "0"},
			{Key: "reverse", Label: "是否倒序", Type: "text", Default: "false"},
			{Key: "filter", Label: "筛选值", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "stock_list_range",
		Label:       "股票列表分页",
		Group:       "股票快照",
		Target:      "main",
		Description: "按页读取证券代码列表。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "200"},
		},
	},
	{
		Key:         "stock_list_old",
		Label:       "旧版股票列表",
		Group:       "股票快照",
		Target:      "main",
		Description: "兼容旧协议 0x0450 的证券列表。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "stock_feature_452",
		Label:       "证券扩展信息",
		Group:       "股票快照",
		Target:      "main",
		Description: "主行情试验协议 0x0452。",
		Params: []methodParam{
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "30"},
		},
	},
	{
		Key:         "stock_kline",
		Label:       "股票 K 线",
		Group:       "股票快照",
		Target:      "main",
		Description: "获取主行情 K 线。",
		Params: []methodParam{
			{Key: "category", Label: "K线类型", Type: "number", Default: "4"},
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "20"},
			{Key: "times", Label: "倍数", Type: "number", Default: "1"},
			{Key: "adjust", Label: "复权", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "stock_kline_offset",
		Label:       "偏移 K 线",
		Group:       "股票快照",
		Target:      "main",
		Description: "主行情增强协议 0x052d 的偏移 K 线。",
		Params: []methodParam{
			{Key: "category", Label: "K线类型", Type: "number", Default: "4"},
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "20"},
			{Key: "times", Label: "倍数", Type: "number", Default: "1"},
			{Key: "adjust", Label: "复权", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "stock_quotes_encrypt",
		Label:       "加密行情",
		Group:       "股票快照",
		Target:      "main",
		Description: "主行情增强协议 0x0547 的加密行情。",
		Params: []methodParam{
			{Key: "markets", Label: "市场列表", Type: "text", Default: "1,0", Help: "可只填一个市场，多个代码时会自动复用"},
			{Key: "codes", Label: "代码列表", Type: "textarea", Default: "999999,399001", Help: "逗号或换行分隔证券代码"},
		},
	},
	{
		Key:         "stock_tick_chart",
		Label:       "股票当日分时",
		Group:       "股票分时",
		Target:      "main",
		Description: "当日分时图数据。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "20"},
		},
	},
	{
		Key:         "stock_history_tick_chart",
		Label:       "股票历史分时",
		Group:       "股票分时",
		Target:      "main",
		Description: "历史分时图数据。",
		Params: []methodParam{
			{Key: "date", Label: "日期", Type: "number", Default: "20260316"},
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
		},
	},
	{
		Key:         "stock_transaction",
		Label:       "股票逐笔成交",
		Group:       "股票分时",
		Target:      "main",
		Description: "当日逐笔成交。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "20"},
		},
	},
	{
		Key:         "stock_history_transaction",
		Label:       "股票历史成交",
		Group:       "股票分时",
		Target:      "main",
		Description: "历史成交回放。",
		Params: []methodParam{
			{Key: "date", Label: "日期", Type: "number", Default: "20260316"},
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "20"},
		},
	},
	{
		Key:         "stock_history_transaction_trans",
		Label:       "历史成交带方向",
		Group:       "股票分时",
		Target:      "main",
		Description: "主行情增强协议 0x0fc6，返回 BUY/SELL/NEUTRAL。",
		Params: []methodParam{
			{Key: "date", Label: "日期", Type: "number", Default: "20260316"},
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "20"},
		},
	},
	{
		Key:         "stock_history_orders",
		Label:       "股票历史委托",
		Group:       "股票分时",
		Target:      "main",
		Description: "历史委托分布。",
		Params: []methodParam{
			{Key: "date", Label: "日期", Type: "number", Default: "20260316"},
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
		},
	},
	{
		Key:         "stock_index_info",
		Label:       "指数概况",
		Group:       "股票指数",
		Target:      "main",
		Description: "指数概况、动量和日线摘要。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "399001"},
		},
	},
	{
		Key:         "stock_chart_sampling",
		Label:       "抽样图",
		Group:       "股票指数",
		Target:      "main",
		Description: "抽样图价格序列。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
		},
	},
	{
		Key:         "stock_auction",
		Label:       "集合竞价",
		Group:       "股票监控",
		Target:      "main",
		Description: "集合竞价明细。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "20"},
		},
	},
	{
		Key:         "stock_top_board",
		Label:       "排行九宫格",
		Group:       "股票监控",
		Target:      "main",
		Description: "排行榜九宫格榜单。",
		Params: []methodParam{
			{Key: "category", Label: "分类", Type: "number", Default: "6"},
			{Key: "size", Label: "数量", Type: "number", Default: "5"},
		},
	},
	{
		Key:         "stock_unusual",
		Label:       "异动监控",
		Group:       "股票监控",
		Target:      "main",
		Description: "异动监控列表。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "20"},
		},
	},
	{
		Key:         "stock_volume_profile",
		Label:       "成交分布",
		Group:       "股票监控",
		Target:      "main",
		Description: "成交分布。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
		},
	},
	{
		Key:         "stock_company_info",
		Label:       "公司信息聚合",
		Group:       "股票资料",
		Target:      "main",
		Description: "F10 聚合信息、财务和除权除息。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
		},
	},
	{
		Key:         "stock_company_categories",
		Label:       "公司信息目录",
		Group:       "股票资料",
		Target:      "main",
		Description: "F10 分类目录。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
		},
	},
	{
		Key:         "stock_company_content",
		Label:       "公司信息正文",
		Group:       "股票资料",
		Target:      "main",
		Description: "F10 原始内容。未填写 filename 时自动读取第一条分类。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
			{Key: "filename", Label: "文件名", Type: "text", Default: "", Help: "留空则自动选第一条分类"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "length", Label: "长度", Type: "number", Default: "1024"},
		},
	},
	{
		Key:         "stock_finance",
		Label:       "财务信息",
		Group:       "股票资料",
		Target:      "main",
		Description: "财务信息。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
		},
	},
	{
		Key:         "stock_xdxr",
		Label:       "除权除息",
		Group:       "股票资料",
		Target:      "main",
		Description: "除权除息信息。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
		},
	},
	{
		Key:         "stock_block_grouped",
		Label:       "板块文件分组",
		Group:       "股票资料",
		Target:      "main",
		Description: "按板块分组展示 block 文件。",
		Params: []methodParam{
			{Key: "filename", Label: "文件名", Type: "text", Default: "block_fg.dat"},
		},
	},
	{
		Key:         "ex_count",
		Label:       "扩展市场数量",
		Group:       "扩展快照",
		Target:      "ex",
		Description: "扩展市场标的总数。",
	},
	{
		Key:         "ex_category_list",
		Label:       "扩展分类列表",
		Group:       "扩展快照",
		Target:      "ex",
		Description: "扩展市场分类列表。",
	},
	{
		Key:         "ex_list",
		Label:       "扩展标的列表",
		Group:       "扩展快照",
		Target:      "ex",
		Description: "扩展市场标的分页列表。",
		Params: []methodParam{
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "30"},
		},
	},
	{
		Key:         "ex_list_extra",
		Label:       "扩展试验列表",
		Group:       "扩展试验",
		Target:      "ex",
		Description: "扩展市场试验协议 0x23f6。",
		Params: []methodParam{
			{Key: "a", Label: "参数 A", Type: "number", Default: "0"},
			{Key: "b", Label: "参数 B", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "30"},
		},
	},
	{
		Key:         "ex_quote",
		Label:       "扩展单条行情",
		Group:       "扩展快照",
		Target:      "ex",
		Description: "单个扩展市场行情。",
		Params: []methodParam{
			{Key: "category", Label: "分类", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
		},
	},
	{
		Key:         "ex_quotes",
		Label:       "扩展批量行情",
		Group:       "扩展快照",
		Target:      "ex",
		Description: "扩展市场批量行情。",
		Params: []methodParam{
			{Key: "categories", Label: "分类列表", Type: "text", Default: "74,71", Help: "可只填一个分类，多个代码时会自动复用"},
			{Key: "codes", Label: "代码列表", Type: "textarea", Default: "TSLA,00700", Help: "逗号或换行分隔代码"},
		},
	},
	{
		Key:         "ex_quotes2",
		Label:       "扩展批量行情2",
		Group:       "扩展快照",
		Target:      "ex",
		Description: "扩展市场第二种批量行情接口。",
		Params: []methodParam{
			{Key: "categories", Label: "分类列表", Type: "text", Default: "74,71", Help: "可只填一个分类，多个代码时会自动复用"},
			{Key: "codes", Label: "代码列表", Type: "textarea", Default: "TSLA,00700", Help: "逗号或换行分隔代码"},
		},
	},
	{
		Key:         "ex_quotes_list",
		Label:       "扩展排序行情",
		Group:       "扩展快照",
		Target:      "ex",
		Description: "扩展市场排序行情。",
		Params: []methodParam{
			{Key: "category", Label: "分类", Type: "number", Default: "74"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "30"},
			{Key: "sort_type", Label: "排序类型", Type: "number", Default: "0"},
			{Key: "reverse", Label: "是否倒序", Type: "text", Default: "false"},
		},
	},
	{
		Key:         "ex_kline",
		Label:       "扩展 K 线",
		Group:       "扩展快照",
		Target:      "ex",
		Description: "扩展市场 K 线。",
		Params: []methodParam{
			{Key: "category", Label: "分类", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
			{Key: "period", Label: "周期", Type: "number", Default: "4"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "20"},
			{Key: "times", Label: "倍数", Type: "number", Default: "1"},
		},
	},
	{
		Key:         "ex_experiment_2487",
		Label:       "扩展试验报价 2487",
		Group:       "扩展试验",
		Target:      "ex",
		Description: "扩展市场试验协议 0x2487。",
		Params: []methodParam{
			{Key: "category", Label: "分类", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
		},
	},
	{
		Key:         "ex_experiment_2488",
		Label:       "扩展试验报价 2488",
		Group:       "扩展试验",
		Target:      "ex",
		Description: "扩展市场试验协议 0x2488。",
		Params: []methodParam{
			{Key: "category", Label: "分类", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
			{Key: "mode", Label: "模式", Type: "number", Default: "55"},
		},
	},
	{
		Key:         "ex_kline2",
		Label:       "扩展 K 线 2",
		Group:       "扩展试验",
		Target:      "ex",
		Description: "扩展市场 K 线协议 0x2489。",
		Params: []methodParam{
			{Key: "category", Label: "分类", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
			{Key: "period", Label: "周期", Type: "number", Default: "4"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "20"},
			{Key: "times", Label: "倍数", Type: "number", Default: "1"},
		},
	},
	{
		Key:         "ex_tick_chart",
		Label:       "扩展分时图",
		Group:       "扩展分时",
		Target:      "ex",
		Description: "扩展市场当日或历史分时图。",
		Params: []methodParam{
			{Key: "category", Label: "分类", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
			{Key: "date", Label: "日期", Type: "number", Default: "0", Help: "填 0 取当日，填如 20260330 取历史"},
		},
	},
	{
		Key:         "ex_history_transaction",
		Label:       "扩展历史成交",
		Group:       "扩展分时",
		Target:      "ex",
		Description: "扩展市场历史成交。",
		Params: []methodParam{
			{Key: "date", Label: "日期", Type: "number", Default: "20260330"},
			{Key: "category", Label: "分类", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
		},
	},
	{
		Key:         "ex_chart_sampling",
		Label:       "扩展抽样图",
		Group:       "扩展分时",
		Target:      "ex",
		Description: "扩展市场抽样图。",
		Params: []methodParam{
			{Key: "category", Label: "分类", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
		},
	},
	{
		Key:         "ex_board_list",
		Label:       "扩展榜单",
		Group:       "扩展分时",
		Target:      "ex",
		Description: "扩展市场板块榜单，部分主机可能较慢。",
		Params: []methodParam{
			{Key: "board_type", Label: "榜单类型", Type: "number", Default: "0"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "page_size", Label: "页大小", Type: "number", Default: "20"},
		},
	},
	{
		Key:         "ex_mapping_2562",
		Label:       "扩展映射 2562",
		Group:       "扩展试验",
		Target:      "ex",
		Description: "扩展市场映射试验协议 0x2562。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "47"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "30"},
		},
	},
	{
		Key:         "ex_table",
		Label:       "扩展总表",
		Group:       "扩展表格",
		Target:      "ex",
		Description: "扩展市场总表，自动拆成表格行。",
	},
	{
		Key:         "ex_table_detail",
		Label:       "扩展详细表",
		Group:       "扩展表格",
		Target:      "ex",
		Description: "扩展市场详细表，自动拆成表格行。",
	},
	{
		Key:         "mac_board_list",
		Label:       "MAC 板块列表",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 主站板块分页列表。",
		Params: []methodParam{
			{Key: "board_type", Label: "板块类型", Type: "number", Default: "0"},
			{Key: "count", Label: "总量", Type: "number", Default: "50"},
		},
	},
	{
		Key:         "mac_board_members",
		Label:       "MAC 板块成员",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "按板块代码查询成分股。",
		Params: []methodParam{
			{Key: "board_symbol", Label: "板块代码", Type: "text", Default: "880761"},
			{Key: "count", Label: "总量", Type: "number", Default: "50"},
		},
	},
	{
		Key:         "mac_board_members_quotes",
		Label:       "MAC 成分报价",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "按板块代码查询成分报价。",
		Params: []methodParam{
			{Key: "board_symbol", Label: "板块代码", Type: "text", Default: "880761"},
			{Key: "count", Label: "总量", Type: "number", Default: "50"},
		},
	},
	{
		Key:         "mac_symbol_belong_board",
		Label:       "股票所属板块",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "查询单只股票所属板块。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000100"},
		},
	},
	{
		Key:         "mac_symbol_bars",
		Label:       "MAC 统一 K 线",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 主站统一 K 线。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000100"},
			{Key: "period", Label: "周期", Type: "number", Default: "4"},
			{Key: "times", Label: "倍数", Type: "number", Default: "1"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "总量", Type: "number", Default: "20"},
			{Key: "adjust", Label: "复权", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "mac_ex_board_list",
		Label:       "MAC 港美板块",
		Group:       "MAC 协议",
		Target:      "mac-ex",
		Description: "MAC 扩展站板块列表，例如港股/美股板块。",
		Params: []methodParam{
			{Key: "board_type", Label: "板块类型", Type: "number", Default: "0", Help: "0=HK_ALL 3=US_ALL"},
			{Key: "count", Label: "总量", Type: "number", Default: "50"},
		},
	},
	{
		Key:         "mac_ex_symbol_bars",
		Label:       "MAC 扩展 K 线",
		Group:       "MAC 协议",
		Target:      "mac-ex",
		Description: "MAC 扩展站统一 K 线。",
		Params: []methodParam{
			{Key: "market", Label: "分类", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
			{Key: "period", Label: "周期", Type: "number", Default: "4"},
			{Key: "times", Label: "倍数", Type: "number", Default: "1"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "总量", Type: "number", Default: "20"},
			{Key: "adjust", Label: "复权", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "main_connect_info",
		Label:       "主站连接信息",
		Group:       "连接状态",
		Target:      "main",
		Description: "连接主行情服务器并显示欢迎信息。",
	},
	{
		Key:         "main_heartbeat",
		Label:       "主站心跳",
		Group:       "连接状态",
		Target:      "main",
		Description: "主站服务心跳包。",
	},
	{
		Key:         "main_server_info",
		Label:       "主站服务信息",
		Group:       "连接状态",
		Target:      "main",
		Description: "主站服务信息 0x0015。",
	},
	{
		Key:         "main_exchange_announcement",
		Label:       "交易所公告",
		Group:       "连接状态",
		Target:      "main",
		Description: "主站交易所公告 0x0002。",
	},
	{
		Key:         "main_announcement",
		Label:       "服务商公告",
		Group:       "连接状态",
		Target:      "main",
		Description: "主站服务商公告 0x000a。",
	},
	{
		Key:         "main_todo_b",
		Label:       "主站试验 000b",
		Group:       "主站试验",
		Target:      "main",
		Description: "主站试验协议 0x000b，返回原始字节。",
	},
	{
		Key:         "main_todo_fde",
		Label:       "主站试验 0fde",
		Group:       "主站试验",
		Target:      "main",
		Description: "主站试验协议 0x0fde，返回原始字节。",
	},
	{
		Key:         "main_client_264b",
		Label:       "客户端信息 264b",
		Group:       "主站试验",
		Target:      "main",
		Description: "客户端信息协议 0x264b，返回原始字节。",
	},
	{
		Key:         "main_client_26ac",
		Label:       "客户端信息 26ac",
		Group:       "主站试验",
		Target:      "main",
		Description: "客户端信息协议 0x26ac，返回原始字节。",
	},
	{
		Key:         "main_client_26ad",
		Label:       "客户端信息 26ad",
		Group:       "主站试验",
		Target:      "main",
		Description: "客户端信息协议 0x26ad，返回原始字节。",
	},
	{
		Key:         "main_client_26ae",
		Label:       "客户端信息 26ae",
		Group:       "主站试验",
		Target:      "main",
		Description: "客户端信息协议 0x26ae，返回原始字节。",
	},
	{
		Key:         "main_client_26b1",
		Label:       "客户端信息 26b1",
		Group:       "主站试验",
		Target:      "main",
		Description: "客户端信息协议 0x26b1，返回原始字节。",
	},
	{
		Key:         "ex_server_info",
		Label:       "扩展站连接信息",
		Group:       "连接状态",
		Target:      "ex",
		Description: "扩展市场登录信息和服务器信息。",
	},
	{
		Key:         "mac_connect_info",
		Label:       "MAC 主站连接",
		Group:       "连接状态",
		Target:      "mac",
		Description: "连接 MAC 主站并显示当前主机。",
	},
	{
		Key:         "mac_ex_connect_info",
		Label:       "MAC 扩展站连接",
		Group:       "连接状态",
		Target:      "mac-ex",
		Description: "连接 MAC 扩展站并显示当前主机。",
	},
	{
		Key:         "main_host_probe",
		Label:       "主站测速",
		Group:       "连接状态",
		Target:      "main",
		Description: "对内置主行情 host 列表做 TCP 测速并排序。",
		Params: []methodParam{
			{Key: "timeout_ms", Label: "超时毫秒", Type: "number", Default: "1000"},
		},
	},
	{
		Key:         "ex_host_probe",
		Label:       "扩展站测速",
		Group:       "连接状态",
		Target:      "ex",
		Description: "对内置扩展市场 host 列表做 TCP 测速并排序。",
		Params: []methodParam{
			{Key: "timeout_ms", Label: "超时毫秒", Type: "number", Default: "1000"},
		},
	},
	{
		Key:         "mac_host_probe",
		Label:       "MAC 主站测速",
		Group:       "连接状态",
		Target:      "mac",
		Description: "对内置 MAC 主站 host 列表做 TCP 测速并排序。",
		Params: []methodParam{
			{Key: "timeout_ms", Label: "超时毫秒", Type: "number", Default: "1000"},
		},
	},
	{
		Key:         "mac_ex_host_probe",
		Label:       "MAC 扩展测速",
		Group:       "连接状态",
		Target:      "mac-ex",
		Description: "对内置 MAC 扩展站 host 列表做 TCP 测速并排序。",
		Params: []methodParam{
			{Key: "timeout_ms", Label: "超时毫秒", Type: "number", Default: "1000"},
		},
	},
	{
		Key:         "broker_host_list",
		Label:       "券商地址列表",
		Group:       "连接状态",
		Target:      "main",
		Description: "显示内置的券商行情 host 列表。",
	},
}

var methodMap = makeMethodMap(methodDefs)

var mainHosts = gotdx.MainHostAddresses()

var exHosts = gotdx.ExHostAddresses()

var macHosts = gotdx.MACHostAddresses()

var macExHosts = gotdx.MACExHostAddresses()

func makeMethodMap(defs []methodDef) map[string]methodDef {
	out := make(map[string]methodDef, len(defs))
	for _, def := range defs {
		out[def.Key] = def
	}
	return out
}

func executeQuery(req queryRequest) (*queryResponse, error) {
	def, ok := methodMap[req.Method]
	if !ok {
		return nil, fmt.Errorf("未知方法: %s", req.Method)
	}
	if req.Params == nil {
		req.Params = map[string]string{}
	}

	started := time.Now()
	payload, requestView, err := runMethod(def, req.Params)
	if err != nil {
		return nil, err
	}

	rows := payload.rows
	totalRows := len(rows)
	rows = limitRows(rows, 1000)

	return &queryResponse{
		Method:        def.Key,
		Label:         def.Label,
		Group:         def.Group,
		Target:        def.Target,
		Description:   def.Description,
		Request:       requestView,
		Columns:       payload.columns,
		Rows:          rows,
		TotalRows:     totalRows,
		DisplayedRows: len(rows),
		DurationMS:    time.Since(started).Milliseconds(),
		Warning:       payload.warning,
		Raw:           payload.raw,
	}, nil
}

func runMethod(def methodDef, params map[string]string) (queryPayload, map[string]any, error) {
	switch def.Key {
	case "main_connect_info":
		client := newMainClient()
		defer client.Disconnect()
		reply, err := client.Connect()
		if err != nil {
			return queryPayload{}, nil, err
		}
		rows := [][]string{
			{"info", reply.Info},
			{"host", currentMainHost(client)},
		}
		return queryPayload{
			columns: []string{"field", "value"},
			rows:    rows,
			raw:     reply,
		}, map[string]any{}, nil
	case "main_heartbeat":
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetServerHeartbeat()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    [][]string{{"date", fmt.Sprintf("%d", reply.Date)}},
				raw:     reply,
			}, nil
		})
		return payload, map[string]any{}, err
	case "main_server_info":
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetServerInfo()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromServerInfo(reply),
				raw:     reply,
			}, nil
		})
		return payload, map[string]any{}, err
	case "main_exchange_announcement":
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetExchangeAnnouncement()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    [][]string{{"version", fmt.Sprintf("%d", reply.Version)}, {"content", reply.Content}},
				raw:     reply,
			}, nil
		})
		return payload, map[string]any{}, err
	case "main_announcement":
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetAnnouncement()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromAnnouncement(reply),
				raw:     reply,
			}, nil
		})
		return payload, map[string]any{}, err
	case "main_todo_b":
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetTodoB()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromRawReply(reply),
				raw:     reply,
				warning: "原始实验协议，仅展示长度和十六进制预览。",
			}, nil
		})
		return payload, map[string]any{}, err
	case "main_todo_fde":
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetTodoFDE()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromRawReply(reply),
				raw:     reply,
				warning: "原始实验协议，仅展示长度和十六进制预览。",
			}, nil
		})
		return payload, map[string]any{}, err
	case "main_client_264b":
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetClient264B()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromRawReply(reply),
				raw:     reply,
				warning: "客户端信息协议暂以原始响应方式展示。",
			}, nil
		})
		return payload, map[string]any{}, err
	case "main_client_26ac":
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetClient26AC()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromRawReply(reply),
				raw:     reply,
				warning: "客户端信息协议暂以原始响应方式展示。",
			}, nil
		})
		return payload, map[string]any{}, err
	case "main_client_26ad":
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetClient26AD()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromRawReply(reply),
				raw:     reply,
				warning: "客户端信息协议暂以原始响应方式展示。",
			}, nil
		})
		return payload, map[string]any{}, err
	case "main_client_26ae":
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetClient26AE()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromRawReply(reply),
				raw:     reply,
				warning: "客户端信息协议暂以原始响应方式展示。",
			}, nil
		})
		return payload, map[string]any{}, err
	case "main_client_26b1":
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetClient26B1()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromRawReply(reply),
				raw:     reply,
				warning: "客户端信息协议暂以原始响应方式展示。",
			}, nil
		})
		return payload, map[string]any{}, err
	case "mac_connect_info":
		client := newMACClient()
		defer client.Disconnect()
		if err := client.ConnectMAC(); err != nil {
			return queryPayload{}, nil, err
		}
		return queryPayload{
			columns: []string{"field", "value"},
			rows:    [][]string{{"host", currentMACHost(client)}},
			raw:     map[string]any{"host": currentMACHost(client)},
		}, map[string]any{}, nil
	case "mac_ex_connect_info":
		client := newMACExClient()
		defer client.Disconnect()
		if err := client.ConnectMAC(); err != nil {
			return queryPayload{}, nil, err
		}
		return queryPayload{
			columns: []string{"field", "value"},
			rows:    [][]string{{"host", currentMACExHost(client)}},
			raw:     map[string]any{"host": currentMACExHost(client)},
		}, map[string]any{}, nil
	case "main_host_probe":
		timeoutMS, err := parseUint32Value(params, "timeout_ms", 1000)
		if err != nil {
			return queryPayload{}, nil, err
		}
		results := gotdx.ProbeHosts(gotdx.MainHosts(), time.Duration(timeoutMS)*time.Millisecond)
		return queryPayload{
			columns: []string{"name", "address", "latency_ms", "reachable", "error"},
			rows:    rowsFromHostProbeResults(results),
			raw:     results,
		}, map[string]any{"timeout_ms": timeoutMS}, nil
	case "ex_host_probe":
		timeoutMS, err := parseUint32Value(params, "timeout_ms", 1000)
		if err != nil {
			return queryPayload{}, nil, err
		}
		results := gotdx.ProbeHosts(gotdx.ExHosts(), time.Duration(timeoutMS)*time.Millisecond)
		return queryPayload{
			columns: []string{"name", "address", "latency_ms", "reachable", "error"},
			rows:    rowsFromHostProbeResults(results),
			raw:     results,
		}, map[string]any{"timeout_ms": timeoutMS}, nil
	case "mac_host_probe":
		timeoutMS, err := parseUint32Value(params, "timeout_ms", 1000)
		if err != nil {
			return queryPayload{}, nil, err
		}
		results := gotdx.ProbeHosts(gotdx.MACHosts(), time.Duration(timeoutMS)*time.Millisecond)
		return queryPayload{
			columns: []string{"name", "address", "latency_ms", "reachable", "error"},
			rows:    rowsFromHostProbeResults(results),
			raw:     results,
		}, map[string]any{"timeout_ms": timeoutMS}, nil
	case "mac_ex_host_probe":
		timeoutMS, err := parseUint32Value(params, "timeout_ms", 1000)
		if err != nil {
			return queryPayload{}, nil, err
		}
		results := gotdx.ProbeHosts(gotdx.MACExHosts(), time.Duration(timeoutMS)*time.Millisecond)
		return queryPayload{
			columns: []string{"name", "address", "latency_ms", "reachable", "error"},
			rows:    rowsFromHostProbeResults(results),
			raw:     results,
		}, map[string]any{"timeout_ms": timeoutMS}, nil
	case "broker_host_list":
		hosts := gotdx.BrokerHosts()
		return queryPayload{
			columns: []string{"name", "ip", "port", "address"},
			rows:    rowsFromHostInfos(hosts),
			raw:     hosts,
		}, map[string]any{}, nil
	case "stock_count":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetSecurityCount(market)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    [][]string{{"market", fmt.Sprintf("%d", market)}, {"count", fmt.Sprintf("%d", reply.Count)}},
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_quotes":
		markets, err := parseUint8List(valueOrDefault(params, "markets", "0,1"))
		if err != nil {
			return queryPayload{}, nil, err
		}
		codes := parseCodeList(valueOrDefault(params, "codes", "000001,600000"))
		markets, err = expandUint8List(markets, len(codes), "markets")
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"markets": markets, "codes": codes}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.StockQuotes(markets, codes)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "code", "price", "pre_close", "change", "vol", "amount", "rise_speed", "turnover"},
				rows:    rowsFromQuoteList(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_quotes_detail":
		markets, err := parseUint8List(valueOrDefault(params, "markets", "0,1"))
		if err != nil {
			return queryPayload{}, nil, err
		}
		codes := parseCodeList(valueOrDefault(params, "codes", "000001,600000"))
		markets, err = expandUint8List(markets, len(codes), "markets")
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"markets": markets, "codes": codes}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.StockQuotesDetail(markets, codes)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "code", "time", "price", "open", "high", "low", "vol", "amount", "turnover"},
				rows:    rowsFromQuoteDetail(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_quotes_list":
		category, err := parseUint8Value(params, "category", 6)
		if err != nil {
			return queryPayload{}, nil, err
		}
		start, err := parseUint16Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint16Value(params, "count", 30)
		if err != nil {
			return queryPayload{}, nil, err
		}
		sortType, err := parseUint16Value(params, "sort_type", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		reverse, err := parseBoolValue(params, "reverse", false)
		if err != nil {
			return queryPayload{}, nil, err
		}
		filter, err := parseUint16Value(params, "filter", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"category": category, "start": start, "count": count, "sort_type": sortType, "reverse": reverse, "filter": filter}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.StockQuotesList(category, start, count, sortType, reverse, filter)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "code", "price", "pre_close", "change", "vol", "amount", "rise_speed", "turnover"},
				rows:    rowsFromQuoteList(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_list_range":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint32Value(params, "count", 200)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "start": start, "count": count}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetSecurityListRange(market, start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"code", "name", "pre_close", "vol_unit", "decimal_point"},
				rows:    rowsFromSecurityList(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "stock_list_old":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		start, err := parseUint16Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "start": start}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetSecurityListOld(market, start)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"code", "name", "pre_close", "vol_unit", "decimal_point"},
				rows:    rowsFromSecurityList(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "stock_feature_452":
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint32Value(params, "count", 30)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"start": start, "count": count}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetSecurityFeature452(start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "code", "p1", "p2"},
				rows:    rowsFromSecurityFeature452(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "stock_kline":
		category, err := parseUint16Value(params, "category", 4)
		if err != nil {
			return queryPayload{}, nil, err
		}
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		start, err := parseUint16Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint16Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		times, err := parseUint16Value(params, "times", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		adjust, err := parseUint16Value(params, "adjust", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"category": category, "market": market, "code": code, "start": start, "count": count, "times": times, "adjust": adjust}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.StockKLine(category, market, code, start, count, times, adjust)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"datetime", "open", "high", "low", "close", "vol", "amount", "turnover"},
				rows:    rowsFromSecurityBars(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_kline_offset":
		category, err := parseUint16Value(params, "category", 4)
		if err != nil {
			return queryPayload{}, nil, err
		}
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		start, err := parseUint16Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint16Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		times, err := parseUint16Value(params, "times", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		adjust, err := parseUint16Value(params, "adjust", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"category": category, "market": market, "code": code, "start": start, "count": count, "times": times, "adjust": adjust}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.StockKLineOffset(category, market, code, start, count, times, adjust)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"datetime", "open", "high", "low", "close", "vol", "amount", "turnover"},
				rows:    rowsFromSecurityBars(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_quotes_encrypt":
		markets, err := parseUint8List(valueOrDefault(params, "markets", "1,0"))
		if err != nil {
			return queryPayload{}, nil, err
		}
		codes := parseCodeList(valueOrDefault(params, "codes", "999999,399001"))
		markets, err = expandUint8List(markets, len(codes), "markets")
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"markets": markets, "codes": codes}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetQuotesEncrypt(markets, codes)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "code", "time", "close", "pre_close", "open", "high", "low", "vol", "amount"},
				rows:    rowsFromEncryptedQuotes(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "stock_tick_chart":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		start, err := parseUint16Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint16Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "code": code, "start": start, "count": count}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetTickChart(market, code, start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"index", "price", "avg", "vol"},
				rows:    rowsFromMinuteTimeData(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "stock_history_tick_chart":
		date, err := parseUint32Value(params, "date", 20260316)
		if err != nil {
			return queryPayload{}, nil, err
		}
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		request := map[string]any{"date": date, "market": market, "code": code}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetHistoryTickChart(date, market, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"index", "price", "avg", "vol"},
				rows:    rowsFromHistoryMinuteTimeData(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "stock_transaction":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		start, err := parseUint16Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint16Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "code": code, "start": start, "count": count}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetTransactionData(market, code, start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"time", "price", "vol", "num", "action"},
				rows:    rowsFromTransaction(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "stock_history_transaction":
		date, err := parseUint32Value(params, "date", 20260316)
		if err != nil {
			return queryPayload{}, nil, err
		}
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		start, err := parseUint16Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint16Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"date": date, "market": market, "code": code, "start": start, "count": count}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetHistoryTransactionData(date, market, code, start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"time", "price", "vol", "num", "action"},
				rows:    rowsFromHistoryTransaction(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "stock_history_transaction_trans":
		date, err := parseUint32Value(params, "date", 20260316)
		if err != nil {
			return queryPayload{}, nil, err
		}
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		start, err := parseUint16Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint16Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"date": date, "market": market, "code": code, "start": start, "count": count}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetHistoryTransactionDataWithTrans(date, market, code, start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"time", "price", "vol", "num", "action"},
				rows:    rowsFromHistoryTransactionWithTrans(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "stock_history_orders":
		date, err := parseUint32Value(params, "date", 20260316)
		if err != nil {
			return queryPayload{}, nil, err
		}
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		request := map[string]any{"date": date, "market": market, "code": code}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetHistoryOrders(date, market, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"price", "unknown", "vol"},
				rows:    rowsFromHistoryOrders(reply.List),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_index_info":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "399001")
		request := map[string]any{"market": market, "code": code}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			info, err := client.GetIndexInfo(market, code)
			if err != nil {
				return queryPayload{}, err
			}
			momentum, err := client.GetIndexMomentum(market, code)
			if err != nil {
				return queryPayload{}, err
			}
			bars, err := client.GetIndexBars(gotdx.KLINE_TYPE_DAILY, market, code, 0, 5)
			if err != nil {
				return queryPayload{}, err
			}
			rows := [][]string{
				{"summary", info.Code, info.ServerTime, formatFloat(info.Close), formatFloat(info.Open), formatFloat(info.High), formatFloat(info.Low), fmt.Sprintf("%d", info.UpCount), fmt.Sprintf("%d", info.DownCount)},
				{"momentum", "-", "-", "-", "-", "-", "-", fmt.Sprintf("%d", momentum.Count), fmt.Sprintf("%d", lastInt(momentum.Values))},
			}
			for _, bar := range bars.List {
				rows = append(rows, []string{"bar", code, bar.DateTime, formatFloat(bar.Close), formatFloat(bar.Open), formatFloat(bar.High), formatFloat(bar.Low), "", ""})
			}
			return queryPayload{
				columns: []string{"type", "code", "time", "close", "open", "high", "low", "metric_a", "metric_b"},
				rows:    rows,
				raw: map[string]any{
					"info":     info,
					"momentum": momentum,
					"bars":     bars.List,
				},
			}, nil
		})
		return payload, request, err
	case "stock_chart_sampling":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		request := map[string]any{"market": market, "code": code}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetChartSampling(market, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"index", "price", "pre_close", "change"},
				rows:    rowsFromSampling(reply.PreClose, reply.Prices),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_auction":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint32Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "code": code, "start": start, "count": count}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetAuction(market, code, start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"time", "price", "matched", "unmatched"},
				rows:    rowsFromAuction(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "stock_top_board":
		category, err := parseUint8Value(params, "category", 6)
		if err != nil {
			return queryPayload{}, nil, err
		}
		size, err := parseUint8Value(params, "size", 5)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"category": category, "size": size}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetTopBoard(category, size)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"list", "market", "code", "price", "value"},
				rows:    rowsFromTopBoard(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_unusual":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint32Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "start": start, "count": count}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetUnusual(market, start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"index", "market", "code", "time", "desc", "value", "unusual_type"},
				rows:    rowsFromUnusual(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "stock_volume_profile":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		request := map[string]any{"market": market, "code": code}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.StockVolumeProfile(market, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"price", "vol", "buy", "sell", "turnover"},
				rows:    rowsFromVolumeProfile(reply.VolProfiles, reply.Turnover),
				raw:     reply,
				warning: "部分主站返回的价格档位仍可能存在异常跳变，适合作为协议调试观察。",
			}, nil
		})
		return payload, request, err
	case "stock_company_info":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		request := map[string]any{"market": market, "code": code}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetCompanyInfo(market, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"section", "preview"},
				rows:    rowsFromCompanyBundle(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_company_categories":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		request := map[string]any{"market": market, "code": code}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetCompanyCategories(market, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"name", "filename", "start", "length"},
				rows:    rowsFromCompanyCategories(reply.Categories),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_company_content":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		filename := strings.TrimSpace(valueOrDefault(params, "filename", ""))
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		length, err := parseUint32Value(params, "length", 1024)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "code": code, "filename": filename, "start": start, "length": length}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			actualFilename := filename
			actualStart := start
			actualLength := length
			autoPick := map[string]any{}
			if actualFilename == "" {
				categories, err := client.GetCompanyCategories(market, code)
				if err != nil {
					return queryPayload{}, err
				}
				if len(categories.Categories) == 0 {
					return queryPayload{}, fmt.Errorf("no company categories for %s", code)
				}
				first := categories.Categories[0]
				actualFilename = first.Filename
				actualStart = first.Start
				actualLength = first.Length
				autoPick = map[string]any{
					"name":     first.Name,
					"filename": first.Filename,
					"start":    first.Start,
					"length":   first.Length,
				}
			}
			reply, err := client.GetCompanyContent(market, code, actualFilename, actualStart, actualLength)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows: [][]string{
					{"market", fmt.Sprintf("%d", market)},
					{"code", code},
					{"filename", actualFilename},
					{"start", fmt.Sprintf("%d", actualStart)},
					{"length", fmt.Sprintf("%d", actualLength)},
					{"content", preview(reply.Content, 2000)},
				},
				raw: map[string]any{
					"auto_pick": autoPick,
					"reply":     reply,
				},
			}, nil
		})
		return payload, request, err
	case "stock_finance":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		request := map[string]any{"market": market, "code": code}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetFinanceInfo(market, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromFinanceInfo(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_xdxr":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		request := map[string]any{"market": market, "code": code}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetXDXRInfo(market, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"date", "category", "name", "fenhong", "peigujia", "songzhuangu", "peigu", "suogu", "xingquanjia"},
				rows:    rowsFromXDXR(reply.List),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_block_grouped":
		filename := valueOrDefault(params, "filename", gotdx.BlockFileFG)
		request := map[string]any{"filename": filename}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetGroupedBlockFile(filename)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"block_name", "block_type", "stock_count", "sample_codes"},
				rows:    rowsFromBlockGroups(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "ex_count":
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetCount()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    [][]string{{"count", fmt.Sprintf("%d", reply.Count)}},
				raw:     reply,
			}, nil
		})
		return payload, map[string]any{}, err
	case "ex_category_list":
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetCategoryList()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "code", "name", "abbr"},
				rows:    rowsFromExCategoryList(reply.List),
				raw:     reply,
			}, nil
		})
		return payload, map[string]any{}, err
	case "ex_list":
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint16Value(params, "count", 30)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"start": start, "count": count}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetList(start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "category", "code", "name", "desc_1", "desc_2", "desc_3"},
				rows:    rowsFromExList(reply.List),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "ex_list_extra":
		a, err := parseUint16Value(params, "a", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		b, err := parseUint16Value(params, "b", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint16Value(params, "count", 30)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"a": a, "b": b, "count": count}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetListExtra(a, b, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"category", "code", "flag", "values"},
				rows:    rowsFromExListExtra(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "ex_server_info":
		client := newExClient()
		defer client.Disconnect()
		login, err := client.ConnectEx()
		if err != nil {
			return queryPayload{}, nil, err
		}
		info, err := client.GetExServerInfo()
		if err != nil {
			return queryPayload{}, nil, err
		}
		rows := [][]string{
			{"host", currentExHost(client)},
			{"login_time", login.DateTime},
			{"login_server", login.ServerName},
			{"login_ip", login.IP},
			{"server_name", info.ServerName},
			{"version", info.Version},
			{"delay", fmt.Sprintf("%d", info.Delay)},
			{"time_now", info.TimeNow},
			{"info", info.Info},
		}
		return queryPayload{
			columns: []string{"field", "value"},
			rows:    rows,
			raw:     map[string]any{"login": login, "info": info},
		}, map[string]any{}, nil
	case "ex_quote":
		category, err := parseUint8Value(params, "category", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		request := map[string]any{"category": category, "code": code}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetQuote(category, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"code", "date", "close", "open", "high", "low", "vol", "avg"},
				rows:    rowsFromExQuotes([]proto.ExQuoteItem{reply.Item}),
				raw:     reply.Item,
			}, nil
		})
		return payload, request, err
	case "ex_quotes":
		categories, err := parseUint8List(valueOrDefault(params, "categories", "74,71"))
		if err != nil {
			return queryPayload{}, nil, err
		}
		codes := parseCodeList(valueOrDefault(params, "codes", "TSLA,00700"))
		categories, err = expandUint8List(categories, len(codes), "categories")
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"categories": categories, "codes": codes}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetQuotes(categories, codes)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"code", "date", "close", "open", "high", "low", "vol", "avg"},
				rows:    rowsFromExQuotes(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "ex_quotes2":
		categories, err := parseUint8List(valueOrDefault(params, "categories", "74,71"))
		if err != nil {
			return queryPayload{}, nil, err
		}
		codes := parseCodeList(valueOrDefault(params, "codes", "TSLA,00700"))
		categories, err = expandUint8List(categories, len(codes), "categories")
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"categories": categories, "codes": codes}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetQuotes2(categories, codes)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"code", "date", "close", "open", "high", "low", "vol", "avg"},
				rows:    rowsFromExQuotes(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "ex_quotes_list":
		category, err := parseUint8Value(params, "category", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		start, err := parseUint16Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint16Value(params, "count", 30)
		if err != nil {
			return queryPayload{}, nil, err
		}
		sortType, err := parseUint16Value(params, "sort_type", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		reverse, err := parseBoolValue(params, "reverse", false)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"category": category, "start": start, "count": count, "sort_type": sortType, "reverse": reverse}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetQuotesList(category, start, count, sortType, reverse)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"code", "date", "close", "open", "high", "low", "vol", "avg"},
				rows:    rowsFromExQuotes(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "ex_kline":
		category, err := parseUint8Value(params, "category", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		period, err := parseUint16Value(params, "period", 4)
		if err != nil {
			return queryPayload{}, nil, err
		}
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint16Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		times, err := parseUint16Value(params, "times", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"category": category, "code": code, "period": period, "start": start, "count": count, "times": times}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetKLine(category, code, period, start, count, times)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"datetime", "open", "high", "low", "close", "vol", "amount"},
				rows:    rowsFromExKLine(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "ex_experiment_2487":
		category, err := parseUint8Value(params, "category", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		request := map[string]any{"category": category, "code": code}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetExperiment2487(category, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromExExperiment2487(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "ex_experiment_2488":
		category, err := parseUint8Value(params, "category", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		mode, err := parseUint16Value(params, "mode", 55)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"category": category, "code": code, "mode": mode}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetExperiment2488(category, code, mode)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"id", "values"},
				rows:    rowsFromExExperiment2488(reply.List),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "ex_kline2":
		category, err := parseUint8Value(params, "category", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		period, err := parseUint16Value(params, "period", 4)
		if err != nil {
			return queryPayload{}, nil, err
		}
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint32Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		times, err := parseUint16Value(params, "times", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"category": category, "code": code, "period": period, "start": start, "count": count, "times": times}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetKLine2(category, code, period, start, count, times)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"datetime", "open", "high", "low", "close", "vol", "amount"},
				rows:    rowsFromExKLine(reply.List),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "ex_tick_chart":
		category, err := parseUint8Value(params, "category", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		date, err := parseUint32Value(params, "date", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"category": category, "code": code, "date": date}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			var rows [][]string
			var raw any
			if date == 0 {
				reply, err := client.ExGetTickChart(category, code)
				if err != nil {
					return queryPayload{}, err
				}
				rows = rowsFromExTick(reply.List)
				raw = reply.List
			} else {
				reply, err := client.ExGetHistoryTickChart(date, category, code)
				if err != nil {
					return queryPayload{}, err
				}
				rows = rowsFromExTick(reply.List)
				raw = reply.List
			}
			return queryPayload{
				columns: []string{"time", "price", "avg", "vol"},
				rows:    rows,
				raw:     raw,
			}, nil
		})
		return payload, request, err
	case "ex_history_transaction":
		date, err := parseUint32Value(params, "date", 20260330)
		if err != nil {
			return queryPayload{}, nil, err
		}
		category, err := parseUint8Value(params, "category", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		request := map[string]any{"date": date, "category": category, "code": code}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetHistoryTransaction(date, category, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"time", "price", "vol", "action"},
				rows:    rowsFromExHistoryTransaction(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "ex_chart_sampling":
		category, err := parseUint8Value(params, "category", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		request := map[string]any{"category": category, "code": code}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetChartSampling(category, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"index", "price", "pre_close", "change"},
				rows:    rowsFromSampling(0, reply.Prices),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "ex_board_list":
		boardType, err := parseUint16Value(params, "board_type", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		start, err := parseUint16Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		pageSize, err := parseUint16Value(params, "page_size", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"board_type": boardType, "start": start, "page_size": pageSize}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetBoardList(boardType, start, pageSize)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "code", "name", "price", "rise_speed", "symbol_code", "symbol_name", "symbol_price"},
				rows:    rowsFromExBoardList(reply.List),
				raw:     reply,
				warning: "部分扩展主机的 board_list 响应较慢，超时通常是服务端行为。",
			}, nil
		})
		return payload, request, err
	case "ex_mapping_2562":
		market, err := parseUint16Value(params, "market", 47)
		if err != nil {
			return queryPayload{}, nil, err
		}
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint32Value(params, "count", 30)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "start": start, "count": count}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetMapping2562(market, start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"category", "name", "unknown", "index", "switch", "codes"},
				rows:    rowsFromExMapping2562(reply.List),
				raw:     reply.List,
			}, nil
		})
		return payload, request, err
	case "ex_table":
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			content, err := client.ExGetTable()
			if err != nil {
				return queryPayload{}, err
			}
			rows := parseExTableRows(content)
			return queryPayload{
				columns: []string{"key", "category", "code", "name"},
				rows:    rows,
				raw:     rawTextPreview(content),
			}, nil
		})
		return payload, map[string]any{}, err
	case "ex_table_detail":
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			content, err := client.ExGetTableDetail()
			if err != nil {
				return queryPayload{}, err
			}
			columns, rows := parseExTableDetailRows(content)
			return queryPayload{
				columns: columns,
				rows:    rows,
				raw:     rawTextPreview(content),
			}, nil
		})
		return payload, map[string]any{}, err
	case "mac_board_list":
		boardType, err := parseUint16Value(params, "board_type", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint32Value(params, "count", 50)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"board_type": boardType, "count": count}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			list, err := client.MACBoardList(boardType, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "code", "name", "price", "rise_speed", "symbol_code", "symbol_name", "symbol_price"},
				rows:    rowsFromMACBoardList(list),
				raw:     list,
			}, nil
		})
		return payload, request, err
	case "mac_ex_board_list":
		boardType, err := parseUint16Value(params, "board_type", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint32Value(params, "count", 50)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"board_type": boardType, "count": count}
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			list, err := client.MACBoardList(boardType, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "code", "name", "price", "rise_speed", "symbol_code", "symbol_name", "symbol_price"},
				rows:    rowsFromMACBoardList(list),
				raw:     list,
			}, nil
		})
		return payload, request, err
	case "mac_board_members":
		boardSymbol := valueOrDefault(params, "board_symbol", "880761")
		count, err := parseUint32Value(params, "count", 50)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"board_symbol": boardSymbol, "count": count}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			list, err := client.MACBoardMembers(boardSymbol, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "symbol", "name"},
				rows:    rowsFromMACBoardMembers(list),
				raw:     list,
			}, nil
		})
		return payload, request, err
	case "mac_board_members_quotes":
		boardSymbol := valueOrDefault(params, "board_symbol", "880761")
		count, err := parseUint32Value(params, "count", 50)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"board_symbol": boardSymbol, "count": count}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			list, err := client.MACBoardMembersQuotes(boardSymbol, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "symbol", "name", "close", "pre_close", "rise_speed", "turnover_rate", "pe_ttm"},
				rows:    rowsFromMACBoardMemberQuotes(list),
				raw:     list,
			}, nil
		})
		return payload, request, err
	case "mac_symbol_belong_board":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000100")
		request := map[string]any{"market": market, "code": code}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			list, err := client.MACSymbolBelongBoard(code, market)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"board_type", "status_code", "board_code", "board_name", "price", "pre_close", "metric1", "metric2", "metric3"},
				rows:    rowsFromMACBelongBoards(list),
				raw:     list,
			}, nil
		})
		return payload, request, err
	case "mac_symbol_bars":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000100")
		period, err := parseUint16Value(params, "period", 4)
		if err != nil {
			return queryPayload{}, nil, err
		}
		times, err := parseUint16Value(params, "times", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint32Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		adjust, err := parseUint16Value(params, "adjust", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "code": code, "period": period, "times": times, "start": start, "count": count, "adjust": adjust}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			list, err := client.MACSymbolBars(market, code, period, times, start, count, adjust)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"datetime", "open", "high", "low", "close", "vol", "amount", "float_shares"},
				rows:    rowsFromMACSymbolBars(list),
				raw:     list,
			}, nil
		})
		return payload, request, err
	case "mac_ex_symbol_bars":
		market, err := parseUint8Value(params, "market", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		period, err := parseUint16Value(params, "period", 4)
		if err != nil {
			return queryPayload{}, nil, err
		}
		times, err := parseUint16Value(params, "times", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint32Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		adjust, err := parseUint16Value(params, "adjust", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "code": code, "period": period, "times": times, "start": start, "count": count, "adjust": adjust}
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			list, err := client.MACSymbolBars(market, code, period, times, start, count, adjust)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"datetime", "open", "high", "low", "close", "vol", "amount", "float_shares"},
				rows:    rowsFromMACSymbolBars(list),
				raw:     list,
			}, nil
		})
		return payload, request, err
	default:
		return queryPayload{}, nil, fmt.Errorf("暂不支持的方法: %s", def.Key)
	}
}

func newMainClient() *gotdx.Client {
	return gotdx.New(
		gotdx.WithTCPAddress(mainHosts[0]),
		gotdx.WithTCPAddressPool(mainHosts[1:]...),
		gotdx.WithTimeoutSec(6),
	)
}

func newExClient() *gotdx.Client {
	return gotdx.NewEx(
		gotdx.WithExTCPAddress(exHosts[0]),
		gotdx.WithExTCPAddressPool(exHosts[1:]...),
		gotdx.WithTimeoutSec(6),
	)
}

func newMACClient() *gotdx.Client {
	return gotdx.NewMAC(
		gotdx.WithMacTCPAddress(macHosts[0]),
		gotdx.WithMacTCPAddressPool(macHosts[1:]...),
		gotdx.WithTimeoutSec(6),
	)
}

func newMACExClient() *gotdx.Client {
	return gotdx.NewMACEx(
		gotdx.WithMacExTCPAddress(macExHosts[0]),
		gotdx.WithMacExTCPAddressPool(macExHosts[1:]...),
		gotdx.WithTimeoutSec(6),
	)
}

func withMainClient(fn func(*gotdx.Client) (queryPayload, error)) (queryPayload, error) {
	client := newMainClient()
	defer client.Disconnect()
	if _, err := client.Connect(); err != nil {
		return queryPayload{}, err
	}
	return fn(client)
}

func withExClient(fn func(*gotdx.Client) (queryPayload, error)) (queryPayload, error) {
	client := newExClient()
	defer client.Disconnect()
	if _, err := client.ConnectEx(); err != nil {
		return queryPayload{}, err
	}
	return fn(client)
}

func withMACClient(fn func(*gotdx.Client) (queryPayload, error)) (queryPayload, error) {
	client := newMACClient()
	defer client.Disconnect()
	if err := client.ConnectMAC(); err != nil {
		return queryPayload{}, err
	}
	return fn(client)
}

func withMACExClient(fn func(*gotdx.Client) (queryPayload, error)) (queryPayload, error) {
	client := newMACExClient()
	defer client.Disconnect()
	if err := client.ConnectMAC(); err != nil {
		return queryPayload{}, err
	}
	return fn(client)
}

func currentMainHost(client *gotdx.Client) string {
	if client == nil {
		return ""
	}
	return client.CurrentAddress()
}

func currentExHost(client *gotdx.Client) string {
	if client == nil {
		return ""
	}
	return client.CurrentAddress()
}

func currentMACHost(client *gotdx.Client) string {
	if client == nil {
		return ""
	}
	return client.CurrentAddress()
}

func currentMACExHost(client *gotdx.Client) string {
	if client == nil {
		return ""
	}
	return client.CurrentAddress()
}

func rowsFromHostInfos(items []gotdx.HostInfo) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Name,
			item.IP,
			strconv.Itoa(item.Port),
			item.Address(),
		})
	}
	return rows
}

func rowsFromHostProbeResults(items []gotdx.HostProbeResult) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		latencyMS := ""
		if item.Reachable {
			latencyMS = strconv.FormatInt(item.Latency.Milliseconds(), 10)
		}
		rows = append(rows, []string{
			item.Name,
			item.Address,
			latencyMS,
			strconv.FormatBool(item.Reachable),
			item.Error,
		})
	}
	return rows
}

func parseCodeList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseUint8List(value string) ([]uint8, error) {
	parts := parseCodeList(value)
	out := make([]uint8, 0, len(parts))
	for _, part := range parts {
		v, err := strconv.ParseUint(part, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid uint8 list value %q", part)
		}
		out = append(out, uint8(v))
	}
	return out, nil
}

func expandUint8List(values []uint8, targetLen int, label string) ([]uint8, error) {
	if targetLen == 0 {
		return values, nil
	}
	if len(values) == targetLen {
		return values, nil
	}
	if len(values) == 1 && targetLen > 1 {
		out := make([]uint8, targetLen)
		for i := range out {
			out[i] = values[0]
		}
		return out, nil
	}
	return nil, fmt.Errorf("%s 数量必须和代码数量一致", label)
}

func parseUint8Value(params map[string]string, key string, def uint8) (uint8, error) {
	value := valueOrDefault(params, key, strconv.FormatUint(uint64(def), 10))
	v, err := strconv.ParseUint(strings.TrimSpace(value), 10, 8)
	if err != nil {
		return 0, fmt.Errorf("%s 必须是 uint8", key)
	}
	return uint8(v), nil
}

func parseUint16Value(params map[string]string, key string, def uint16) (uint16, error) {
	value := valueOrDefault(params, key, strconv.FormatUint(uint64(def), 10))
	v, err := strconv.ParseUint(strings.TrimSpace(value), 10, 16)
	if err != nil {
		return 0, fmt.Errorf("%s 必须是 uint16", key)
	}
	return uint16(v), nil
}

func parseUint32Value(params map[string]string, key string, def uint32) (uint32, error) {
	value := valueOrDefault(params, key, strconv.FormatUint(uint64(def), 10))
	v, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s 必须是 uint32", key)
	}
	return uint32(v), nil
}

func parseBoolValue(params map[string]string, key string, def bool) (bool, error) {
	value := strings.TrimSpace(valueOrDefault(params, key, strconv.FormatBool(def)))
	switch strings.ToLower(value) {
	case "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off", "":
		return false, nil
	default:
		return false, fmt.Errorf("%s 必须是布尔值", key)
	}
}

func valueOrDefault(params map[string]string, key string, def string) string {
	value, ok := params[key]
	if !ok || strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

func limitRows(rows [][]string, max int) [][]string {
	if len(rows) <= max {
		return rows
	}
	return rows[:max]
}

func rawTextPreview(content string) map[string]any {
	const maxBytes = 4000
	if len(content) <= maxBytes {
		return map[string]any{
			"preview":   content,
			"length":    len(content),
			"truncated": false,
		}
	}
	return map[string]any{
		"preview":   content[:maxBytes],
		"length":    len(content),
		"truncated": true,
	}
}

func rowsFromRawReply(reply *proto.RawDataReply) [][]string {
	if reply == nil {
		return nil
	}
	return [][]string{
		{"length", fmt.Sprintf("%d", reply.Length)},
		{"hex_preview", preview(reply.Hex, 512)},
	}
}

func rowsFromQuoteDetail(items []proto.SecurityQuote) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Market),
			item.Code,
			item.ServerTime,
			formatFloat(item.Price),
			formatFloat(item.Open),
			formatFloat(item.High),
			formatFloat(item.Low),
			fmt.Sprintf("%d", item.Vol),
			formatFloat(item.Amount),
			formatFloat(item.Turnover),
		})
	}
	return rows
}

func rowsFromQuoteList(items []proto.QuoteListItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Market),
			item.Code,
			formatFloat(item.Price),
			formatFloat(item.PreClose),
			formatFloat(item.Price - item.PreClose),
			fmt.Sprintf("%d", item.Vol),
			formatFloat(item.Amount),
			formatFloat(item.RiseSpeed),
			formatFloat(item.Turnover),
		})
	}
	return rows
}

func rowsFromSecurityList(items []proto.Security) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Code,
			item.Name,
			formatFloat(item.PreClose),
			fmt.Sprintf("%d", item.VolUnit),
			fmt.Sprintf("%d", item.DecimalPoint),
		})
	}
	return rows
}

func rowsFromSecurityFeature452(items []proto.SecurityFeature452Item) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Market),
			item.Code,
			formatFloat(item.P1),
			formatFloat(item.P2),
		})
	}
	return rows
}

func rowsFromSecurityBars(items []proto.SecurityBar) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.DateTime,
			formatFloat(item.Open),
			formatFloat(item.High),
			formatFloat(item.Low),
			formatFloat(item.Close),
			formatFloat(item.Vol),
			formatFloat(item.Amount),
			formatFloat(item.Turnover),
		})
	}
	return rows
}

func rowsFromMinuteTimeData(items []proto.MinuteTimeData) [][]string {
	rows := make([][]string, 0, len(items))
	for i, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", i),
			formatFloat(item.Price),
			formatFloat(item.Avg),
			fmt.Sprintf("%d", item.Vol),
		})
	}
	return rows
}

func rowsFromHistoryMinuteTimeData(items []proto.HistoryMinuteTimeData) [][]string {
	rows := make([][]string, 0, len(items))
	for i, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", i),
			formatFloat(item.Price),
			formatFloat(item.Avg),
			fmt.Sprintf("%d", item.Vol),
		})
	}
	return rows
}

func rowsFromTransaction(items []proto.TransactionData) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Time,
			formatFloat(item.Price),
			fmt.Sprintf("%d", item.Vol),
			fmt.Sprintf("%d", item.Num),
			fmt.Sprintf("%d", item.BuyOrSell),
		})
	}
	return rows
}

func rowsFromHistoryTransaction(items []proto.HistoryTransactionData) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Time,
			formatFloat(item.Price),
			fmt.Sprintf("%d", item.Vol),
			fmt.Sprintf("%d", item.Num),
			fmt.Sprintf("%d", item.BuyOrSell),
		})
	}
	return rows
}

func rowsFromHistoryTransactionWithTrans(items []proto.HistoryTransactionDataWithTrans) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Time,
			formatFloat(item.Price),
			fmt.Sprintf("%d", item.Vol),
			fmt.Sprintf("%d", item.Num),
			item.Action,
		})
	}
	return rows
}

func rowsFromEncryptedQuotes(items []proto.EncryptedQuoteItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Market),
			item.Code,
			item.Time,
			formatFloat(item.Close),
			formatFloat(item.PreClose),
			formatFloat(item.Open),
			formatFloat(item.High),
			formatFloat(item.Low),
			fmt.Sprintf("%d", item.Vol),
			formatFloat(item.Amount),
		})
	}
	return rows
}

func rowsFromAnnouncement(item *proto.AnnouncementReply) [][]string {
	if item == nil {
		return nil
	}
	rows := [][]string{{"has_content", strconv.FormatBool(item.HasContent)}}
	if item.HasContent {
		rows = append(rows,
			[]string{"expire_date", item.ExpireDate},
			[]string{"title", item.Title},
			[]string{"author", item.Author},
			[]string{"content", item.Content},
		)
	}
	return rows
}

func rowsFromServerInfo(item *proto.InfoReply) [][]string {
	if item == nil {
		return nil
	}
	return [][]string{
		{"delay", fmt.Sprintf("%d", item.Delay)},
		{"info", item.Info},
		{"content", item.Content},
		{"server_sign", item.ServerSign},
		{"time_now", item.TimeNow},
		{"region", fmt.Sprintf("%d", item.Region)},
		{"switch", fmt.Sprintf("%d", item.MaybeSwitch)},
	}
}

func rowsFromHistoryOrders(items []proto.HistoryOrderData) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			formatFloat(item.Price),
			fmt.Sprintf("%d", item.Unknown),
			fmt.Sprintf("%d", item.Vol),
		})
	}
	return rows
}

func rowsFromAuction(items []proto.AuctionData) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Time,
			formatFloat(item.Price),
			fmt.Sprintf("%d", item.Matched),
			fmt.Sprintf("%d", item.Unmatched),
		})
	}
	return rows
}

func rowsFromTopBoard(reply *proto.GetTopBoardReply) [][]string {
	type namedList struct {
		name  string
		items []proto.TopBoardItem
	}
	lists := []namedList{
		{name: "increase", items: reply.Increase},
		{name: "decrease", items: reply.Decrease},
		{name: "amplitude", items: reply.Amplitude},
		{name: "rise_speed", items: reply.RiseSpeed},
		{name: "fall_speed", items: reply.FallSpeed},
		{name: "vol_ratio", items: reply.VolRatio},
		{name: "pos_commission_ratio", items: reply.PosCommissionRatio},
		{name: "neg_commission_ratio", items: reply.NegCommissionRatio},
		{name: "turnover", items: reply.Turnover},
	}
	rows := make([][]string, 0)
	for _, list := range lists {
		for _, item := range list.items {
			rows = append(rows, []string{
				list.name,
				fmt.Sprintf("%d", item.Market),
				item.Code,
				formatFloat(item.Price),
				formatFloat(item.Value),
			})
		}
	}
	return rows
}

func rowsFromUnusual(items []proto.UnusualData) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Index),
			fmt.Sprintf("%d", item.Market),
			item.Code,
			item.Time,
			item.Desc,
			item.Value,
			fmt.Sprintf("%d", item.UnusualType),
		})
	}
	return rows
}

func rowsFromVolumeProfile(items []proto.VolumeProfileItem, turnover float64) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			formatFloat(item.Price),
			fmt.Sprintf("%d", item.Vol),
			fmt.Sprintf("%d", item.Buy),
			fmt.Sprintf("%d", item.Sell),
			formatFloat(turnover),
		})
	}
	return rows
}

func rowsFromCompanyBundle(bundle *gotdx.CompanyInfoBundle) [][]string {
	rows := make([][]string, 0, len(bundle.Sections)+2)
	for _, section := range bundle.Sections {
		rows = append(rows, []string{section.Name, preview(section.Content, 120)})
	}
	rows = append(rows, []string{"xdxr_count", fmt.Sprintf("%d", len(bundle.XDXR))})
	if bundle.Finance != nil {
		rows = append(rows, []string{"finance", fmt.Sprintf("updated=%d revenue=%.2f net_profit=%.2f", bundle.Finance.UpdatedDate, bundle.Finance.OperatingRevenue, bundle.Finance.NetProfit)})
	}
	return rows
}

func rowsFromCompanyCategories(items []proto.CompanyCategory) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Name,
			item.Filename,
			fmt.Sprintf("%d", item.Start),
			fmt.Sprintf("%d", item.Length),
		})
	}
	return rows
}

func rowsFromFinanceInfo(item *proto.GetFinanceInfoReply) [][]string {
	if item == nil {
		return nil
	}
	return [][]string{
		{"code", item.Code},
		{"updated_date", fmt.Sprintf("%d", item.UpdatedDate)},
		{"ipo_date", fmt.Sprintf("%d", item.IPODate)},
		{"total_shares", fmt.Sprintf("%.2f", item.TotalShares)},
		{"float_shares", fmt.Sprintf("%.2f", item.FloatShares)},
		{"eps", fmt.Sprintf("%.4f", item.EPS)},
		{"total_assets", fmt.Sprintf("%.2f", item.TotalAssets)},
		{"current_assets", fmt.Sprintf("%.2f", item.CurrentAssets)},
		{"current_liabilities", fmt.Sprintf("%.2f", item.CurrentLiabilities)},
		{"total_equity", fmt.Sprintf("%.2f", item.TotalEquity)},
		{"operating_revenue", fmt.Sprintf("%.2f", item.OperatingRevenue)},
		{"operating_profit", fmt.Sprintf("%.2f", item.OperatingProfit)},
		{"total_profit", fmt.Sprintf("%.2f", item.TotalProfit)},
		{"net_profit", fmt.Sprintf("%.2f", item.NetProfit)},
		{"net_assets_per_share", fmt.Sprintf("%.4f", item.NetAssetsPerShare)},
		{"shareholder_count", fmt.Sprintf("%.2f", item.ShareholderCount)},
	}
}

func rowsFromXDXR(items []proto.XDXRItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Date.Format("2006-01-02"),
			fmt.Sprintf("%d", item.Category),
			item.Name,
			formatFloat32Ptr(item.Fenhong),
			formatFloat32Ptr(item.Peigujia),
			formatFloat32Ptr(item.Songzhuangu),
			formatFloat32Ptr(item.Peigu),
			formatFloat32Ptr(item.Suogu),
			formatFloat32Ptr(item.Xingquanjia),
		})
	}
	return rows
}

func rowsFromBlockGroups(groups []gotdx.BlockGroup) [][]string {
	rows := make([][]string, 0, len(groups))
	for _, group := range groups {
		rows = append(rows, []string{
			group.BlockName,
			fmt.Sprintf("%d", group.BlockType),
			fmt.Sprintf("%d", group.StockCount),
			strings.Join(group.Codes[:minInt(5, len(group.Codes))], ","),
		})
	}
	return rows
}

func rowsFromExQuotes(items []proto.ExQuoteItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Code,
			item.Date,
			formatFloat(item.Close),
			formatFloat(item.Open),
			formatFloat(item.High),
			formatFloat(item.Low),
			fmt.Sprintf("%d", item.Vol),
			formatFloat(item.Avg),
		})
	}
	return rows
}

func rowsFromExCategoryList(items []proto.ExCategoryItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Market),
			fmt.Sprintf("%d", item.Code),
			item.Name,
			item.Abbr,
		})
	}
	return rows
}

func rowsFromExList(items []proto.ExListItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		desc1, desc2, desc3 := "", "", ""
		if len(item.Desc) > 0 {
			desc1 = formatFloat(item.Desc[0])
		}
		if len(item.Desc) > 1 {
			desc2 = formatFloat(item.Desc[1])
		}
		if len(item.Desc) > 2 {
			desc3 = formatFloat(item.Desc[2])
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Market),
			fmt.Sprintf("%d", item.Category),
			item.Code,
			item.Name,
			desc1,
			desc2,
			desc3,
		})
	}
	return rows
}

func rowsFromExListExtra(items []proto.ExExtraListItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		values := make([]string, 0, len(item.Values))
		for _, value := range item.Values {
			values = append(values, fmt.Sprintf("%d", value))
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Category),
			item.Code,
			fmt.Sprintf("%d", item.Flag),
			strings.Join(values, ","),
		})
	}
	return rows
}

func rowsFromExKLine(items []proto.ExKLineItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.DateTime,
			formatFloat(item.Open),
			formatFloat(item.High),
			formatFloat(item.Low),
			formatFloat(item.Close),
			fmt.Sprintf("%d", item.Vol),
			formatFloat(item.Amount),
		})
	}
	return rows
}

func rowsFromExExperiment2487(item *proto.ExExperiment2487Reply) [][]string {
	if item == nil {
		return nil
	}
	return [][]string{
		{"category", fmt.Sprintf("%d", item.Category)},
		{"code", item.Code},
		{"active", fmt.Sprintf("%d", item.Active)},
		{"pre_close", formatFloat(item.PreClose)},
		{"open", formatFloat(item.Open)},
		{"high", formatFloat(item.High)},
		{"low", formatFloat(item.Low)},
		{"close", formatFloat(item.Close)},
		{"u1", formatFloat(item.U1)},
		{"price", formatFloat(item.Price)},
		{"vol", fmt.Sprintf("%d", item.Vol)},
		{"cur_vol", fmt.Sprintf("%d", item.CurVol)},
		{"amount", formatFloat(item.Amount)},
		{"tail_hex", preview(item.TailHex, 256)},
	}
}

func rowsFromExExperiment2488(items []proto.ExExperiment2488Item) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		values := make([]string, 0, len(item.Values))
		for _, value := range item.Values {
			values = append(values, fmt.Sprintf("%d", value))
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.ID),
			strings.Join(values, ","),
		})
	}
	return rows
}

func rowsFromExTick(items []proto.ExTickChartData) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Time,
			formatFloat(item.Price),
			formatFloat(item.Avg),
			fmt.Sprintf("%d", item.Vol),
		})
	}
	return rows
}

func rowsFromExHistoryTransaction(items []proto.ExHistoryTransactionItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Time,
			fmt.Sprintf("%d", item.Price),
			fmt.Sprintf("%d", item.Vol),
			item.Action,
		})
	}
	return rows
}

func rowsFromExBoardList(items []proto.ExBoardListItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Market),
			item.Code,
			item.Name,
			formatFloat(item.Price),
			formatFloat(item.RiseSpeed),
			item.SymbolCode,
			item.SymbolName,
			formatFloat(item.SymbolPrice),
		})
	}
	return rows
}

func rowsFromExMapping2562(items []proto.ExMapping2562Item) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		codes := []string{
			formatFloat(item.Code1),
			formatFloat(item.Code2),
			formatFloat(item.Code3),
			fmt.Sprintf("%d", item.Code4),
			fmt.Sprintf("%d", item.Code5),
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Category),
			item.Name,
			fmt.Sprintf("%d", item.Unknown),
			fmt.Sprintf("%d", item.Index),
			fmt.Sprintf("%d", item.Switch),
			strings.Join(codes, ","),
		})
	}
	return rows
}

func rowsFromMACBoardList(items []proto.MACBoardListItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Market),
			item.Code,
			item.Name,
			formatFloat(item.Price),
			formatFloat(item.RiseSpeed),
			item.SymbolCode,
			item.SymbolName,
			formatFloat(item.SymbolPrice),
		})
	}
	return rows
}

func rowsFromMACBoardMembers(items []proto.MACBoardMemberItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Market),
			item.Symbol,
			item.Name,
		})
	}
	return rows
}

func rowsFromMACBoardMemberQuotes(items []proto.MACBoardMemberQuoteItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Market),
			item.Symbol,
			item.Name,
			formatFloat(item.Close),
			formatFloat(item.PreClose),
			formatFloat(item.RiseSpeed),
			formatFloat(item.TurnoverRate),
			formatFloat(item.PETTM),
		})
	}
	return rows
}

func rowsFromMACBelongBoards(items []proto.MACBelongBoardItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.BoardType,
			fmt.Sprintf("%d", item.StatusCode),
			item.BoardCode,
			item.BoardName,
			formatFloat(item.Price),
			formatFloat(item.PreClose),
			formatFloat(item.Metric1),
			formatFloat(item.Metric2),
			formatFloat(item.Metric3),
		})
	}
	return rows
}

func rowsFromMACSymbolBars(items []proto.MACSymbolBar) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.DateTime,
			formatFloat(item.Open),
			formatFloat(item.High),
			formatFloat(item.Low),
			formatFloat(item.Close),
			formatFloat(item.Vol),
			formatFloat(item.Amount),
			formatFloat(item.FloatShares),
		})
	}
	return rows
}

func rowsFromSampling(preClose float64, prices []float64) [][]string {
	rows := make([][]string, 0, len(prices))
	for i, price := range prices {
		change := ""
		if preClose != 0 {
			change = formatFloat(price - preClose)
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i),
			formatFloat(price),
			formatFloat(preClose),
			change,
		})
	}
	return rows
}

func parseExTableRows(content string) [][]string {
	rows := make([][]string, 0)
	for _, entry := range strings.Split(content, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, "|")
		key := parts[0]
		category := ""
		code := key
		if idx := strings.IndexByte(key, '#'); idx >= 0 {
			category = key[:idx]
			code = key[idx+1:]
		}
		name := ""
		if len(parts) > 1 {
			name = parts[1]
		}
		rows = append(rows, []string{key, category, code, name})
	}
	return rows
}

func parseExTableDetailRows(content string) ([]string, [][]string) {
	rows := make([][]string, 0)
	maxCols := 0
	for _, entry := range strings.Split(content, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, "|")
		rows = append(rows, parts)
		if len(parts) > maxCols {
			maxCols = len(parts)
		}
	}
	if maxCols == 0 {
		return []string{"key"}, rows
	}

	columns := make([]string, 0, maxCols)
	for i := 0; i < maxCols; i++ {
		if i == 0 {
			columns = append(columns, "key")
		} else {
			columns = append(columns, fmt.Sprintf("c%d", i+1))
		}
	}

	for i := range rows {
		if len(rows[i]) < maxCols {
			padded := make([]string, maxCols)
			copy(padded, rows[i])
			rows[i] = padded
		}
	}
	return columns, rows
}

func preview(text string, max int) string {
	text = strings.TrimSpace(text)
	if len(text) <= max {
		return text
	}
	return text[:max]
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func formatFloat32Ptr(value *float32) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(float64(*value), 'f', 4, 64)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func lastInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	return values[len(values)-1]
}

func ensureMethodsSorted() {
	sort.Slice(methodDefs, func(i, j int) bool {
		gi := methodGroupRank(methodDefs[i].Group)
		gj := methodGroupRank(methodDefs[j].Group)
		if gi != gj {
			return gi < gj
		}
		if methodDefs[i].Group == methodDefs[j].Group {
			return methodDefs[i].Label < methodDefs[j].Label
		}
		return methodDefs[i].Group < methodDefs[j].Group
	})
}

func methodGroupRank(group string) int {
	switch group {
	case "股票快照":
		return 0
	case "股票分时":
		return 1
	case "股票指数":
		return 2
	case "股票监控":
		return 3
	case "股票资料":
		return 4
	case "主站试验":
		return 5
	case "扩展快照":
		return 10
	case "扩展分时":
		return 11
	case "扩展表格":
		return 12
	case "扩展试验":
		return 13
	case "MAC 协议":
		return 15
	case "连接状态":
		return 20
	default:
		return 100
	}
}

func init() {
	ensureMethodsSorted()
	methodMap = makeMethodMap(methodDefs)
}

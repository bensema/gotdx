package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/proto"
	"github.com/bensema/gotdx/types"
)

// methodParam 描述前端表单中单个查询参数的定义，由前端据此渲染输入控件并回填默认值。
type methodParam struct {
	Key         string `json:"key"`                   // 参数名，与 `/api/query` 请求体 `params` 中的键一一对应
	Label       string `json:"label"`                 // 表单展示名称，仅用于界面标签，不参与协议编码
	Type        string `json:"type"`                  // 输入控件类型：`number` 数值框、`text` 单行文本、`textarea` 多行文本
	Default     string `json:"default"`               // 参数默认值字符串；前端未填写时以该值提交，数值型也以字符串承载
	Placeholder string `json:"placeholder,omitempty"` // 输入框占位提示文本，无占位提示时省略该字段
	Help        string `json:"help,omitempty"`        // 参数补充说明，展示在输入控件旁，无说明时省略该字段
}

// methodDef 描述一个可在网页上调用的 `gotdx` 方法及其分组、目标连接与参数定义。
type methodDef struct {
	Key         string        `json:"key"`         // 方法唯一标识，请求体 `method` 按该键在 `methodMap` 中匹配
	Label       string        `json:"label"`       // 前端展示名称，用于方法列表与响应回显
	Group       string        `json:"group"`       // 分组名称，决定前端排序与展示归类，取值见分组优先级定义
	Target      string        `json:"target"`      // 目标连接类型：`main` 主站、`ex` 扩展站、`mac` MAC 主站、`mac_ex` MAC 扩展站、`icfqs` 资讯站
	Description string        `json:"description"` // 方法功能说明，展示在方法详情中
	Params      []methodParam `json:"params"`      // 参数定义列表，顺序即表单展示顺序；无参数时为空切片
}

// queryRequest 是 `/api/query` 接口收到的查询请求体。
type queryRequest struct {
	Method string            `json:"method"` // 方法标识，对应 `methodDef.Key`；为空或未登记时返回未知方法错误
	Params map[string]string `json:"params"` // 字符串形式的查询参数，键为 `methodParam.Key`；缺失时按空映射处理
}

// queryResponse 是 `/api/query` 接口返回给前端的查询结果，包含元信息、表格列与行、耗时与原始响应。
type queryResponse struct {
	Method        string         `json:"method"`            // 实际执行的方法标识，回显请求中的 `method`
	Label         string         `json:"label"`             // 方法展示名称，取自 `methodDef.Label`
	Group         string         `json:"group"`             // 方法分组名称，取自 `methodDef.Group`
	Target        string         `json:"target"`            // 目标连接类型，取值同 `methodDef.Target`
	Description   string         `json:"description"`       // 方法功能说明，取自 `methodDef.Description`
	Request       map[string]any `json:"request"`           // 经规整后的请求参数视图，用于前端回显本次实际入参
	Columns       []string       `json:"columns"`           // 表格列名，顺序与每行 `rows` 的列下标对应
	Rows          [][]string     `json:"rows"`              // 表格数据行，受最大行数 1000 截断；每行为字符串单元格切片
	TotalRows     int            `json:"total_rows"`        // 截断前的数据总行数，用于前端提示结果被裁剪
	DisplayedRows int            `json:"displayed_rows"`    // 实际返回的数据行数，等于 `Rows` 的长度
	DurationMS    int64          `json:"duration_ms"`       // 本次查询耗时，单位毫秒，从分发方法前开始计时
	Warning       string         `json:"warning,omitempty"` // 结果附带的提示信息（如部分字段语义待确认），无提示时省略该字段
	Raw           any            `json:"raw,omitempty"`     // 协议原始响应对象，供前端折叠查看；无原始响应时省略该字段
}

// queryPayload 是方法执行后返回给响应组装层的内部结果，字段不直接对外序列化。
type queryPayload struct {
	columns []string   // 表格列名，顺序与 `rows` 每行的列下标对应
	rows    [][]string // 表格数据行，组装响应前可由 `limitRows` 截断
	raw     any        // 协议原始响应对象，原样放入 `queryResponse.Raw`
	warning string     // 结果附带的提示信息，原样放入 `queryResponse.Warning`；为空表示无提示
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
		Key:         "stock_file_meta",
		Label:       "文件元信息",
		Group:       "股票资料",
		Target:      "main",
		Description: "获取主站文件元信息。",
		Params: []methodParam{
			{Key: "filename", Label: "文件名", Type: "text", Default: "block.dat"},
		},
	},
	{
		Key:         "stock_file_download",
		Label:       "文件片段下载",
		Group:       "股票资料",
		Target:      "main",
		Description: "下载主站文件片段并显示十六进制预览。",
		Params: []methodParam{
			{Key: "filename", Label: "文件名", Type: "text", Default: "block.dat"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "size", Label: "长度", Type: "number", Default: "1024"},
		},
	},
	{
		Key:         "stock_file_full",
		Label:       "完整文件下载",
		Group:       "股票资料",
		Target:      "main",
		Description: "下载主站完整文件并显示文本/十六进制预览。",
		Params: []methodParam{
			{Key: "filename", Label: "文件名", Type: "text", Default: "block.dat"},
		},
	},
	{
		Key:         "stock_table_file",
		Label:       "表格文件",
		Group:       "股票资料",
		Target:      "main",
		Description: "读取竖线分隔表格文件并按行展示。",
		Params: []methodParam{
			{Key: "filename", Label: "文件名", Type: "text", Default: "tdxhy.cfg"},
		},
	},
	{
		Key:         "stock_csv_file",
		Label:       "CSV 文件",
		Group:       "股票资料",
		Target:      "main",
		Description: "读取 CSV 文件并按列展示。",
		Params: []methodParam{
			{Key: "filename", Label: "文件名", Type: "text", Default: "spec/speckzzdata.txt"},
		},
	},
	{
		Key:         "stock_block_flat",
		Label:       "板块文件平铺",
		Group:       "股票资料",
		Target:      "main",
		Description: "按 block 文件逐行展开板块和代码。",
		Params: []methodParam{
			{Key: "filename", Label: "文件名", Type: "text", Default: "block_gn.dat"},
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
		Key:         "goods_count",
		Label:       "商品数量",
		Group:       "TDX 商品",
		Target:      "mac_ex",
		Description: "对齐 TDX goods_count，获取扩展市场商品总数。",
	},
	{
		Key:         "goods_category_list",
		Label:       "商品分类",
		Group:       "TDX 商品",
		Target:      "mac_ex",
		Description: "对齐 TDX goods_category_list，获取商品分类列表。",
	},
	{
		Key:         "goods_list",
		Label:       "商品列表",
		Group:       "TDX 商品",
		Target:      "mac_ex",
		Description: "对齐 TDX goods_list，获取扩展市场商品列表。",
		Params: []methodParam{
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "30"},
		},
	},
	{
		Key:         "goods_varieties",
		Label:       "商品品种",
		Group:       "TDX 商品",
		Target:      "mac_ex",
		Description: "对齐 TDX goods_varieties，获取期货/期权等合约品种。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "47"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "30", Help: "最大 1000"},
		},
	},
	{
		Key:         "goods_quote",
		Label:       "商品单只报价",
		Group:       "TDX 商品",
		Target:      "mac_ex",
		Description: "对齐 TDX goods_quotes 的单只调用形式。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
		},
	},
	{
		Key:         "goods_quotes",
		Label:       "商品批量报价",
		Group:       "TDX 商品",
		Target:      "mac_ex",
		Description: "对齐 TDX goods_quotes，批量获取商品报价。",
		Params: []methodParam{
			{Key: "markets", Label: "市场列表", Type: "text", Default: "74,71", Help: "可只填一个市场，多个代码时会自动复用"},
			{Key: "codes", Label: "代码列表", Type: "textarea", Default: "TSLA,00700"},
		},
	},
	{
		Key:         "goods_quotes_list",
		Label:       "商品行情列表",
		Group:       "TDX 商品",
		Target:      "mac_ex",
		Description: "对齐 TDX goods_quotes_list，获取可排序商品行情列表。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "74"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "30"},
			{Key: "sort_type", Label: "排序类型", Type: "number", Default: "0"},
			{Key: "reverse", Label: "是否倒序", Type: "text", Default: "false"},
		},
	},
	{
		Key:         "goods_kline",
		Label:       "商品 K 线",
		Group:       "TDX 商品",
		Target:      "mac_ex",
		Description: "对齐 TDX goods_kline，使用 MAC 统一 K 线协议。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
			{Key: "period", Label: "周期", Type: "number", Default: "4"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "数量", Type: "number", Default: "20"},
			{Key: "times", Label: "倍数", Type: "number", Default: "1"},
			{Key: "adjust", Label: "复权", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "goods_tick_chart",
		Label:       "商品分时",
		Group:       "TDX 商品",
		Target:      "mac_ex",
		Description: "对齐 TDX goods_tick_chart，获取实时或历史分时。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
			{Key: "query_date", Label: "日期", Type: "number", Default: "0", Help: "填 0 取实时"},
		},
	},
	{
		Key:         "goods_chart_sampling",
		Label:       "商品分时缩略",
		Group:       "TDX 商品",
		Target:      "mac_ex",
		Description: "对齐 TDX goods_chart_sampling，获取分时缩略采样。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
		},
	},
	{
		Key:         "goods_history_transaction",
		Label:       "商品历史成交",
		Group:       "TDX 商品",
		Target:      "mac_ex",
		Description: "对齐 TDX goods_history_transaction，获取商品历史逐笔成交。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
			{Key: "query_date", Label: "日期", Type: "number", Default: "20260330"},
			{Key: "count", Label: "数量", Type: "number", Default: "2000"},
		},
	},
	{
		Key:         "ex_file_meta",
		Label:       "扩展文件元信息",
		Group:       "扩展表格",
		Target:      "ex",
		Description: "获取扩展市场文件元信息。",
		Params: []methodParam{
			{Key: "filename", Label: "文件名", Type: "text", Default: "US_stock.dat"},
		},
	},
	{
		Key:         "ex_file_download",
		Label:       "扩展文件下载",
		Group:       "扩展表格",
		Target:      "ex",
		Description: "下载扩展市场文件片段并显示十六进制预览。",
		Params: []methodParam{
			{Key: "filename", Label: "文件名", Type: "text", Default: "US_stock.dat"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "size", Label: "长度", Type: "number", Default: "1024"},
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
		Key:         "mac_board_count",
		Label:       "MAC 板块数量",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 主站板块总数。",
		Params: []methodParam{
			{Key: "board_type", Label: "板块类型", Type: "number", Default: "255"},
		},
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
		Description: "按板块代码查询成分股，可透传排序参数。",
		Params: []methodParam{
			{Key: "board_symbol", Label: "板块代码", Type: "text", Default: "880761"},
			{Key: "count", Label: "总量", Type: "number", Default: "50"},
			{Key: "sort_type", Label: "排序类型", Type: "number", Default: "14"},
			{Key: "sort_order", Label: "排序顺序", Type: "number", Default: "1"},
		},
	},
	{
		Key:         "mac_board_members_quotes",
		Label:       "MAC 成分报价",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "按板块代码查询成分报价，可透传排序参数。",
		Params: []methodParam{
			{Key: "board_symbol", Label: "板块代码", Type: "text", Default: "880761"},
			{Key: "count", Label: "总量", Type: "number", Default: "50"},
			{Key: "sort_type", Label: "排序类型", Type: "number", Default: "14"},
			{Key: "sort_order", Label: "排序顺序", Type: "number", Default: "1"},
		},
	},
	{
		Key:         "mac_board_members_quotes_dynamic",
		Label:       "MAC 成分报价实验",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "按位图动态解析 MAC 成分报价",
		Params: []methodParam{
			{Key: "board_symbol", Label: "板块代码", Type: "text", Default: "880761"},
			{Key: "count", Label: "总量", Type: "number", Default: "20"},
			{Key: "sort_type", Label: "排序类型", Type: "number", Default: "14"},
			{Key: "sort_order", Label: "排序顺序", Type: "number", Default: "1"},
			{Key: "exclude_filter", Label: "排除过滤", Type: "number", Default: "0", Help: "位图值：新股=1 科创=2 ST=4 创业=8 互联互通=16 北交=32 核准制=64 注册制=128"},
			{Key: "field_bitmap", Label: "字段位图", Type: "text", Default: "", Help: "留空/default=默认位图，full=20字节全1，或填写40位hex"},
		},
	},
	{
		Key:         "mac_symbol_quotes",
		Label:       "MAC 批量报价实验",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "按位图动态解析 MAC 批量股票报价。",
		Params: []methodParam{
			{Key: "markets", Label: "市场列表", Type: "text", Default: "0,1"},
			{Key: "codes", Label: "代码列表", Type: "text", Default: "000001,600000"},
			{Key: "field_bitmap", Label: "字段位图", Type: "text", Default: "", Help: "留空/default=默认位图，full=20字节全1，或填写40位hex"},
		},
	},
	{
		Key:         "icfqs_topic_list_raw",
		Label:       "ICFQS 主题列表",
		Group:       "ICFQS 主题投资",
		Target:      "http",
		Description: "特色指数/主题列表。setcode: 1普通主题 2概念 4风格 5地域 6行业。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用默认 ICFQS 节点"},
			{Key: "category", Label: "分类", Type: "text", Default: "", Help: "如 X11、D01；留空不过滤"},
			{Key: "setcode", Label: "类型", Type: "text", Default: "", Help: "1普通主题 2概念 4风格 5地域 6行业；留空不过滤"},
			{Key: "page", Label: "页码", Type: "number", Default: "1"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0", Help: "ResultSets 中要展示的表，从 0 开始"},
		},
	},
	{
		Key:         "icfqs_topic_search_raw",
		Label:       "ICFQS 主题搜索",
		Group:       "ICFQS 主题投资",
		Target:      "http",
		Description: "按主题名称关键词搜索。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用默认 ICFQS 节点"},
			{Key: "keyword", Label: "关键词", Type: "text", Default: "有色"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "icfqs_new_topics_raw",
		Label:       "ICFQS 新增主题",
		Group:       "ICFQS 主题投资",
		Target:      "http",
		Description: "新增题材/主题列表。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用默认 ICFQS 节点"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "icfqs_hot_topics_raw",
		Label:       "ICFQS 热门主题",
		Group:       "ICFQS 主题投资",
		Target:      "http",
		Description: "热门主题列表。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用默认 ICFQS 节点"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "icfqs_topic_events_raw",
		Label:       "ICFQS 主题事件",
		Group:       "ICFQS 主题投资",
		Target:      "http",
		Description: "近期主题事件原始接口。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用默认 ICFQS 节点"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "icfqs_top_topics_raw",
		Label:       "ICFQS 领涨主题",
		Group:       "ICFQS 主题投资",
		Target:      "http",
		Description: "领涨板块/热门主题排行。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用默认 ICFQS 节点"},
			{Key: "topn", Label: "数量", Type: "number", Default: "10"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "icfqs_topic_detail_raw",
		Label:       "ICFQS 主题详情",
		Group:       "ICFQS 主题投资",
		Target:      "http",
		Description: "单主题详情说明。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用默认 ICFQS 节点"},
			{Key: "code", Label: "主题代码", Type: "text", Default: "308742"},
			{Key: "setcode", Label: "类型", Type: "text", Default: "2", Help: "1普通主题 2概念 4风格 5地域 6行业"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "icfqs_topic_kline_raw",
		Label:       "ICFQS 主题走势",
		Group:       "ICFQS 主题投资",
		Target:      "http",
		Description: "单主题趋势/K 线序列。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用默认 ICFQS 节点"},
			{Key: "code", Label: "主题代码", Type: "text", Default: "308742"},
			{Key: "setcode", Label: "类型", Type: "text", Default: "2"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "icfqs_topic_stocks_raw",
		Label:       "ICFQS 主题成分股",
		Group:       "ICFQS 主题投资",
		Target:      "http",
		Description: "主题关联成分股列表。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用默认 ICFQS 节点"},
			{Key: "code", Label: "主题代码", Type: "text", Default: "308742"},
			{Key: "setcode", Label: "类型", Type: "text", Default: "2"},
			{Key: "page", Label: "页码", Type: "number", Default: "1"},
			{Key: "size", Label: "每页数量", Type: "number", Default: "20"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "icfqs_topic_quotes_raw",
		Label:       "ICFQS 主题行情",
		Group:       "ICFQS 行情",
		Target:      "http",
		Description: "主题行情及成分股快照，默认展示 ResultSets[1]。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用默认 ICFQS 节点"},
			{Key: "setcodes", Label: "类型列表", Type: "text", Default: "2"},
			{Key: "codes", Label: "主题代码列表", Type: "text", Default: "308742"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "1"},
		},
	},
	{
		Key:         "icfqs_topic_rotation_raw",
		Label:       "ICFQS 主题轮动",
		Group:       "ICFQS 行情",
		Target:      "http",
		Description: "主题轮动排行。data_num: 1前后10 2前10 3前20 4前30 5后20 6后30。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用默认 ICFQS 节点"},
			{Key: "data_num", Label: "范围", Type: "number", Default: "1"},
			{Key: "data_type", Label: "指标", Type: "number", Default: "1", Help: "1涨幅 2涨停数 3成交额 4主力净额"},
			{Key: "data_date", Label: "周期", Type: "number", Default: "2", Help: "1近5日 2近10日 3近20日 4近30日"},
			{Key: "theme_type", Label: "主题类型", Type: "text", Default: "0", Help: "0全部；1/2/4/5/6 为具体类型"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "2"},
		},
	},
	{
		Key:         "icfqs_quotes_batch_raw",
		Label:       "ICFQS 批量行情",
		Group:       "ICFQS 行情",
		Target:      "http",
		Description: "ICFQS HQServ.PBCombHQ 批量行情原始接口。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用默认 ICFQS 节点"},
			{Key: "setcodes", Label: "市场列表", Type: "text", Default: "0,1"},
			{Key: "codes", Label: "代码列表", Type: "text", Default: "000001,600000"},
			{Key: "want_col", Label: "字段", Type: "text", Default: "CLOSE,NOW"},
		},
	},
	{
		Key:         "icfqs_lhb_detail_raw",
		Label:       "ICFQS 个股龙虎榜",
		Group:       "ICFQS 龙虎榜",
		Target:      "http",
		Description: "个股龙虎榜详情，默认使用 hot.icfqs.com。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用 hot.icfqs.com:7615"},
			{Key: "symbol", Label: "股票代码", Type: "text", Default: "000066"},
			{Key: "start_date", Label: "开始日期", Type: "text", Default: "", Help: "YYYY-MM-DD；留空用服务端默认"},
			{Key: "end_date", Label: "结束日期", Type: "text", Default: "", Help: "YYYY-MM-DD；留空用服务端默认"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "icfqs_yyb_detail_raw",
		Label:       "ICFQS 营业部龙虎榜",
		Group:       "ICFQS 龙虎榜",
		Target:      "http",
		Description: "营业部龙虎榜详情，默认使用 hot.icfqs.com。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用 hot.icfqs.com:7615"},
			{Key: "yyb_name", Label: "营业部", Type: "text", Default: "东方财富证券拉萨团结路第二证券营业部"},
			{Key: "start_date", Label: "开始日期", Type: "text", Default: ""},
			{Key: "end_date", Label: "结束日期", Type: "text", Default: ""},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "icfqs_yz_detail_raw",
		Label:       "ICFQS 游资详情",
		Group:       "ICFQS 龙虎榜",
		Target:      "http",
		Description: "游资席位详情，默认使用 hot.icfqs.com。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用 hot.icfqs.com:7615"},
			{Key: "code", Label: "游资代码", Type: "text", Default: ""},
			{Key: "start_date", Label: "开始日期", Type: "text", Default: ""},
			{Key: "end_date", Label: "结束日期", Type: "text", Default: ""},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "icfqs_mrfp_latest_date_raw",
		Label:       "ICFQS 复盘日期",
		Group:       "ICFQS 每日复盘",
		Target:      "http",
		Description: "每日复盘最新可用日期，默认使用 hot.icfqs.com。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用 hot.icfqs.com:7615"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "icfqs_mrfp_raw",
		Label:       "ICFQS 每日复盘",
		Group:       "ICFQS 每日复盘",
		Target:      "http",
		Description: "每日复盘原始接口。review_type 常用值：base、jrpm、ztfx、rdgl、rq。",
		Params: []methodParam{
			{Key: "host", Label: "Host", Type: "text", Default: "", Help: "留空使用 hot.icfqs.com:7615"},
			{Key: "review_type", Label: "类型", Type: "text", Default: "jrpm"},
			{Key: "date", Label: "日期", Type: "text", Default: "", Help: "YYYYMMDD 或 YYYY-MM-DD；留空取服务端默认"},
			{Key: "limit", Label: "数量", Type: "number", Default: "30"},
			{Key: "table_index", Label: "表序号", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "mac_quotes",
		Label:       "MAC 行情快照",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 主站单只标的快照与分时采样。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
			{Key: "query_date", Label: "查询日期", Type: "number", Default: "0", Help: "0=实时，YYYYMMDD=指定日期"},
		},
	},
	{
		Key:         "mac_capital_flow",
		Label:       "MAC 资金流向",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 主站资金流向，来自 0x1218 head=2 新协议。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
		},
	},
	{
		Key:         "mac_server_info",
		Label:       "MAC 服务信息",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 主站服务端交易日、交易时段和状态参数。",
	},
	{
		Key:         "mac_kline_offset",
		Label:       "MAC K线偏移",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC K线偏移信息协议，仅返回总量/回显类信息，不返回K线明细。",
		Params: []methodParam{
			{Key: "offset", Label: "偏移", Type: "number", Default: "0"},
			{Key: "count", Label: "总量", Type: "number", Default: "128000"},
		},
	},
	{
		Key:         "mac_file_list",
		Label:       "MAC 文件信息",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 文件列表/元信息查询。",
		Params: []methodParam{
			{Key: "filename", Label: "文件名", Type: "text", Default: "StockInfo.dat"},
			{Key: "offset", Label: "偏移", Type: "number", Default: "0"},
		},
	},
	{
		Key:         "mac_file_download",
		Label:       "MAC 文件下载",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 文件片段下载。",
		Params: []methodParam{
			{Key: "filename", Label: "文件名", Type: "text", Default: "StockInfo.dat"},
			{Key: "index", Label: "索引", Type: "number", Default: "1"},
			{Key: "offset", Label: "偏移", Type: "number", Default: "0"},
			{Key: "size", Label: "大小", Type: "number", Default: "512"},
		},
	},
	{
		Key:         "mac_transactions",
		Label:       "MAC 分时成交",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 主站逐笔成交，支持指定日期与自动分页。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "总量", Type: "number", Default: "20", Help: "单次最多 1000，超出会自动分页"},
			{Key: "query_date", Label: "查询日期", Type: "number", Default: "0", Help: "0=实时，YYYYMMDD=指定日期"},
		},
	},
	{
		Key:         "mac_auction",
		Label:       "MAC 竞价",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 主站竞价明细。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "总量", Type: "number", Default: "20", Help: "单次最多 500，超出会自动分页"},
		},
	},
	{
		Key:         "mac_tick_charts",
		Label:       "MAC 多日分时",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 主站多日分时，返回按日期分组的分时点与最新摘要。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
			{Key: "query_date", Label: "起始日期", Type: "number", Default: "0", Help: "0=服务端默认，YYYYMMDD=指定起始日期"},
			{Key: "days", Label: "天数", Type: "number", Default: "5"},
		},
	},
	{
		Key:         "mac_symbol_info",
		Label:       "MAC 股票摘要",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 主站单只股票摘要，包含活跃度、内外盘、VR 等字段。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "code", Label: "代码", Type: "text", Default: "000001"},
		},
	},
	{
		Key:         "mac_market_monitor",
		Label:       "MAC 市场监控",
		Group:       "MAC 协议",
		Target:      "mac",
		Description: "MAC 主站市场监控/异动列表，附带名称与原始 v1-v4 字段。",
		Params: []methodParam{
			{Key: "market", Label: "市场", Type: "number", Default: "0"},
			{Key: "start", Label: "起始", Type: "number", Default: "0"},
			{Key: "count", Label: "总量", Type: "number", Default: "20", Help: "单次最多 600，超出会自动分页"},
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
		Key:         "mac_ex_board_count",
		Label:       "扩展板块数量",
		Group:       "MAC 协议",
		Target:      "mac-ex",
		Description: "MAC 扩展站板块总数。",
		Params: []methodParam{
			{Key: "board_type", Label: "板块类型", Type: "number", Default: "0", Help: "0=HK_ALL 3=US_ALL"},
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
		Key:         "mac_ex_quotes",
		Label:       "扩展行情快照",
		Group:       "MAC 协议",
		Target:      "mac-ex",
		Description: "MAC 扩展站单只标的快照与分时采样。",
		Params: []methodParam{
			{Key: "market", Label: "分类", Type: "number", Default: "74"},
			{Key: "code", Label: "代码", Type: "text", Default: "TSLA"},
			{Key: "query_date", Label: "查询日期", Type: "number", Default: "0", Help: "0=实时，YYYYMMDD=指定日期"},
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

// methodMap 按方法标识索引方法定义，供请求分发时按 O(1) 快速查找；`init` 阶段会在方法定义
// 排序后重新生成，保证运行期查找顺序与前端展示顺序一致。
var methodMap = makeMethodMap(methodDefs)

// `mainHosts`、`exHosts`、`macHosts`、`macExHosts` 分别缓存内置主站、扩展站、MAC 主站与 MAC
// 扩展站地址列表，进程启动时一次性取好，供各客户端工厂按首个地址主连、其余地址候选回退。
var mainHosts = gotdx.MainHostAddresses()

var exHosts = gotdx.ExHostAddresses()

var macHosts = gotdx.MACHostAddresses()

var macExHosts = gotdx.MACExHostAddresses()

// makeMethodMap 把方法定义列表转换为按 `Key` 索引的查询表。
//
// `defs` 为输入方法定义切片；返回映射以各定义的 `Key` 为键，重复键以后出现者覆盖前者；
// 输入为空时返回空映射。
func makeMethodMap(defs []methodDef) map[string]methodDef {
	out := make(map[string]methodDef, len(defs))
	for _, def := range defs {
		out[def.Key] = def
	}
	return out
}

// executeQuery 按方法标识查找定义并执行查询，统计耗时、截断表格行数后组装响应。
//
// `req` 为 HTTP 层解析出的查询请求；`req.Params` 为 `nil` 时按空映射处理。方法未登记在
// `methodMap` 中、参数解析失败或协议执行失败时返回错误，由 HTTP 层统一转为 400 响应；成功时
// 返回包含列名、截断后数据行、耗时与原始响应的 `queryResponse`，表格行数上限为 1000。
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

// runMethod 是方法分发中枢：按 `def.Key` 进入对应分支，建立目标连接、解析参数、调用 `gotdx`
// 协议并把响应转换为表格载荷。
//
// `def` 为待执行方法的定义；`params` 为前端提交的字符串参数映射，键对应 `methodParam.Key`。
// 第一个返回值为表格载荷（列名、数据行、原始响应与提示）；第二个返回值是回显给前端的规整
// 请求参数视图；连接握手失败、参数解析失败或协议执行失败时返回错误，载荷与请求视图置空。
// 未登记的方法键落到默认分支并返回“暂不支持”错误。
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
				columns: []string{"datetime", "last", "open", "high", "low", "close", "vol", "amount", "turnover", "rise_price", "rise_rate"},
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
				// columns: []string{"datetime", "open", "high", "low", "close", "vol", "amount", "turnover"},
				columns: []string{"datetime", "last", "open", "high", "low", "close", "vol", "amount", "turnover", "rise_price", "rise_rate"},
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
			resp := &proto.GetHistoryTransactionDataWithTransReply{}
			size := uint16(1800)
			for startto := uint16(0); ; startto += size {
				reply, err := client.GetHistoryTransactionDataWithTrans(date, market, code, startto, size)
				if err != nil {
					return queryPayload{}, err
				}
				resp.Count += reply.Count
				resp.List = append(reply.List, resp.List...)
				if reply.Count < size {
					break
				}
			}
			return queryPayload{
				columns: []string{"time", "price", "vol", "num", "action"},
				rows:    rowsFromHistoryTransactionWithTrans(resp.List),
				raw:     resp.List,
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
			bars, err := client.GetIndexBars(types.KLINE_TYPE_DAILY, market, code, 0, 5)
			if err != nil {
				return queryPayload{}, err
			}
			rows := [][]string{
				{"summary", info.Code, info.ServerTime, formatFloat(info.Close), formatFloat(info.Open), formatFloat(info.High), formatFloat(info.Low), fmt.Sprintf("%d", info.UpCount), fmt.Sprintf("%d", info.DownCount)},
				{"momentum", "-", "-", "-", "-", "-", "-", fmt.Sprintf("%d", momentum.Count), fmt.Sprintf("%d", lastInt(momentum.Values))},
			}
			for _, bar := range bars.List {
				rows = append(rows, []string{"bar", code, bar.DateTime.Format("2006-01-02 15:04:05"), formatFloat(bar.Close), formatFloat(bar.Open), formatFloat(bar.High), formatFloat(bar.Low), "", ""})
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
				columns: []string{"time", "price", "matched", "unmatched", "flag"},
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
	case "stock_file_meta":
		filename := valueOrDefault(params, "filename", types.BlockFileDefault)
		request := map[string]any{"filename": filename}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetFileMeta(filename)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromFileMeta(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "stock_file_download":
		filename := valueOrDefault(params, "filename", types.BlockFileDefault)
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		size, err := parseUint32Value(params, "size", 1024)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"filename": filename, "start": start, "size": size}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.DownloadFile(filename, start, size)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows: [][]string{
					{"size", fmt.Sprintf("%d", reply.Size)},
					{"data_len", fmt.Sprintf("%d", len(reply.Data))},
				},
				raw: rawBytesPreview(reply.Data),
			}, nil
		})
		return payload, request, err
	case "stock_file_full":
		filename := valueOrDefault(params, "filename", types.BlockFileDefault)
		request := map[string]any{"filename": filename}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			content, err := client.DownloadFullFile(filename, 0)
			if err != nil {
				return queryPayload{}, err
			}
			raw := rawFullFilePreview(content)
			return queryPayload{
				columns: []string{"field", "value"},
				rows: [][]string{
					{"length", fmt.Sprintf("%d", len(content))},
					{"text_preview", raw["text_preview"].(string)},
				},
				raw:     raw,
				warning: "完整文件下载可能较慢；更适合文本配置和小型辅助文件。",
			}, nil
		})
		return payload, request, err
	case "stock_table_file":
		filename := valueOrDefault(params, "filename", "tdxhy.cfg")
		request := map[string]any{"filename": filename}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			rows, err := client.GetTableFile(filename)
			if err != nil {
				return queryPayload{}, err
			}
			columns, normalized := normalizeTableRows(rows, "col")
			return queryPayload{
				columns: columns,
				rows:    normalized,
				raw: map[string]any{
					"filename": filename,
					"rows":     limitRows(normalized, 50),
				},
			}, nil
		})
		return payload, request, err
	case "stock_csv_file":
		filename := valueOrDefault(params, "filename", "spec/speckzzdata.txt")
		request := map[string]any{"filename": filename}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			rows, err := client.GetCSVFile(filename)
			if err != nil {
				return queryPayload{}, err
			}
			columns, normalized := normalizeTableRows(rows, "col")
			return queryPayload{
				columns: columns,
				rows:    normalized,
				raw: map[string]any{
					"filename": filename,
					"rows":     limitRows(normalized, 50),
				},
			}, nil
		})
		return payload, request, err
	case "stock_block_flat":
		filename := valueOrDefault(params, "filename", types.BlockFileGN)
		request := map[string]any{"filename": filename}
		payload, err := withMainClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetParsedBlockFile(filename)
			if err != nil {
				return queryPayload{}, err
			}
			rows := rowsFromBlockFlat(reply)
			return queryPayload{
				columns: []string{"block_name", "block_type", "code_index", "code"},
				rows:    rows,
				raw:     limitRows(rows, 50),
			}, nil
		})
		return payload, request, err
	case "stock_block_grouped":
		filename := valueOrDefault(params, "filename", types.BlockFileFG)
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
				columns: []string{"goods_type", "code", "name", "abbr"},
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
			{"login_unknown", strings.Join(login.Unknown, ",")},
			{"server_name", info.ServerName},
			{"server_sign", info.ServerSign},
			{"server_sign2", info.ServerSign2},
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
				columns: []string{"datetime", "pre_close", "open", "high", "low", "close", "vol", "amount"},
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
				columns: []string{"datetime", "pre_close", "open", "high", "low", "close", "vol", "amount"},
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

	case "goods_count":
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			count, err := client.GoodsCount()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{columns: []string{"field", "value"}, rows: [][]string{{"count", fmt.Sprintf("%d", count)}}, raw: count}, nil
		})
		return payload, map[string]any{}, err
	case "goods_category_list":
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			items, err := client.GoodsCategoryList()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{columns: []string{"goods_type", "code", "name", "abbr"}, rows: rowsFromExCategoryList(items), raw: items}, nil
		})
		return payload, map[string]any{}, err
	case "goods_list":
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint16Value(params, "count", 30)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"start": start, "count": count}
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			items, err := client.GoodsList(start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{columns: []string{"market", "category", "code", "name", "desc_1", "desc_2", "desc_3"}, rows: rowsFromExList(items), raw: items}, nil
		})
		return payload, request, err
	case "goods_varieties":
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
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			items, err := client.GoodsVarieties(market, start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{columns: []string{"category", "name", "unknown", "index", "switch", "codes"}, rows: rowsFromExMapping2562(items), raw: items}, nil
		})
		return payload, request, err
	case "goods_quote":
		market, err := parseUint8Value(params, "market", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		request := map[string]any{"market": market, "code": code}
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			item, err := client.GoodsQuote(market, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{columns: []string{"code", "date", "close", "open", "high", "low", "vol", "avg"}, rows: rowsFromExQuotes([]proto.ExQuoteItem{*item}), raw: item}, nil
		})
		return payload, request, err
	case "goods_quotes":
		markets, err := parseUint8List(valueOrDefault(params, "markets", "74,71"))
		if err != nil {
			return queryPayload{}, nil, err
		}
		codes := parseCodeList(valueOrDefault(params, "codes", "TSLA,00700"))
		markets, err = expandUint8List(markets, len(codes), "markets")
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"markets": markets, "codes": codes}
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			items, err := client.GoodsQuotes(markets, codes)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{columns: []string{"code", "date", "close", "open", "high", "low", "vol", "avg"}, rows: rowsFromExQuotes(items), raw: items}, nil
		})
		return payload, request, err
	case "goods_quotes_list":
		market, err := parseUint8Value(params, "market", 74)
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
		request := map[string]any{"market": market, "start": start, "count": count, "sort_type": sortType, "reverse": reverse}
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			items, err := client.GoodsQuotesList(market, start, count, sortType, reverse)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{columns: []string{"code", "date", "close", "open", "high", "low", "vol", "avg"}, rows: rowsFromExQuotes(items), raw: items}, nil
		})
		return payload, request, err
	case "goods_kline":
		market, err := parseUint8Value(params, "market", 74)
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
		adjust, err := parseUint16Value(params, "adjust", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "code": code, "period": period, "start": start, "count": count, "times": times, "adjust": adjust}
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			items, err := client.GoodsKLine(market, code, period, start, count, times, adjust)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{columns: []string{"datetime", "open", "high", "low", "close", "vol", "amount", "float_shares", "turnover"}, rows: rowsFromMACSymbolBars(items), raw: items}, nil
		})
		return payload, request, err
	case "goods_tick_chart":
		market, err := parseUint8Value(params, "market", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		queryDate, err := parseUint32Value(params, "query_date", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "code": code, "query_date": queryDate}
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			items, err := client.GoodsTickChart(market, code, queryDate)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{columns: []string{"time", "price", "avg", "vol", "momentum"}, rows: rowsFromMACQuoteChart(items), raw: items}, nil
		})
		return payload, request, err
	case "goods_chart_sampling":
		market, err := parseUint8Value(params, "market", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		request := map[string]any{"market": market, "code": code}
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			prices, err := client.GoodsChartSampling(market, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{columns: []string{"index", "price", "pre_close", "change"}, rows: rowsFromSampling(0, prices), raw: prices}, nil
		})
		return payload, request, err
	case "goods_history_transaction":
		market, err := parseUint8Value(params, "market", 74)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		queryDate, err := parseUint32Value(params, "query_date", 20260330)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint32Value(params, "count", gotdx.DefaultGoodsHistoryTransactionCount)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "code": code, "query_date": queryDate, "count": count}
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			items, err := client.GoodsHistoryTransaction(market, code, queryDate, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{columns: []string{"time", "price", "vol", "trade_count", "buy_or_sell"}, rows: rowsFromMACTransactions(items), raw: items}, nil
		})
		return payload, request, err
	case "ex_file_meta":
		filename := valueOrDefault(params, "filename", "US_stock.dat")
		request := map[string]any{"filename": filename}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExGetFileMeta(filename)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromFileMeta(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "ex_file_download":
		filename := valueOrDefault(params, "filename", "US_stock.dat")
		start, err := parseUint32Value(params, "start", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		size, err := parseUint32Value(params, "size", 1024)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"filename": filename, "start": start, "size": size}
		payload, err := withExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.ExDownloadFile(filename, start, size)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows: [][]string{
					{"size", fmt.Sprintf("%d", reply.Size)},
					{"data_len", fmt.Sprintf("%d", len(reply.Data))},
				},
				raw: rawBytesPreview(reply.Data),
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
	case "mac_board_count":
		boardType, err := parseUint16Value(params, "board_type", types.BoardTypeAll)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"board_type": boardType}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			total, err := client.MACBoardCount(boardType)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    [][]string{{"total", fmt.Sprintf("%d", total)}},
				raw:     map[string]any{"board_type": boardType, "total": total},
			}, nil
		})
		return payload, request, err
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
		sortType, err := parseUint16Value(params, "sort_type", 14)
		if err != nil {
			return queryPayload{}, nil, err
		}
		sortOrder, err := parseUint16Value(params, "sort_order", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"board_symbol": boardSymbol, "count": count, "sort_type": sortType, "sort_order": sortOrder}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			list, err := client.MACBoardMembersWithSort(boardSymbol, count, sortType, sortOrder)
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
		sortType, err := parseUint16Value(params, "sort_type", 14)
		if err != nil {
			return queryPayload{}, nil, err
		}
		sortOrder, err := parseUint8Value(params, "sort_order", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"board_symbol": boardSymbol, "count": count, "sort_type": sortType, "sort_order": sortOrder}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			list, err := client.MACBoardMembersQuotesWithSort(boardSymbol, count, sortType, sortOrder)
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
	case "mac_board_members_quotes_dynamic":
		boardSymbol := valueOrDefault(params, "board_symbol", "880761")
		count, err := parseUint32Value(params, "count", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		sortType, err := parseUint16Value(params, "sort_type", 14)
		if err != nil {
			return queryPayload{}, nil, err
		}
		sortOrder, err := parseUint8Value(params, "sort_order", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		excludeFilter, err := parseUint8Value(params, "exclude_filter", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		fieldBitmapText := valueOrDefault(params, "field_bitmap", "")
		fieldBitmap, err := parseMACBoardMembersQuotesFieldBitmap(fieldBitmapText)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{
			"board_symbol":     boardSymbol,
			"count":            count,
			"sort_type":        sortType,
			"sort_order":       sortOrder,
			"exclude_filter":   excludeFilter,
			"field_bitmap":     fieldBitmapText,
			"field_bitmap_hex": hex.EncodeToString(fieldBitmap[:]),
		}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.MACBoardMembersQuotesDynamicWithFilter(boardSymbol, count, sortType, sortOrder, fieldBitmap, gotdx.MACFilterType(excludeFilter))
			if err != nil {
				return queryPayload{}, err
			}
			columns := columnsFromMACBoardMemberQuotesDynamic(reply)
			rows := rowsFromMACBoardMemberQuotesDynamic(reply)
			return queryPayload{
				columns: columns,
				rows:    rows,
				raw: map[string]any{
					"field_bitmap_hex": hex.EncodeToString(reply.FieldBitmap[:]),
					"active_fields":    reply.ActiveFields,
					"field_columns":    []string{"bit", "name", "format", "description"},
					"field_rows":       rowsFromMACDynamicFieldDefs(reply.ActiveFields),
					"count":            reply.Count,
					"total":            reply.Total,
					"stocks":           reply.Stocks,
				},
				warning: "这是实验接口，字段命名以协议比对为主，未知字段可能继续调整。",
			}, nil
		})
		return payload, request, err
	case "mac_quotes":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		queryDate, err := parseUint32Value(params, "query_date", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		request := map[string]any{"market": market, "code": code, "query_date": queryDate}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.MACQuotesWithDate(market, code, queryDate)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"time", "price", "avg", "vol", "momentum"},
				rows:    rowsFromMACQuoteChart(reply.ChartData),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "mac_symbol_quotes":
		markets, err := parseUint8List(valueOrDefault(params, "markets", "0,1"))
		if err != nil {
			return queryPayload{}, nil, err
		}
		codes := parseCodeList(valueOrDefault(params, "codes", "000001,600000"))
		fieldBitmapText := valueOrDefault(params, "field_bitmap", "")
		fieldBitmap, err := parseMACBoardMembersQuotesFieldBitmap(fieldBitmapText)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{
			"markets":          markets,
			"codes":            codes,
			"field_bitmap":     fieldBitmapText,
			"field_bitmap_hex": hex.EncodeToString(fieldBitmap[:]),
		}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.MACSymbolQuotes(markets, codes, fieldBitmap)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: columnsFromMACSymbolQuotes(reply),
				rows:    rowsFromMACSymbolQuotes(reply),
				raw: map[string]any{
					"field_bitmap_hex": hex.EncodeToString(reply.FieldBitmap[:]),
					"active_fields":    rowsFromMACDynamicFieldDefs(reply.ActiveFields),
					"reply":            reply,
				},
				warning: "这是实验接口，字段命名以协议比对为主，未知字段可能继续调整。",
			}, nil
		})
		return payload, request, err
	case "icfqs_topic_list_raw":
		host := valueOrDefault(params, "host", "")
		category := valueOrDefault(params, "category", "")
		setcode := valueOrDefault(params, "setcode", "")
		page, err := parseUint32Value(params, "page", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "category": category, "setcode": setcode, "page": page, "table_index": tableIndex}
		raw, err := newICFQSViewerClient(host).ICFQSTopicListRaw(context.Background(), category, setcode, int(page))
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_topic_search_raw":
		host := valueOrDefault(params, "host", "")
		keyword := valueOrDefault(params, "keyword", "有色")
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "keyword": keyword, "table_index": tableIndex}
		raw, err := newICFQSViewerClient(host).ICFQSSearchTopicsRaw(context.Background(), keyword)
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_new_topics_raw":
		host := valueOrDefault(params, "host", "")
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "table_index": tableIndex}
		raw, err := newICFQSViewerClient(host).ICFQSNewTopicsRaw(context.Background())
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_hot_topics_raw":
		host := valueOrDefault(params, "host", "")
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "table_index": tableIndex}
		raw, err := newICFQSViewerClient(host).ICFQSHotTopicsRaw(context.Background())
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_topic_events_raw":
		host := valueOrDefault(params, "host", "")
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "table_index": tableIndex}
		raw, err := newICFQSViewerClient(host).ICFQSEventsRaw(context.Background())
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_top_topics_raw":
		host := valueOrDefault(params, "host", "")
		topN, err := parseUint32Value(params, "topn", 10)
		if err != nil {
			return queryPayload{}, nil, err
		}
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "topn": topN, "table_index": tableIndex}
		raw, err := newICFQSViewerClient(host).ICFQSTopTopicsRaw(context.Background(), int(topN))
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_topic_detail_raw":
		host := valueOrDefault(params, "host", "")
		code := valueOrDefault(params, "code", "308742")
		setcode := valueOrDefault(params, "setcode", "2")
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "code": code, "setcode": setcode, "table_index": tableIndex}
		raw, err := newICFQSViewerClient(host).ICFQSTopicDetailRaw(context.Background(), code, setcode)
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_topic_kline_raw":
		host := valueOrDefault(params, "host", "")
		code := valueOrDefault(params, "code", "308742")
		setcode := valueOrDefault(params, "setcode", "2")
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "code": code, "setcode": setcode, "table_index": tableIndex}
		raw, err := newICFQSViewerClient(host).ICFQSTopicKLineRaw(context.Background(), code, setcode)
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_topic_stocks_raw":
		host := valueOrDefault(params, "host", "")
		code := valueOrDefault(params, "code", "308742")
		setcode := valueOrDefault(params, "setcode", "2")
		page, err := parseUint32Value(params, "page", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		size, err := parseUint32Value(params, "size", 20)
		if err != nil {
			return queryPayload{}, nil, err
		}
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "code": code, "setcode": setcode, "page": page, "size": size, "table_index": tableIndex}
		raw, err := newICFQSViewerClient(host).ICFQSTopicStocksRaw(context.Background(), code, setcode, int(page), int(size))
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_topic_quotes_raw":
		host := valueOrDefault(params, "host", "")
		codeList, err := parseICFQSCodeParams(params, "2", "308742")
		if err != nil {
			return queryPayload{}, nil, err
		}
		tableIndex, err := parseICFQSTableIndex(params, 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "codes": codeList, "table_index": tableIndex}
		raw, err := newICFQSViewerClient(host).ICFQSTopicQuotesRaw(context.Background(), codeList)
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_topic_rotation_raw":
		host := valueOrDefault(params, "host", "")
		dataNum, err := parseUint32Value(params, "data_num", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		dataType, err := parseUint32Value(params, "data_type", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		dataDate, err := parseUint32Value(params, "data_date", 2)
		if err != nil {
			return queryPayload{}, nil, err
		}
		themeType := valueOrDefault(params, "theme_type", "0")
		tableIndex, err := parseICFQSTableIndex(params, 2)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "data_num": dataNum, "data_type": dataType, "data_date": dataDate, "theme_type": themeType, "table_index": tableIndex}
		raw, err := newICFQSViewerClient(host).ICFQSTopicRotationRaw(context.Background(), int(dataNum), int(dataType), int(dataDate), themeType)
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_quotes_batch_raw":
		host := valueOrDefault(params, "host", "")
		codeList, err := parseICFQSCodeParams(params, "0,1", "000001,600000")
		if err != nil {
			return queryPayload{}, nil, err
		}
		wantCols := parseCodeList(valueOrDefault(params, "want_col", "CLOSE,NOW"))
		request := map[string]any{"host": host, "codes": codeList, "want_col": wantCols}
		raw, err := newICFQSViewerClient(host).ICFQSQuotesBatchRaw(context.Background(), codeList, wantCols)
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSListItems(raw), request, nil
	case "icfqs_lhb_detail_raw":
		host := valueOrDefault(params, "host", "")
		symbol := valueOrDefault(params, "symbol", "000066")
		startDate := valueOrDefault(params, "start_date", "")
		endDate := valueOrDefault(params, "end_date", "")
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "symbol": symbol, "start_date": startDate, "end_date": endDate, "table_index": tableIndex}
		raw, err := newICFQSViewerHotClient(host).ICFQSLHBDetailRaw(context.Background(), symbol, startDate, endDate)
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_yyb_detail_raw":
		host := valueOrDefault(params, "host", "")
		yybName := valueOrDefault(params, "yyb_name", "")
		startDate := valueOrDefault(params, "start_date", "")
		endDate := valueOrDefault(params, "end_date", "")
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "yyb_name": yybName, "start_date": startDate, "end_date": endDate, "table_index": tableIndex}
		raw, err := newICFQSViewerHotClient(host).ICFQSYYBDetailRaw(context.Background(), yybName, startDate, endDate)
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_yz_detail_raw":
		host := valueOrDefault(params, "host", "")
		code := valueOrDefault(params, "code", "")
		startDate := valueOrDefault(params, "start_date", "")
		endDate := valueOrDefault(params, "end_date", "")
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "code": code, "start_date": startDate, "end_date": endDate, "table_index": tableIndex}
		raw, err := newICFQSViewerHotClient(host).ICFQSYZDetailRaw(context.Background(), code, startDate, endDate)
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_mrfp_latest_date_raw":
		host := valueOrDefault(params, "host", "")
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "table_index": tableIndex}
		raw, err := newICFQSViewerHotClient(host).ICFQSMRFPLatestDateRaw(context.Background())
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "icfqs_mrfp_raw":
		host := valueOrDefault(params, "host", "")
		reviewType := valueOrDefault(params, "review_type", "jrpm")
		date := valueOrDefault(params, "date", "")
		limit, err := parseUint32Value(params, "limit", 30)
		if err != nil {
			return queryPayload{}, nil, err
		}
		tableIndex, err := parseICFQSTableIndex(params, 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"host": host, "review_type": reviewType, "date": date, "limit": limit, "table_index": tableIndex}
		raw, err := newICFQSViewerHotClient(host).ICFQSMRFPRaw(context.Background(), reviewType, date, int(limit))
		if err != nil {
			return queryPayload{}, request, err
		}
		return payloadFromICFQSTable(raw, tableIndex), request, nil
	case "mac_capital_flow":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		request := map[string]any{"market": market, "code": code}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.MACCapitalFlow(market, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "query_info", "ext", "today_main_in", "today_main_out", "today_retail_in", "today_retail_out", "today_main_net_in", "today_retail_net_in", "five_day_main_buy", "five_day_main_sell", "five_day_super_net", "five_day_large_net", "five_day_medium_net", "five_day_small_net", "five_day_main_net_in"},
				rows:    rowsFromMACCapitalFlow(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "mac_server_info":
		request := map[string]any{}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.MACServerInfo()
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromMACServerInfo(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "mac_kline_offset":
		offset, err := parseUint32Value(params, "offset", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		count, err := parseUint32Value(params, "count", gotdx.DefaultMACKLineOffsetCount)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"offset": offset, "count": count}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.MACKLineOffset(offset, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    rowsFromMACKLineOffset(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "mac_file_list":
		filename := valueOrDefault(params, "filename", "StockInfo.dat")
		offset, err := parseUint32Value(params, "offset", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"filename": filename, "offset": offset}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.MACFileList(filename, offset)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"offset", "size", "flag", "hash"},
				rows:    rowsFromMACFileList(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "mac_file_download":
		filename := valueOrDefault(params, "filename", "StockInfo.dat")
		index, err := parseUint32Value(params, "index", 1)
		if err != nil {
			return queryPayload{}, nil, err
		}
		offset, err := parseUint32Value(params, "offset", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		size, err := parseUint32Value(params, "size", 512)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"filename": filename, "index": index, "offset": offset, "size": size}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.GetMACFileDownload(filename, index, offset, size)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows: [][]string{
					{"index", fmt.Sprintf("%d", reply.Index)},
					{"size", fmt.Sprintf("%d", reply.Size)},
					{"data_len", fmt.Sprintf("%d", len(reply.Data))},
				},
				raw: rawFullFilePreview(reply.Data),
			}, nil
		})
		return payload, request, err
	case "mac_transactions":
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
		queryDate, err := parseUint32Value(params, "query_date", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "code": code, "start": start, "count": count, "query_date": queryDate}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			list, err := client.MACTransactionsWithDate(market, code, start, count, queryDate)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"time", "price", "vol", "trade_count", "action"},
				rows:    rowsFromMACTransactions(list),
				raw:     list,
			}, nil
		})
		return payload, request, err
	case "mac_auction":
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
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			list, err := client.MACAuction(market, code, start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"time", "price", "matched", "unmatched", "flag"},
				rows:    rowsFromMACAuction(list),
				raw:     list,
			}, nil
		})
		return payload, request, err
	case "mac_tick_charts":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		queryDate, err := parseUint32Value(params, "query_date", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		days, err := parseUint16Value(params, "days", 5)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"market": market, "code": code, "query_date": queryDate, "days": days}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.MACTickCharts(market, code, queryDate, days)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"date", "pre_close", "time", "price", "avg", "vol", "unknown"},
				rows:    rowsFromMACTickCharts(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "mac_symbol_info":
		market, err := parseUint8Value(params, "market", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "000001")
		request := map[string]any{"market": market, "code": code}
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.MACSymbolInfo(market, code)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"market", "code", "name", "datetime", "activity", "pre_close", "open", "high", "low", "close", "momentum", "vol", "amount", "inside_volume", "outside_volume", "decimal", "unknown_a", "unknown_b", "unknown_c", "vr", "turnover", "avg"},
				rows:    rowsFromMACSymbolInfo(reply),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	case "mac_market_monitor":
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
		payload, err := withMACClient(func(client *gotdx.Client) (queryPayload, error) {
			list, err := client.MACMarketMonitor(market, start, count)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"index", "market", "code", "name", "time", "desc", "value", "unusual_type", "v1", "v2", "v3", "v4"},
				rows:    rowsFromMACMarketMonitor(list),
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
				columns: []string{"datetime", "open", "high", "low", "close", "vol", "amount", "float_shares", "turnover"},
				rows:    rowsFromMACSymbolBars(list),
				raw:     list,
			}, nil
		})
		return payload, request, err
	case "mac_ex_board_count":
		boardType, err := parseUint16Value(params, "board_type", types.ExBoardTypeHKAll)
		if err != nil {
			return queryPayload{}, nil, err
		}
		request := map[string]any{"board_type": boardType}
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			total, err := client.MACBoardCount(boardType)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"field", "value"},
				rows:    [][]string{{"total", fmt.Sprintf("%d", total)}},
				raw:     map[string]any{"board_type": boardType, "total": total},
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
				columns: []string{"datetime", "open", "high", "low", "close", "vol", "amount", "float_shares", "turnover"},
				rows:    rowsFromMACSymbolBars(list),
				raw:     list,
			}, nil
		})
		return payload, request, err
	case "mac_ex_quotes":
		market, err := parseUint8Value(params, "market", types.ExCategoryUSStock)
		if err != nil {
			return queryPayload{}, nil, err
		}
		queryDate, err := parseUint32Value(params, "query_date", 0)
		if err != nil {
			return queryPayload{}, nil, err
		}
		code := valueOrDefault(params, "code", "TSLA")
		request := map[string]any{"market": market, "code": code, "query_date": queryDate}
		payload, err := withMACExClient(func(client *gotdx.Client) (queryPayload, error) {
			reply, err := client.MACQuotesWithDate(market, code, queryDate)
			if err != nil {
				return queryPayload{}, err
			}
			return queryPayload{
				columns: []string{"time", "price", "avg", "vol", "momentum"},
				rows:    rowsFromMACQuoteChart(reply.ChartData),
				raw:     reply,
			}, nil
		})
		return payload, request, err
	default:
		return queryPayload{}, nil, fmt.Errorf("暂不支持的方法: %s", def.Key)
	}
}

// newMainClient 按内置主站地址池构造主站客户端，连接超时固定为 6 秒。
//
// 返回的客户端尚未发起连接：`mainHosts[0]` 作为主连地址，`mainHosts[1:]` 作为候选地址池，
// 主连失败时由客户端内部依次回退；调用方需自行调用 `Connect` 建立连接。
func newMainClient() *gotdx.Client {
	return gotdx.New(
		gotdx.WithTCPAddress(mainHosts[0]),
		gotdx.WithTCPAddressPool(mainHosts[1:]...),
		gotdx.WithTimeoutSec(6),
	)
}

// newExClient 按内置扩展站地址池构造扩展行情客户端，连接超时固定为 6 秒。
//
// 返回的客户端尚未发起连接：`exHosts[0]` 作为主连地址，`exHosts[1:]` 作为候选地址池；
// 调用方需自行调用 `ConnectEx` 建立连接。
func newExClient() *gotdx.Client {
	return gotdx.NewEx(
		gotdx.WithExTCPAddress(exHosts[0]),
		gotdx.WithExTCPAddressPool(exHosts[1:]...),
		gotdx.WithTimeoutSec(6),
	)
}

// newMACClient 按内置 MAC 主站地址池构造 MAC 行情客户端，连接超时固定为 6 秒。
//
// 返回的客户端尚未发起连接：`macHosts[0]` 作为主连地址，`macHosts[1:]` 作为候选地址池；
// 调用方需自行调用 `ConnectMAC` 建立连接。
func newMACClient() *gotdx.Client {
	return gotdx.NewMAC(
		gotdx.WithMacTCPAddress(macHosts[0]),
		gotdx.WithMacTCPAddressPool(macHosts[1:]...),
		gotdx.WithTimeoutSec(6),
	)
}

// newMACExClient 按内置 MAC 扩展站地址池构造 MAC 扩展行情客户端，连接超时固定为 6 秒。
//
// 返回的客户端尚未发起连接：`macExHosts[0]` 作为主连地址，`macExHosts[1:]` 作为候选地址池；
// 调用方需自行调用 `ConnectMAC` 建立连接。
func newMACExClient() *gotdx.Client {
	return gotdx.NewMACEx(
		gotdx.WithMacExTCPAddress(macExHosts[0]),
		gotdx.WithMacExTCPAddressPool(macExHosts[1:]...),
		gotdx.WithTimeoutSec(6),
	)
}

// withMainClient 建立主站连接、执行 `fn`，函数返回后断开连接。
//
// `fn` 接收已连接的主站客户端并返回表格载荷；握手失败时直接返回空载荷与错误，不进入 `fn`；
// `fn` 返回的载荷与错误原样向上传递，连接在 `defer` 中统一断开。
func withMainClient(fn func(*gotdx.Client) (queryPayload, error)) (queryPayload, error) {
	client := newMainClient()
	defer client.Disconnect()
	if _, err := client.Connect(); err != nil {
		return queryPayload{}, err
	}
	return fn(client)
}

// withExClient 建立扩展站连接、执行 `fn`，函数返回后断开连接。
//
// `fn` 接收已连接的扩展站客户端并返回表格载荷；握手失败时直接返回空载荷与错误，不进入 `fn`；
// `fn` 返回的载荷与错误原样向上传递，连接在 `defer` 中统一断开。
func withExClient(fn func(*gotdx.Client) (queryPayload, error)) (queryPayload, error) {
	client := newExClient()
	defer client.Disconnect()
	if _, err := client.ConnectEx(); err != nil {
		return queryPayload{}, err
	}
	return fn(client)
}

// withMACClient 建立 MAC 主站连接、执行 `fn`，函数返回后断开连接。
//
// `fn` 接收已连接的 MAC 主站客户端并返回表格载荷；握手失败时直接返回空载荷与错误，不进入
// `fn`；`fn` 返回的载荷与错误原样向上传递，连接在 `defer` 中统一断开。
func withMACClient(fn func(*gotdx.Client) (queryPayload, error)) (queryPayload, error) {
	client := newMACClient()
	defer client.Disconnect()
	if err := client.ConnectMAC(); err != nil {
		return queryPayload{}, err
	}
	return fn(client)
}

// withMACExClient 建立 MAC 扩展站连接、执行 `fn`，函数返回后断开连接。
//
// `fn` 接收已连接的 MAC 扩展站客户端并返回表格载荷；握手失败时直接返回空载荷与错误，不进入
// `fn`；`fn` 返回的载荷与错误原样向上传递，连接在 `defer` 中统一断开。
func withMACExClient(fn func(*gotdx.Client) (queryPayload, error)) (queryPayload, error) {
	client := newMACExClient()
	defer client.Disconnect()
	if err := client.ConnectMAC(); err != nil {
		return queryPayload{}, err
	}
	return fn(client)
}

// currentMainHost 返回主站客户端当前连接的 `host:port` 地址。
//
// `client` 为已构造的主站客户端；客户端为 `nil` 时返回空字符串，未连接时返回底层地址查询
// 结果。
func currentMainHost(client *gotdx.Client) string {
	if client == nil {
		return ""
	}
	return client.CurrentAddress()
}

// currentExHost 返回扩展站客户端当前连接的 `host:port` 地址。
//
// `client` 为已构造的扩展站客户端；客户端为 `nil` 时返回空字符串。
func currentExHost(client *gotdx.Client) string {
	if client == nil {
		return ""
	}
	return client.CurrentAddress()
}

// currentMACHost 返回 MAC 主站客户端当前连接的 `host:port` 地址。
//
// `client` 为已构造的 MAC 主站客户端；客户端为 `nil` 时返回空字符串。
func currentMACHost(client *gotdx.Client) string {
	if client == nil {
		return ""
	}
	return client.CurrentAddress()
}

// currentMACExHost 返回 MAC 扩展站客户端当前连接的 `host:port` 地址。
//
// `client` 为已构造的 MAC 扩展站客户端；客户端为 `nil` 时返回空字符串。
func currentMACExHost(client *gotdx.Client) string {
	if client == nil {
		return ""
	}
	return client.CurrentAddress()
}

// rowsFromHostInfos 把内置主机条目转换为表格行，每行依次为名称、IP、端口与 `host:port` 四列。
//
// `items` 为待转换主机条目切片；返回行数与输入条目数一致，输入为空时返回非 `nil` 的空切片。
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

// rowsFromHostProbeResults 把探测结果转换为表格行，每行依次为名称、地址、延迟（毫秒）、
// 可达性与错误信息五列。
//
// `items` 为待转换探测结果切片；仅可达条目的延迟列填入毫秒整数，不可达条目延迟列留空，
// 错误列填入拨号失败原因。
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

// parseCodeList 按逗号、换行、回车、制表符与空白拆分代码字符串，忽略空片段。
//
// `value` 为前端提交的单值或多值代码字符串；返回去除首尾空白后的非空片段切片，输入为空时
// 返回非 `nil` 的空切片。
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

// parseUint8List 把市场类别字符串解析为 `uint8` 切片，按十进制无符号整数解析。
//
// `value` 为逗号或空白分隔的类别编号字符串，拆分规则同 `parseCodeList`；任一片段不是十进制
// 整数或超出 `uint8` 取值范围（0 至 255）时返回错误，成功时返回与片段顺序一致的编号切片。
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

// expandUint8List 把单值市场类别列表广播补齐到与代码列表相同的长度。
//
// `values` 为类别编号切片；`targetLen` 为代码列表长度。目标长度为 0 时原样返回；两者长度
// 一致时原样返回；`values` 仅含一个元素且 `targetLen` 大于 1 时把该值复制填满；其余长度
// 组合视为数量不匹配并返回错误。`label` 为类别名称，仅用于错误提示文案。
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

// parseUint8Value 从参数映射中读取指定键并按 `uint8` 十进制整数解析。
//
// `params` 为前端提交的字符串参数映射；`key` 为参数键；`def` 为参数缺失或去除空白后为空时
// 使用的默认值。数值超出 0 至 255 范围或不是合法十进制整数时返回错误。
func parseUint8Value(params map[string]string, key string, def uint8) (uint8, error) {
	value := valueOrDefault(params, key, strconv.FormatUint(uint64(def), 10))
	v, err := strconv.ParseUint(strings.TrimSpace(value), 10, 8)
	if err != nil {
		return 0, fmt.Errorf("%s 必须是 uint8", key)
	}
	return uint8(v), nil
}

// parseUint16Value 从参数映射中读取指定键并按 `uint16` 十进制整数解析。
//
// `params` 为前端提交的字符串参数映射；`key` 为参数键；`def` 为参数缺失或去除空白后为空时
// 使用的默认值。数值超出 0 至 65535 范围或不是合法十进制整数时返回错误。
func parseUint16Value(params map[string]string, key string, def uint16) (uint16, error) {
	value := valueOrDefault(params, key, strconv.FormatUint(uint64(def), 10))
	v, err := strconv.ParseUint(strings.TrimSpace(value), 10, 16)
	if err != nil {
		return 0, fmt.Errorf("%s 必须是 uint16", key)
	}
	return uint16(v), nil
}

// parseUint32Value 从参数映射中读取指定键并按 `uint32` 十进制整数解析。
//
// `params` 为前端提交的字符串参数映射；`key` 为参数键；`def` 为参数缺失或去除空白后为空时
// 使用的默认值。数值超出 `uint32` 取值范围或不是合法十进制整数时返回错误。
func parseUint32Value(params map[string]string, key string, def uint32) (uint32, error) {
	value := valueOrDefault(params, key, strconv.FormatUint(uint64(def), 10))
	v, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s 必须是 uint32", key)
	}
	return uint32(v), nil
}

// parseBoolValue 从参数映射中读取指定键并识别常见布尔写法。
//
// `params` 为前端提交的字符串参数映射；`key` 为参数键；`def` 为参数缺失或为空时使用的默认值。
// 不区分大小写，`1`、`true`、`yes`、`y`、`on` 解析为真，`0`、`false`、`no`、`n`、`off`、
// 空串解析为假；其余写法无法识别时返回错误。
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

// valueOrDefault 读取参数映射中指定键的字符串值。
//
// `params` 为前端提交的参数映射；`key` 为参数键；`def` 为回退默认值。参数键不存在或取值
// 去除首尾空白后为空时返回 `def`，否则返回去除空白前的原始字符串。
func valueOrDefault(params map[string]string, key string, def string) string {
	value, ok := params[key]
	if !ok || strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

// limitRows 把表格行数截断到 `max` 行以内，避免超长结果撑爆前端表格。
//
// `rows` 为原始数据行切片；`max` 为允许返回的最大行数。未超限时原样返回原切片，超限时返回
// 前 `max` 行的切片视图，不复制行内容。
func limitRows(rows [][]string, max int) [][]string {
	if len(rows) <= max {
		return rows
	}
	return rows[:max]
}

// rawTextPreview 为文本响应生成预览对象，附带总长度与是否截断标记。
//
// `content` 为待预览文本；预览文本上限为 4000 字节，未超限时 `preview` 为全文、
// `truncated` 为 `false`，超限时 `preview` 为前 4000 字节、`truncated` 为 `true`，`length`
// 始终为原始字节长度。
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

// rawBytesPreview 为二进制响应生成预览对象，包含字节长度与十六进制预览文本。
//
// `content` 为待预览字节切片；`length` 为原始字节数，`hex_preview` 为小写十六进制文本，
// 最长展示 512 个十六进制字符。
func rawBytesPreview(content []byte) map[string]any {
	return map[string]any{
		"length":      len(content),
		"hex_preview": preview(hex.EncodeToString(content), 512),
	}
}

// rawFullFilePreview 为下载完成的整文件生成预览对象，同时提供字节长度、十六进制预览与解码后
// 文本预览。
//
// `content` 为下载得到的文件原始字节；`length` 为字节数，`hex_preview` 最长 512 个十六进制
// 字符，`text_preview` 为按协议编码解码后的文本、最长 2000 字符。
func rawFullFilePreview(content []byte) map[string]any {
	text := proto.Utf8ToGbk(content)
	return map[string]any{
		"length":       len(content),
		"hex_preview":  preview(hex.EncodeToString(content), 512),
		"text_preview": preview(text, 2000),
	}
}

// parseMACBoardMembersQuotesFieldBitmap 解析 MAC 板块成员行情的字段位图，返回 20 字节位图。
//
// `value` 为前端提交的位图选择：空串或 `default`（不区分大小写）使用内置默认位图，`full`
// 使用全字段位图；其余输入先去除 `0x` 前缀与空格、逗号、下划线，再按 40 位十六进制字符
// 解码为 20 字节。去除辅助字符后长度不是 40 位，或十六进制格式非法时返回错误。
func parseMACBoardMembersQuotesFieldBitmap(value string) ([20]byte, error) {
	text := strings.TrimSpace(value)
	switch strings.ToLower(text) {
	case "", "default":
		return gotdx.DefaultMACBoardMembersQuotesFieldBitmap(), nil
	case "full":
		return gotdx.FullMACBoardMembersQuotesFieldBitmap(), nil
	}

	text = strings.TrimPrefix(text, "0x")
	text = strings.TrimPrefix(text, "0X")
	replacer := strings.NewReplacer(" ", "", ",", "", "_", "")
	text = replacer.Replace(text)
	if len(text) != 40 {
		return [20]byte{}, fmt.Errorf("field_bitmap 需要 40 位 hex，当前长度=%d", len(text))
	}
	decoded, err := hex.DecodeString(text)
	if err != nil {
		return [20]byte{}, err
	}
	var bitmap [20]byte
	copy(bitmap[:], decoded)
	return bitmap, nil
}

// rowsFromRawReply 把未解析的原始协议响应转换为两行键值表格：响应字节长度与截断后的十六进制
// 预览。
//
// `reply` 为原始响应对象，为 `nil` 时返回 `nil`；十六进制预览最长展示 512 个字符。
func rowsFromRawReply(reply *proto.RawDataReply) [][]string {
	if reply == nil {
		return nil
	}
	return [][]string{
		{"length", fmt.Sprintf("%d", reply.Length)},
		{"hex_preview", preview(reply.Hex, 512)},
	}
}

// rowsFromQuoteDetail 转换盘口详细行情条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromQuoteList 转换排序行情快照条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromSecurityList 转换证券代码与名称列表条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromSecurityFeature452 转换证券扩展信息条目为表格行；P1、P2 字段语义待确认。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromSecurityBars 转换股票 K 线条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
func rowsFromSecurityBars(items []proto.SecurityBar) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.DateTime.Format(time.DateTime),
			formatFloat(item.PreClose),
			formatFloat(item.Open),
			formatFloat(item.High),
			formatFloat(item.Low),
			formatFloat(item.Close),
			formatFloat(item.Vol),
			formatFloat(item.Amount),
			formatFloat(item.Turnover),
			formatFloat(item.RisePrice),
			formatFloat(item.RiseRate),
		})
	}
	return rows
}

// rowsFromMinuteTimeData 转换当日分时图条目为表格行，首列为分时点序号。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromHistoryMinuteTimeData 转换历史分时图条目为表格行，首列为分时点序号。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromTransaction 转换当日逐笔成交条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromHistoryTransaction 转换历史逐笔成交条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
func rowsFromHistoryTransaction(items []proto.HistoryTransactionData) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Time.Format("2006-01-02 15:04"),
			formatFloat(item.Price),
			fmt.Sprintf("%d", item.Vol),
			fmt.Sprintf("%d", item.Num),
			fmt.Sprintf("%d", item.BuyOrSell),
		})
	}
	return rows
}

// rowsFromHistoryTransactionWithTrans 转换带买卖方向的历史逐笔成交条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
func rowsFromHistoryTransactionWithTrans(items []proto.HistoryTransactionDataWithTrans) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Time.Format("2006-01-02 15:04"),
			formatFloat(item.Price),
			fmt.Sprintf("%d", item.Vol),
			fmt.Sprintf("%d", item.Num),
			item.Action,
		})
	}
	return rows
}

// rowsFromEncryptedQuotes 转换加密行情快照条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromAnnouncement 把服务商公告响应转换为键值表格行；`item` 为 `nil` 时返回 `nil`。
//
// 公告无内容（`HasContent` 为 `false`）时仅输出存在标记一行；有内容时追加失效日期、标题、
// 作者与正文四行键值。
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

// rowsFromServerInfo 把主站服务信息响应转换为键值表格行；`item` 为 `nil` 时返回 `nil`。
//
// `unknown1`、`unknown2`、`unknown3` 等字段的具体业务含义语义待确认，按原始数值或片段
// 拼接展示。
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
		{"unknown1", strings.Join(item.Unknown1[:], ",")},
		{"unknown2", strings.Join(item.Unknown2[:], ",")},
		{"unknown3", fmt.Sprintf("%d,%d,%d", item.Unknown3[0], item.Unknown3[1], item.Unknown3[2])},
		{"region", fmt.Sprintf("%d", item.Region)},
		{"switch", fmt.Sprintf("%d", item.MaybeSwitch)},
	}
}

// rowsFromHistoryOrders 转换历史委托档位条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromAuction 转换集合竞价记录为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
func rowsFromAuction(items []proto.AuctionData) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Time,
			formatFloat(item.Price),
			fmt.Sprintf("%d", item.Matched),
			fmt.Sprintf("%d", item.Unmatched),
			fmt.Sprintf("%d", item.Flag),
		})
	}
	return rows
}

// rowsFromTopBoard 把九类排行榜依次展平为表格行，首列标注榜单类别名称。
//
// `reply` 为排行榜响应，依次处理涨幅、跌幅、振幅、涨速、跌速、量比、正委托比、负委托比、
// 换手率九个列表；每行其余列为市场编号、代码、价格与榜单数值。各列表为空时对应榜单不产生
// 行，全部为空则返回空切片。
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

// rowsFromUnusual 转换异动监控记录为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromVolumeProfile 把价位成交分布条目转换为表格行，末列附带整体换手率。
//
// `items` 为价位分布条目切片，每行依次为价位、成交量、主动买量、主动卖量；`turnover` 为
// 响应返回的整体换手率（比率口径，单位随协议约定，语义待确认），同一数值附加到每一分布行末尾。
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

// rowsFromCompanyBundle 聚合公司信息章节、除权除息计数与财务摘要为键值表格行。
//
// `bundle` 为高层方法聚合出的公司信息包：先逐章节输出章节名与截断正文（预览上限 120 字符），
// 再输出除权除息条目总数；财务摘要非空时追加一行更新日期、营业收入与净利润摘要。
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

// rowsFromCompanyCategories 把 F10 章节目录条目转换为表格行。
//
// `items` 为目录条目切片，每行依次为章节名称、文件名、正文起始字节偏移与正文长度；文件名、
// 偏移与长度用于后续按条目读取章节正文。
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

// rowsFromFinanceInfo 把财务信息响应转换为键值表格行；`item` 为 `nil` 时返回 `nil`。
//
// 行内依次输出代码、更新日期、上市日期、股本、每股指标、资产负债与利润等字段；金额类字段按
// 两位小数展示，每股收益与每股净资产按四位小数展示。
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

// rowsFromXDXR 转换除权除息历史记录为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromFileMeta 把文件元信息响应转换为键值表格行；`item` 为 `nil` 时返回 `nil`。
//
// `size` 为文件字节长度，`hash_value` 为定长校验值的十六进制文本；`unknown1`、`unknown2`
// 字段业务含义语义待确认，按原始数值展示。
func rowsFromFileMeta(item *proto.GetFileMetaReply) [][]string {
	if item == nil {
		return nil
	}
	return [][]string{
		{"size", fmt.Sprintf("%d", item.Size)},
		{"unknown1", fmt.Sprintf("%d", item.Unknown1)},
		{"hash_value", hex.EncodeToString(item.HashValue[:])},
		{"unknown2", fmt.Sprintf("%d", item.Unknown2)},
	}
}

// rowsFromBlockFlat 转换扁平化板块成分条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
func rowsFromBlockFlat(items []gotdx.BlockFlatItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.BlockName,
			fmt.Sprintf("%d", item.BlockType),
			fmt.Sprintf("%d", item.CodeIndex),
			item.Code,
		})
	}
	return rows
}

// rowsFromBlockGroups 转换板块分组为表格行，成分代码仅展示前 5 个。
//
// `groups` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromExQuotes 转换扩展市场行情条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromExCategoryList 转换扩展市场品种分类条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
func rowsFromExCategoryList(items []proto.ExCategoryItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		goodsType := fmt.Sprintf("%d", item.GoodsType)
		if item.GoodsTypeName != "" {
			goodsType = fmt.Sprintf("%d:%s", item.GoodsType, item.GoodsTypeName)
		}
		rows = append(rows, []string{
			goodsType,
			fmt.Sprintf("%d", item.Code),
			item.Name,
			item.Abbr,
		})
	}
	return rows
}

// rowsFromExList 把扩展市场标的列表条目转换为表格行。
//
// `items` 为标的条目切片；`Desc` 为变长附加数值数组，最多展示前三项，缺失的对应列留空。每行
// 依次为市场编号、类别编号、代码、名称与三个附加数值列。
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

// rowsFromExListExtra 转换扩展市场试验列表条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromExKLine 转换扩展市场 K 线条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
func rowsFromExKLine(items []proto.ExKLineItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.DateTime.Format(time.DateTime),
			formatFloat(item.PreClose),
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

// rowsFromExExperiment2487 把试验报价 `0x2487` 响应转换为键值表格行；`item` 为 `nil` 时
// 返回 `nil`。
//
// 行情字段含义按通用报价口径展示，`u1` 等无明确命名的字段语义待确认；`tail_hex` 为响应尾部
// 原始字节的十六进制预览，最长 256 字符。
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

// rowsFromExExperiment2488 转换试验报价 `0x2488` 条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromExTick 转换扩展市场分时图条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromExHistoryTransaction 转换扩展市场历史逐笔成交条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromExBoardList 转换扩展市场板块榜单条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromExMapping2562 转换扩展市场代码映射条目为表格行；未知字段语义待确认。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromMACBoardList 转换 MAC 板块榜单条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromMACBoardMembers 转换 MAC 板块成员基础信息为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// rowsFromMACBoardMemberQuotes 转换 MAC 板块成员行情条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
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

// columnsFromMACBoardMemberQuotesDynamic 生成动态板块行情列名；`reply` 为 `nil` 时返回 `nil`。
//
// 前三个固定列为市场、代码、名称，其后按服务端 `ActiveFields` 给出的字段定义顺序拼接动态
// 字段名，列顺序与 `rowsFromMACBoardMemberQuotesDynamic` 输出的行内值顺序一一对应。
func columnsFromMACBoardMemberQuotesDynamic(reply *proto.MACBoardMembersQuotesDynamicReply) []string {
	if reply == nil {
		return nil
	}
	columns := []string{"market", "symbol", "name"}
	for _, field := range reply.ActiveFields {
		columns = append(columns, field.Name)
	}
	return columns
}

// rowsFromMACBoardMemberQuotesDynamic 把动态板块成员行情响应转换为表格行；`reply` 为 `nil`
// 时返回 `nil`。
//
// 每行先输出市场、代码、名称三个固定值，再按 `ActiveFields` 顺序从成员 `Values` 映射中取出
// 对应字段值并格式化；映射中缺失的字段输出空字符串。
func rowsFromMACBoardMemberQuotesDynamic(reply *proto.MACBoardMembersQuotesDynamicReply) [][]string {
	if reply == nil {
		return nil
	}
	rows := make([][]string, 0, len(reply.Stocks))
	for _, item := range reply.Stocks {
		row := []string{
			fmt.Sprintf("%d", item.Market),
			item.Symbol,
			item.Name,
		}
		for _, field := range reply.ActiveFields {
			row = append(row, formatAny(item.Values[field.Name]))
		}
		rows = append(rows, row)
	}
	return rows
}

// columnsFromMACSymbolQuotes 生成 MAC 标的行情列名；`reply` 为 `nil` 时返回 `nil`。
//
// 前三个固定列为市场、代码、名称，其后按服务端 `ActiveFields` 顺序拼接动态字段名。
func columnsFromMACSymbolQuotes(reply *proto.MACSymbolQuotesReply) []string {
	if reply == nil {
		return nil
	}
	columns := []string{"market", "symbol", "name"}
	for _, field := range reply.ActiveFields {
		columns = append(columns, field.Name)
	}
	return columns
}

// rowsFromMACSymbolQuotes 把 MAC 标的动态行情响应转换为表格行；`reply` 为 `nil` 时返回 `nil`。
//
// 每行先输出市场、代码、名称三个固定值，再按 `ActiveFields` 顺序从标的 `Values` 映射中取出
// 对应字段值并格式化；映射中缺失的字段输出空字符串。
func rowsFromMACSymbolQuotes(reply *proto.MACSymbolQuotesReply) [][]string {
	if reply == nil {
		return nil
	}
	rows := make([][]string, 0, len(reply.Stocks))
	for _, item := range reply.Stocks {
		row := []string{
			fmt.Sprintf("%d", item.Market),
			item.Symbol,
			item.Name,
		}
		for _, field := range reply.ActiveFields {
			row = append(row, formatAny(item.Values[field.Name]))
		}
		rows = append(rows, row)
	}
	return rows
}

// rowsFromMACDynamicFieldDefs 转换 MAC 动态字段定义为表格行。
//
// `fields` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
func rowsFromMACDynamicFieldDefs(fields []proto.MACDynamicFieldDef) [][]string {
	rows := make([][]string, 0, len(fields))
	for _, field := range fields {
		rows = append(rows, []string{
			fmt.Sprintf("%d", field.Bit),
			field.Name,
			field.Format,
			field.Description,
		})
	}
	return rows
}

// rowsFromMACQuoteChart 转换 MAC 商品分时图条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
func rowsFromMACQuoteChart(items []proto.MACQuoteChartItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Time,
			formatFloat(item.Price),
			formatFloat(item.Avg),
			fmt.Sprintf("%d", item.Vol),
			formatFloat(item.Momentum),
		})
	}
	return rows
}

// rowsFromMACTransactions 转换 MAC 逐笔成交条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
func rowsFromMACTransactions(items []proto.MACTransactionItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Time,
			formatFloat(item.Price),
			fmt.Sprintf("%d", item.Vol),
			fmt.Sprintf("%d", item.TradeCount),
			fmt.Sprintf("%d", item.BuyOrSell),
		})
	}
	return rows
}

// rowsFromMACCapitalFlow 把 MAC 资金流响应转换为单行汇总表格；`reply` 为 `nil` 时返回 `nil`。
//
// 行内依次为市场编号、查询说明、附加文本、当日主力/散户流入流出与净流入，以及五日主力、超大、
// 大、中、小单净额等汇总数值；金额单位随协议约定，语义待确认。
func rowsFromMACCapitalFlow(reply *proto.MACCapitalFlowReply) [][]string {
	if reply == nil {
		return nil
	}
	return [][]string{{
		fmt.Sprintf("%d", reply.Market),
		reply.QueryInfo,
		reply.Ext,
		formatFloat(reply.TodayMainIn),
		formatFloat(reply.TodayMainOut),
		formatFloat(reply.TodayRetailIn),
		formatFloat(reply.TodayRetailOut),
		formatFloat(reply.TodayMainNetIn),
		formatFloat(reply.TodayRetailNetIn),
		formatFloat(reply.FiveDayMainBuy),
		formatFloat(reply.FiveDayMainSell),
		formatFloat(reply.FiveDaySuperNet),
		formatFloat(reply.FiveDayLargeNet),
		formatFloat(reply.FiveDayMediumNet),
		formatFloat(reply.FiveDaySmallNet),
		formatFloat(reply.FiveDayMainNetIn),
	}}
}

// rowsFromMACServerInfo 把 MAC 服务端信息响应转换为键值表格行；`reply` 为 `nil` 时返回 `nil`。
//
// 两段交易时段分别经 `formatMACTradingSessions` 拼接为可读文本；`TS1` 至 `TS3`、
// `MarketParam1`、`MarketParam2` 等参数字段的具体业务含义语义待确认，按原始数值或十六进制
// 文本展示。
func rowsFromMACServerInfo(reply *proto.MACServerInfoReply) [][]string {
	if reply == nil {
		return nil
	}
	return [][]string{
		{"count", fmt.Sprintf("%d", reply.Count)},
		{"flags", reply.FlagsHex},
		{"tag", reply.Tag},
		{"today", reply.Today},
		{"ts1", fmt.Sprintf("%d", reply.TS1)},
		{"sessions_1", formatMACTradingSessions(reply.Sessions1)},
		{"sessions_2", formatMACTradingSessions(reply.Sessions2)},
		{"flag", fmt.Sprintf("%d", reply.Flag)},
		{"last_trading_day", reply.LastTradingDay},
		{"ts2", fmt.Sprintf("%d", reply.TS2)},
		{"last_trading_day_2", reply.LastTradingDay2},
		{"ts3", fmt.Sprintf("%d", reply.TS3)},
		{"market_param_1", fmt.Sprintf("%d", reply.MarketParam1)},
		{"market_param_2", fmt.Sprintf("%d", reply.MarketParam2)},
		{"extra", reply.ExtraHex},
	}
}

// formatMACTradingSessions 把交易时段列表拼接为可读文本，时段之间以分号加空格分隔。
//
// `sessions` 为时段切片；每个时段输出 `开盘时间-收盘时间`，开盘与收盘分钟数同时为 0 的空时段
// 被跳过；全部为空时段时返回空字符串。
func formatMACTradingSessions(sessions []proto.MACTradingSession) string {
	parts := make([]string, 0, len(sessions))
	for _, session := range sessions {
		if session.OpenMinutes == 0 && session.CloseMinutes == 0 {
			continue
		}
		parts = append(parts, session.Open+"-"+session.Close)
	}
	return strings.Join(parts, "; ")
}

// rowsFromMACKLineOffset 把偏移 K 线响应转换为两行键值表格：总量与本次返回条数；`reply`
// 为 `nil` 时返回 `nil`。
func rowsFromMACKLineOffset(reply *proto.MACKLineOffsetReply) [][]string {
	if reply == nil {
		return nil
	}
	return [][]string{
		{"total", fmt.Sprintf("%d", reply.Total)},
		{"returned", fmt.Sprintf("%d", reply.Returned)},
	}
}

// rowsFromMACFileList 把 MAC 文件列表元信息响应转换为单行表格；`reply` 为 `nil` 时返回 `nil`。
//
// 行内依次为下载偏移、文件字节长度、标志位与校验哈希文本。
func rowsFromMACFileList(reply *proto.MACFileListReply) [][]string {
	if reply == nil {
		return nil
	}
	return [][]string{{
		fmt.Sprintf("%d", reply.Offset),
		fmt.Sprintf("%d", reply.Size),
		fmt.Sprintf("%d", reply.Flag),
		reply.Hash,
	}}
}

// rowsFromMACAuction 转换 MAC 集合竞价记录为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
func rowsFromMACAuction(items []proto.MACAuctionItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Time,
			formatFloat(item.Price),
			fmt.Sprintf("%d", item.Matched),
			fmt.Sprintf("%d", item.Unmatched),
			fmt.Sprintf("%d", item.Flag),
		})
	}
	return rows
}

// rowsFromMACTickCharts 按交易日遍历分时图，把每一天的全部分时点展平为表格行；`reply` 为
// `nil` 时返回 `nil`。
//
// 每行依次为交易日、当日昨收、分时时间、价格、均价、成交量与一个含义语义待确认的占位数值；
// 日期与昨收在同一天内重复附加到每个分时点行上。
func rowsFromMACTickCharts(reply *proto.MACTickChartsReply) [][]string {
	if reply == nil {
		return nil
	}
	rows := make([][]string, 0)
	for _, day := range reply.Charts {
		for _, tick := range day.Ticks {
			rows = append(rows, []string{
				day.Date,
				formatFloat(day.PreClose),
				tick.Time,
				formatFloat(tick.Price),
				formatFloat(tick.Avg),
				fmt.Sprintf("%d", tick.Vol),
				fmt.Sprintf("%d", tick.Unknown),
			})
		}
	}
	return rows
}

// rowsFromMACSymbolInfo 把 MAC 标的概况响应转换为单行表格；`reply` 为 `nil` 时返回 `nil`。
//
// 行内依次为市场编号、代码、名称、行情时间、活跃度、昨收与 OHLC、动量、成交量额、内外盘量、
// 小数位数等字段；`UnknownA`、`UnknownB`、`UnknownC` 等 `unknown` 系列字段语义待确认，按
// 原始数值展示。
func rowsFromMACSymbolInfo(reply *proto.MACSymbolInfoReply) [][]string {
	if reply == nil {
		return nil
	}
	return [][]string{{
		fmt.Sprintf("%d", reply.Market),
		reply.Code,
		reply.Name,
		reply.DateTime.Format(time.DateTime),
		fmt.Sprintf("%d", reply.Activity),
		formatFloat(reply.PreClose),
		formatFloat(reply.Open),
		formatFloat(reply.High),
		formatFloat(reply.Low),
		formatFloat(reply.Close),
		formatFloat(reply.Momentum),
		fmt.Sprintf("%d", reply.Vol),
		formatFloat(reply.Amount),
		fmt.Sprintf("%d", reply.InsideVolume),
		fmt.Sprintf("%d", reply.OutsideVolume),
		fmt.Sprintf("%d", reply.Decimal),
		fmt.Sprintf("%d", reply.UnknownA),
		formatFloat(reply.UnknownB),
		fmt.Sprintf("%d", reply.UnknownC),
		formatFloat(reply.VR),
		formatFloat(reply.Turnover),
		formatFloat(reply.Avg),
	}}
}

// rowsFromMACMarketMonitor 把 MAC 市场监控条目转换为表格行。
//
// `items` 为监控条目切片，每行依次为序号、市场编号、代码、名称、时间、描述、监控数值与异动
// 类型；`V1` 至 `V4` 附加字段语义待确认，按原始整型或浮点值展示。
func rowsFromMACMarketMonitor(items []proto.MACMarketMonitorItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Index),
			fmt.Sprintf("%d", item.Market),
			item.Code,
			item.Name,
			item.Time,
			item.Desc,
			item.Value,
			fmt.Sprintf("%d", item.UnusualType),
			fmt.Sprintf("%d", item.V1),
			formatFloat(item.V2),
			formatFloat(item.V3),
			formatFloat(item.V4),
		})
	}
	return rows
}

// newICFQSViewerClient 构造一个 ICFQS 数据客户端。
//
// `host` 为 `host:port` 形式的自定义站点地址；去除空白后为空时使用客户端内置默认站点，非空
// 时通过地址选项指定该自定义站点。返回的客户端尚未发起握手，连接动作由调用方完成。
func newICFQSViewerClient(host string) *gotdx.ICFQSClient {
	if strings.TrimSpace(host) == "" {
		return gotdx.NewICFQS()
	}
	return gotdx.NewICFQS(gotdx.WithICFQSAddress(host))
}

// newICFQSViewerHotClient 构造一个 ICFQS 热门行情客户端。
//
// `host` 为 `host:port` 形式的自定义站点地址；去除空白后为空时使用热门行情内置默认站点，非空
// 时通过地址选项指定该自定义站点。返回的客户端尚未发起握手。
func newICFQSViewerHotClient(host string) *gotdx.ICFQSClient {
	if strings.TrimSpace(host) == "" {
		return gotdx.NewICFQSHot()
	}
	return gotdx.NewICFQSHot(gotdx.WithICFQSAddress(host))
}

// parseICFQSTableIndex 从参数映射中读取 `table_index` 并解析为结果集索引。
//
// `params` 为前端提交的参数映射；`def` 为参数缺失或为空时使用的默认索引。索引按 `uint32`
// 十进制整数解析，格式非法或超出取值范围时返回错误；成功时返回非负整数索引。
func parseICFQSTableIndex(params map[string]string, def int) (int, error) {
	value, err := parseUint32Value(params, "table_index", uint32(def))
	if err != nil {
		return 0, err
	}
	return int(value), nil
}

// parseICFQSCodeParams 解析 ICFQS 板块代码与证券代码参数，组装为代码对切片。
//
// `params` 为前端参数映射；`setcodesDefault`、`codesDefault` 分别为参数缺失时使用的板块代码、
// 证券代码默认串。`codes` 或 `setcodes` 解析后为空时返回错误；仅给出单个板块代码时广播到全部
// 证券；两侧数量最终不一致时返回数量不匹配错误。成功时返回与证券代码顺序一一对应的
// `ICFQSCode` 切片。
func parseICFQSCodeParams(params map[string]string, setcodesDefault string, codesDefault string) ([]gotdx.ICFQSCode, error) {
	setcodes := parseCodeList(valueOrDefault(params, "setcodes", setcodesDefault))
	codes := parseCodeList(valueOrDefault(params, "codes", codesDefault))
	if len(codes) == 0 {
		return nil, fmt.Errorf("codes 不能为空")
	}
	if len(setcodes) == 0 {
		return nil, fmt.Errorf("setcodes 不能为空")
	}
	if len(setcodes) == 1 && len(codes) > 1 {
		setcode := setcodes[0]
		setcodes = make([]string, len(codes))
		for i := range setcodes {
			setcodes[i] = setcode
		}
	}
	if len(setcodes) != len(codes) {
		return nil, fmt.Errorf("setcodes 数量必须和 codes 数量一致")
	}
	out := make([]gotdx.ICFQSCode, 0, len(codes))
	for i, code := range codes {
		out = append(out, gotdx.ICFQSCode{Setcode: setcodes[i], Code: code})
	}
	return out, nil
}

// payloadFromICFQSTable 把指定 `ResultSets` 表组装为表格载荷。
//
// `raw` 为 ICFQS 接口返回的原始映射；`tableIndex` 为待展示结果集的非负索引。表格列名与数据行
// 由 `rowsFromICFQSTable` 生成，原始响应整体放入载荷 `raw` 字段供前端查看，并附带展示提示。
func payloadFromICFQSTable(raw map[string]any, tableIndex int) queryPayload {
	columns, rows := rowsFromICFQSTable(raw, tableIndex)
	return queryPayload{
		columns: columns,
		rows:    rows,
		raw: map[string]any{
			"table_index": tableIndex,
			"tables":      gotdx.ICFQSFormatTables(raw),
			"raw":         raw,
		},
		warning: "ICFQS 原始接口直接展示指定 ResultSets 表，完整响应见 Raw。",
	}
}

// payloadFromICFQSListItems 把 ICFQS `ListItem` 结构组装为表格载荷。
//
// `raw` 为接口返回的原始映射；列名与数据行由 `rowsFromICFQSListItems` 生成，原始响应整体放入
// 载荷 `raw` 字段，并附带展示提示。
func payloadFromICFQSListItems(raw map[string]any) queryPayload {
	columns, rows := rowsFromICFQSListItems(raw)
	return queryPayload{
		columns: columns,
		rows:    rows,
		raw:     raw,
		warning: "ICFQS 原始接口直接展示 ListItem，完整响应见 Raw。",
	}
}

// rowsFromICFQSTables 取第 0 号 `ResultSets` 表，返回列名与数据行；无结果集时两者均为 `nil`。
//
// `raw` 为接口原始映射；固定以索引 0 调用 `rowsFromICFQSTable`，参数语义与该函数一致。
func rowsFromICFQSTables(raw map[string]any) ([]string, [][]string) {
	return rowsFromICFQSTable(raw, 0)
}

// rowsFromICFQSTable 把指定索引的 `ResultSets` 表转换为列名与数据行。
//
// `raw` 为接口原始映射；`tableIndex` 为结果集索引，越界或结果集缺失时返回 `nil` 列名与 `nil`
// 行。列名优先取结果集自带定义，缺失时按最长行长度生成默认列名；行长度短于列数时用空字符串
// 补齐。
func rowsFromICFQSTable(raw map[string]any, tableIndex int) ([]string, [][]string) {
	resultSets, _ := raw["ResultSets"].([]any)
	if tableIndex < 0 || tableIndex >= len(resultSets) {
		return nil, nil
	}
	resultSet, _ := resultSets[tableIndex].(map[string]any)
	columns := columnsFromICFQSResultSet(resultSet)
	rawRows, _ := resultSet["Content"].([]any)
	if len(columns) == 0 {
		columns = columnsFromICFQSContent(rawRows)
	}
	rows := make([][]string, 0, len(rawRows))
	for _, rawRow := range rawRows {
		values, _ := rawRow.([]any)
		row := make([]string, 0, len(columns))
		for i := range columns {
			if i < len(values) {
				row = append(row, formatAny(values[i]))
			} else {
				row = append(row, "")
			}
		}
		rows = append(rows, row)
	}
	return columns, rows
}

// columnsFromICFQSResultSet 从结果集映射中提取列名。
//
// `resultSet` 为单个结果集映射；优先使用 `ColName` 给出的列名数组，缺失或为空时回退到
// `ColDes` 中各列定义的 `Name` 字段，名称为空的定义被跳过；两种来源都缺失时返回空切片。
func columnsFromICFQSResultSet(resultSet map[string]any) []string {
	if rawColumns, ok := resultSet["ColName"].([]any); ok && len(rawColumns) > 0 {
		columns := make([]string, 0, len(rawColumns))
		for _, rawColumn := range rawColumns {
			columns = append(columns, fmt.Sprint(rawColumn))
		}
		return columns
	}
	rawColumnDefs, _ := resultSet["ColDes"].([]any)
	columns := make([]string, 0, len(rawColumnDefs))
	for _, rawColumnDef := range rawColumnDefs {
		columnDef, _ := rawColumnDef.(map[string]any)
		if name := strings.TrimSpace(fmt.Sprint(columnDef["Name"])); name != "" {
			columns = append(columns, name)
		}
	}
	return columns
}

// columnsFromICFQSContent 在结果集未提供列名时，按最长行长度生成默认列名。
//
// `rawRows` 为原始行切片；列名依次为 `col_0`、`col_1` 等；所有行均为空时返回空切片。
func columnsFromICFQSContent(rawRows []any) []string {
	maxLen := 0
	for _, rawRow := range rawRows {
		values, _ := rawRow.([]any)
		if len(values) > maxLen {
			maxLen = len(values)
		}
	}
	columns := make([]string, 0, maxLen)
	for i := 0; i < maxLen; i++ {
		columns = append(columns, fmt.Sprintf("col_%d", i))
	}
	return columns
}

// rowsFromICFQSListItems 按 `ListHead` 表头解析 `ListItem` 列表，返回列名与数据行。
//
// `raw` 为接口原始映射；列名取自 `ListHead.ItemHead`，逐行取 `ListItem` 内 `Item` 数组对应位置
// 的值并格式化，行长度不足时用空字符串补齐；表头或条目缺失时对应结果为空切片。
func rowsFromICFQSListItems(raw map[string]any) ([]string, [][]string) {
	head, _ := raw["ListHead"].(map[string]any)
	rawHeads, _ := head["ItemHead"].([]any)
	columns := make([]string, 0, len(rawHeads))
	for _, rawHead := range rawHeads {
		columns = append(columns, fmt.Sprint(rawHead))
	}
	rawItems, _ := raw["ListItem"].([]any)
	rows := make([][]string, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, _ := rawItem.(map[string]any)
		values, _ := item["Item"].([]any)
		row := make([]string, 0, len(columns))
		for i := range columns {
			if i < len(values) {
				row = append(row, formatAny(values[i]))
			} else {
				row = append(row, "")
			}
		}
		rows = append(rows, row)
	}
	return columns, rows
}

// rowsFromMapRows 把对象映射切片转换为表格列名与数据行。
//
// `items` 为映射切片；列名取首条记录的键并按字典序排列，后续记录缺键时对应单元格留空；
// 输入为空时返回 `nil` 列名与 `nil` 行。
func rowsFromMapRows(items []map[string]any) ([]string, [][]string) {
	if len(items) == 0 {
		return nil, nil
	}
	columns := make([]string, 0, len(items[0]))
	for column := range items[0] {
		columns = append(columns, column)
	}
	sort.Strings(columns)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := make([]string, 0, len(columns))
		for _, column := range columns {
			row = append(row, formatAny(item[column]))
		}
		rows = append(rows, row)
	}
	return columns, rows
}

// rowsFromMACBelongBoards 把标的所属板块条目转换为表格行。
//
// `items` 为归属板块条目切片，每行依次为板块类型、状态码、板块代码、板块名称、板块价格、
// 昨收；`Metric1` 至 `Metric3` 附加数值语义待确认，按原始浮点值展示。
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

// rowsFromMACSymbolBars 转换 MAC 商品 K 线条目为表格行。
//
// `items` 为待转换条目切片；返回行数与输入条目数一一对应，输入为空时返回非 `nil` 的空切片，每行列顺序由函数固定。
func rowsFromMACSymbolBars(items []proto.MACSymbolBar) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.DateTime.Format(time.DateTime),
			formatFloat(item.Open),
			formatFloat(item.High),
			formatFloat(item.Low),
			formatFloat(item.Close),
			formatFloat(item.Vol),
			formatFloat(item.Amount),
			formatFloat(item.FloatShares),
			formatFloat(item.Turnover),
		})
	}
	return rows
}

// rowsFromSampling 把缩略采样价格序列转换为序号、价格、昨收与涨跌额表格行。
//
// `preClose` 为昨收价；`prices` 为按时间顺序排列的采样价格切片，每行序号从 0 起。`preClose`
// 为 0（协议未提供昨收）时涨跌额列留空，否则涨跌额为当前价格减去昨收。
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

// parseExTableRows 解析扩展市场普通表格文本，返回四列键值行。
//
// `content` 为服务端返回的逗号分隔文本；逗号分隔条目，条目内部以竖线分隔，首段为键、第二段
// 为名称。键中含 `#` 时拆分为类别与代码两列，其余行类别列留空；空条目被跳过。
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

// parseExTableDetailRows 解析扩展市场详细表格文本，返回列名与数据行。
//
// `content` 为逗号分隔、条目内以竖线分隔的文本；按最长条目生成列名，首列固定为 `key`，其余
// 依次为 `c2`、`c3` 等；不足列数的行用空字符串补齐。文本为空时返回单列 `key` 与空行切片。
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

// normalizeTableRows 统一表格行列数并生成默认列名。
//
// `rows` 为原始数据行切片；`prefix` 为列名前缀。按最长行确定列数，列名依次为 `prefix_0`、
// `prefix_1` 等，短行用空字符串补齐；输入无数据时返回 `nil` 列名与 `nil` 行。
func normalizeTableRows(rows [][]string, prefix string) ([]string, [][]string) {
	maxColumns := 0
	for _, row := range rows {
		if len(row) > maxColumns {
			maxColumns = len(row)
		}
	}
	if maxColumns == 0 {
		return nil, nil
	}
	columns := make([]string, 0, maxColumns)
	for i := 0; i < maxColumns; i++ {
		columns = append(columns, fmt.Sprintf("%s_%d", prefix, i))
	}
	normalized := make([][]string, 0, len(rows))
	for _, row := range rows {
		item := make([]string, maxColumns)
		copy(item, row)
		normalized = append(normalized, item)
	}
	return columns, normalized
}

// preview 去除首尾空白后把文本截断到 `max` 字节以内，用于表格单元格与原始响应预览。
//
// `text` 为待预览文本；`max` 为允许保留的最大字节数，未超限时原样返回，超限时返回前 `max`
// 字节；按字节截断，可能落在多字节字符中间。
func preview(text string, max int) string {
	text = strings.TrimSpace(text)
	if len(text) <= max {
		return text
	}
	return text[:max]
}

// formatFloat 以两位小数的定点格式把 `float64` 数值格式化为表格字符串。
//
// `value` 为待格式化数值；不使用科学计数法，固定保留两位小数。
func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

// formatAny 把协议动态字段值转换为表格字符串。
//
// `value` 为动态字段原始值：`nil` 输出空字符串；`float64`、`float32` 按两位小数输出；各无符号
// 与有符号整型按十进制输出；字符串原样返回；其余类型走通用 `%v` 格式化。
func formatAny(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case float64:
		return formatFloat(v)
	case float32:
		return formatFloat(float64(v))
	case uint32:
		return fmt.Sprintf("%d", v)
	case uint16:
		return fmt.Sprintf("%d", v)
	case uint8:
		return fmt.Sprintf("%d", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatFloat32Ptr 格式化可选 `float32` 指针，固定保留四位小数。
//
// `value` 为待格式化指针；指针为 `nil`（协议未提供该值）时返回空字符串。
func formatFloat32Ptr(value *float32) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(float64(*value), 'f', 4, 64)
}

// minInt 返回两个整数中的较小值；`a`、`b` 为待比较整数。
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// lastInt 返回整数切片的最后一个元素；`values` 为空切片时返回 0。
func lastInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	return values[len(values)-1]
}

// ensureMethodsSorted 对全局方法定义原地排序，保证前端展示顺序稳定。
//
// 先按 `methodGroupRank` 给出的分组优先级升序排列，同组内按展示标签字典序排列，分组不同时再按
// 分组名兜底排序。
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

// methodGroupRank 返回分组的排序优先级编号。
//
// `group` 为方法分组名称；已登记分组返回固定优先级，编号越小越靠前；未登记分组返回 100，
// 排在所有已登记分组之后。
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

// init 在程序启动时先对方法定义排序，再重建方法索引表，保证运行期按名称查找的顺序与前端展示
// 顺序一致。
func init() {
	ensureMethodsSorted()
	methodMap = makeMethodMap(methodDefs)
}

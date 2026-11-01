package gotdx

import (
	"errors"
	"fmt"

	"github.com/bensema/gotdx/proto"
	"github.com/bensema/gotdx/types"
)

// 以下常量定义 MAC 协议各查询的默认获取条数与分页大小：`Count` 类常量为调用方未指定获取总数时
// 使用的默认上限，`Page` 类常量为自动分页时单页默认条数。
const (
	// DefaultMACBoardListCount 为 `MACBoardList` 未指定 `count` 时默认拉取的板块总数上限。
	DefaultMACBoardListCount = 10000
	// DefaultMACBoardPageSize 为板块列表单页默认条数（`GetMACBoardList` 的 `pageSize` 默认值）。
	DefaultMACBoardPageSize uint16 = 150
	// DefaultMACBoardStockCount 为板块成分股/成分报价未指定 `count` 时默认拉取的条目总数上限。
	DefaultMACBoardStockCount uint32 = 10000
	// DefaultMACBoardStockPage 为板块成分股/成分报价单页默认条数（`pageSize` 默认值）。
	DefaultMACBoardStockPage uint8 = 80
	// DefaultMACSymbolBarsCount 为 MAC 证券 K 线参考默认请求条数；当前自动分页按 `DefaultMACSymbolBarsPage` 逐页拉取。
	DefaultMACSymbolBarsCount uint32 = 800
	// DefaultMACSymbolBarsPage 为 MAC 证券 K 线自动分页的单页默认条数。
	DefaultMACSymbolBarsPage uint16 = 700
	// DefaultMACTransactionsCount 为分时成交未指定 `count` 时默认拉取的成交记录上限。
	DefaultMACTransactionsCount uint32 = 1000
	// DefaultMACTransactionsPage 为分时成交自动分页的单页默认条数。
	DefaultMACTransactionsPage uint16 = 1000
	// DefaultMACAuctionCount 为竞价数据未指定 `count` 时默认拉取的竞价记录上限。
	DefaultMACAuctionCount uint32 = 500
	// DefaultMACAuctionPage 为竞价数据自动分页的单页默认条数。
	DefaultMACAuctionPage uint32 = 500
	// DefaultMACMarketMonitorCount 为市场监控未指定 `count` 时默认拉取的异动记录上限。
	DefaultMACMarketMonitorCount uint32 = 600
	// DefaultMACMarketMonitorPage 为市场监控自动分页的单页默认条数。
	DefaultMACMarketMonitorPage uint16 = 600
	// DefaultMACKLineOffsetCount 为偏移 K 线查询未指定 `count` 时使用的默认条数。
	DefaultMACKLineOffsetCount uint32 = 128000
	// defaultMACBoardSortType 为成分股/成分报价默认排序字段编号，14 表示按涨跌幅排序。
	defaultMACBoardSortType uint16 = 14
	// defaultMACBoardSortOrder 为成分股/成分报价默认排序方向编号，1 表示降序。
	defaultMACBoardSortOrder uint16 = 1
)

// maxMACMarketMonitorStart 为市场监控协议 `uint16` 起始偏移的上界（65535），超过该值在翻页前先行拦截。
const maxMACMarketMonitorStart = uint32(^uint16(0))

// ConnectMAC 连接 MAC 行情主站。
//
// 本方法无业务入参；已建立连接池时直接复用现有池；方法加锁避免并发重复拨号。MAC 站无握手协议，
// 拨号即视为连接成功；全部候选地址拨号失败返回最后一次错误，成功返回 `nil`。
func (client *Client) ConnectMAC() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.closed {
		return errors.New("client closed")
	}
	// 幂等：已有池直接复用。
	if len(client.pool) > 0 {
		return nil
	}
	_, err := client.buildPoolLocked()
	return err
}

// GetMACBoardCount 查询指定类型板块的总数（底层协议接口）。
//
// 参数 `boardType` 为板块类型代码，取值见 `types.BoardType*` 常量；返回板块总数应答。连接或协议
// 收发、解析失败时返回 `nil` 与对应错误。
func (client *Client) GetMACBoardCount(boardType uint16) (*proto.MACBoardCountReply, error) {
	obj := proto.NewMACBoardCount(&proto.MACBoardListRequest{BoardType: boardType})
	return client.execute(obj)
}

// GetMACBoardList 分页查询指定类型的板块列表（底层协议接口）。
//
// 参数 `boardType` 为板块类型代码，取值见 `types.BoardType*`；参数 `start` 为起始板块序号（分页偏移），
// 从 0 开始；参数 `pageSize` 为本页请求条数。返回本页板块列表应答；连接或协议失败时返回 `nil` 与
// 对应错误。
func (client *Client) GetMACBoardList(boardType uint16, start uint16, pageSize uint16) (*proto.MACBoardListReply, error) {
	obj := proto.NewMACBoardList(&proto.MACBoardListRequest{
		BoardType: boardType,
		Start:     start,
		PageSize:  pageSize,
	})
	return client.execute(obj)
}

// GetMACBoardMembers 分页查询指定板块的成分股列表（底层协议接口）。
//
// 参数 `boardSymbol` 为板块符号，内部经协议映射为板块代码；参数 `sortType` 为排序字段编号，取值见
// 根包 `MACSortType*` 常量；参数 `start` 为起始成分股序号（分页偏移），从 0 开始；参数 `pageSize` 为
// 本页条数；参数 `sortOrder` 为排序方向编号（1 降序、2 升序、0 不指定）。板块符号映射失败或协议
// 失败时返回 `nil` 与对应错误。
func (client *Client) GetMACBoardMembers(boardSymbol string, sortType uint16, start uint32, pageSize uint8, sortOrder uint16) (*proto.MACBoardMembersReply, error) {
	boardCode, err := protoExchangeBoardCode(boardSymbol)
	if err != nil {
		return nil, err
	}
	obj := proto.NewMACBoardMembers(&proto.MACBoardMembersRequest{
		BoardCode: boardCode,
		SortType:  sortType,
		Start:     start,
		PageSize:  pageSize,
		SortOrder: sortOrder,
	})
	return client.execute(obj)
}

// GetMACBoardMembersQuotes 分页查询指定板块成分股的行情报价（底层协议接口）。
//
// 参数 `boardSymbol` 为板块符号，内部映射为板块代码；参数 `sortType` 为排序字段编号，取值见根包
// `MACSortType*`；参数 `start` 为起始成分股序号（分页偏移），从 0 开始；参数 `pageSize` 为本页条数；
// 参数 `sortOrder` 为排序方向编号（1 降序、2 升序、0 不指定）。板块符号映射失败或协议失败时返回
// `nil` 与对应错误。
func (client *Client) GetMACBoardMembersQuotes(boardSymbol string, sortType uint16, start uint32, pageSize uint8, sortOrder uint8) (*proto.MACBoardMembersQuotesReply, error) {
	boardCode, err := protoExchangeBoardCode(boardSymbol)
	if err != nil {
		return nil, err
	}
	obj := proto.NewMACBoardMembersQuotes(&proto.MACBoardMembersQuotesRequest{
		BoardCode: boardCode,
		SortType:  sortType,
		Start:     start,
		PageSize:  pageSize,
		SortOrder: sortOrder,
	})
	return client.execute(obj)
}

// GetMACBoardMembersQuotesDynamic 获取按位图动态解析的 MAC 板块成分报价。
//
// 参数 `boardSymbol` 为板块符号，内部映射为板块代码；`sortType` 为排序字段编号（见根包
// `MACSortType*`）；`start` 为起始成分股序号，从 0 开始；`pageSize` 为本页条数；`sortOrder` 为排序
// 方向编号（1 降序、2 升序、0 不指定）；`fieldBitmap` 为 20 字节（160 位）字段位图，声明本次返回
// 哪些动态行情字段，传全零时使用内置默认位图。过滤条件固定为 0（不排除）。板块符号映射失败或协议
// 失败时返回 `nil` 与对应错误。
func (client *Client) GetMACBoardMembersQuotesDynamic(boardSymbol string, sortType uint16, start uint32, pageSize uint8, sortOrder uint8, fieldBitmap [20]byte) (*proto.MACBoardMembersQuotesDynamicReply, error) {
	return client.GetMACBoardMembersQuotesDynamicWithFilter(boardSymbol, sortType, start, pageSize, sortOrder, fieldBitmap, 0)
}

// GetMACBoardMembersQuotesDynamicWithFilter 获取按位图动态解析的 MAC 板块成分报价，并设置排除过滤位。
//
// 除 `filter` 外其余参数语义同 `GetMACBoardMembersQuotesDynamic`；参数 `filter` 为排除过滤位组合，
// 0 表示不排除，各过滤位编号见根包 `MACFilterType`。板块符号映射失败或协议失败时返回 `nil` 与
// 对应错误。
func (client *Client) GetMACBoardMembersQuotesDynamicWithFilter(boardSymbol string, sortType uint16, start uint32, pageSize uint8, sortOrder uint8, fieldBitmap [20]byte, filter uint8) (*proto.MACBoardMembersQuotesDynamicReply, error) {
	boardCode, err := protoExchangeBoardCode(boardSymbol)
	if err != nil {
		return nil, err
	}
	obj := proto.NewMACBoardMembersQuotesDynamic(&proto.MACBoardMembersQuotesDynamicRequest{
		BoardCode:   boardCode,
		SortType:    sortType,
		Start:       start,
		PageSize:    pageSize,
		SortOrder:   sortOrder,
		Filter:      filter,
		FieldBitmap: fieldBitmap,
	})
	return client.execute(obj)
}

// GetMACQuotes 获取 MAC 单只标的最新交易日的快照与分时采样。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码。等价于
// `GetMACQuotesWithDate(market, code, 0)`，即查询最新交易日；连接或协议失败时返回 `nil` 与对应错误。
func (client *Client) GetMACQuotes(market uint8, code string) (*proto.MACQuotesReply, error) {
	return client.GetMACQuotesWithDate(market, code, 0)
}

// GetMACSymbolQuotes 获取 MAC 按位图批量股票报价。
//
// 参数 `markets` 为市场代码序列，取值见 `types.Market*`；参数 `codes` 为证券代码序列，与 `markets`
// 按下标一一对应；参数 `fieldBitmap` 为 20 字节（160 位）字段位图，声明每只标的返回哪些动态字段，
// 传全零时使用内置默认位图。两个列表长度不一致时返回 `ErrMarketCodeCount` 错误；连接或协议失败时
// 返回 `nil` 与对应错误。
func (client *Client) GetMACSymbolQuotes(markets []uint8, codes []string, fieldBitmap [20]byte) (*proto.MACSymbolQuotesReply, error) {
	if len(markets) != len(codes) {
		return nil, ErrMarketCodeCount
	}
	stocks := make([]proto.MACSymbolQuoteStock, 0, len(markets))
	for i, market := range markets {
		stocks = append(stocks, proto.MACSymbolQuoteStock{
			Market: uint16(market),
			Code:   makeMACCode22Client(codes[i]),
		})
	}
	obj := proto.NewMACSymbolQuotes(&proto.MACSymbolQuotesRequest{
		FieldBitmap: fieldBitmap,
		Stocks:      stocks,
	})
	return client.execute(obj)
}

// GetMACQuotesWithDate 获取 MAC 单只标的快照与分时采样，并可指定查询日期。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码；参数 `queryDate` 为查询
// 交易日，按 `YYYYMMDD` 打包成整数，传 0 表示取最新交易日。返回快照与分时采样应答；协议失败时
// 返回 `nil` 与对应错误。
func (client *Client) GetMACQuotesWithDate(market uint8, code string, queryDate uint32) (*proto.MACQuotesReply, error) {
	req := &proto.MACQuotesRequest{
		Market: uint16(market),
		Code:   makeMACCode22Client(code),
	}
	if queryDate != 0 {
		// 日期整数拆成低 16 位与高 16 位两个字段填入协议定长结构。
		req.Zero1 = uint16(queryDate & 0xffff)
		req.Zero2 = uint16(queryDate >> 16)
	}
	obj := proto.NewMACQuotes(req)
	return client.execute(obj)
}

// GetMACTransactions 获取 MAC 最新交易日的分时成交明细。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码；参数 `start` 为起始成交
// 序号（分页偏移），从 0 开始；参数 `count` 为请求成交条数。等价于
// `GetMACTransactionsWithDate(market, code, start, count, 0)`，即查询最新交易日；连接或协议失败时返回
// `nil` 与对应错误。
func (client *Client) GetMACTransactions(market uint8, code string, start uint32, count uint16) (*proto.MACTransactionsReply, error) {
	return client.GetMACTransactionsWithDate(market, code, start, count, 0)
}

// GetMACFileList 获取 MAC 文件列表/元信息（大小、状态标志与哈希）。
//
// 参数 `filename` 为协议文件名，内部按定长 70 字节字段填充；参数 `offset` 为协议起始偏移，口径
// 语义待确认。返回文件元信息应答；协议失败时返回 `nil` 与对应错误。
func (client *Client) GetMACFileList(filename string, offset uint32) (*proto.MACFileListReply, error) {
	obj := proto.NewMACFileList(&proto.MACFileListRequest{
		Offset:   offset,
		Filename: makeMACFilename70(filename),
	})
	return client.execute(obj)
}

// GetMACFileDownload 下载 MAC 文件片段。
//
// 参数 `filename` 为协议文件名，内部按定长 70 字节字段填充；参数 `index` 为分片序号，协议从 1 开始；
// 参数 `offset` 为本片段字节偏移；参数 `size` 为本片段请求字节数，传 0 时由协议构造器使用默认大小。
// 返回分片序号、本分片大小与文件原始字节；协议失败时返回 `nil` 与对应错误。
func (client *Client) GetMACFileDownload(filename string, index uint32, offset uint32, size uint32) (*proto.MACFileDownloadReply, error) {
	obj := proto.NewMACFileDownload(&proto.MACFileDownloadRequest{
		Index:    index,
		Offset:   offset,
		Size:     size,
		Filename: makeMACFilename70(filename),
	})
	return client.execute(obj)
}

// GetMACCapitalFlow 获取 MAC 个股资金流向（当日与五日资金指标）。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码。返回资金流向应答；协议
// 失败时返回 `nil` 与对应错误。
func (client *Client) GetMACCapitalFlow(market uint8, code string) (*proto.MACCapitalFlowReply, error) {
	obj := proto.NewMACCapitalFlow(&proto.MACCapitalFlowRequest{
		Market: uint16(market),
		Symbol: makeMACCode8Client(code),
	})
	return client.execute(obj)
}

// GetMACServerInfo 获取 MAC 服务端交易日时段与状态信息，无业务入参；返回服务端状态应答，协议
// 失败时返回 `nil` 与对应错误。
func (client *Client) GetMACServerInfo() (*proto.MACServerInfoReply, error) {
	obj := proto.NewMACServerInfo(nil)
	return client.execute(obj)
}

// GetMACKLineOffset 获取 MAC K 线偏移信息（K 线总数与本次返回条数）。
//
// 参数 `offset` 为起始 K 线条数偏移，从 0 开始；参数 `count` 为期望查询条数，传 0 时由协议构造器
// 使用默认值 128000。返回偏移信息应答；协议失败时返回 `nil` 与对应错误。
func (client *Client) GetMACKLineOffset(offset uint32, count uint32) (*proto.MACKLineOffsetReply, error) {
	obj := proto.NewMACKLineOffset(&proto.MACKLineOffsetRequest{
		Offset: offset,
		Count:  count,
	})
	return client.execute(obj)
}

// GetMACTransactionsWithDate 获取 MAC 分时成交，并可指定查询日期。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码；参数 `queryDate` 为查询
// 交易日，按 `YYYYMMDD` 打包，传 0 表示最新交易日；参数 `start` 为起始成交序号（分页偏移），从 0
// 开始；参数 `count` 为请求成交条数。返回成交明细应答；协议失败时返回 `nil` 与对应错误。
func (client *Client) GetMACTransactionsWithDate(market uint8, code string, start uint32, count uint16, queryDate uint32) (*proto.MACTransactionsReply, error) {
	obj := proto.NewMACTransactions(&proto.MACTransactionsRequest{
		Market:    uint16(market),
		Code:      makeMACCode22Client(code),
		QueryDate: queryDate,
		Start:     start,
		Count:     count,
	})
	return client.execute(obj)
}

// GetMACAuction 获取 MAC 集合竞价数据。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码；参数 `start` 为起始竞价
// 记录序号（分页偏移），从 0 开始；参数 `count` 为请求条数。返回竞价快照应答；协议失败时返回
// `nil` 与对应错误。
func (client *Client) GetMACAuction(market uint8, code string, start uint32, count uint32) (*proto.MACAuctionReply, error) {
	obj := proto.NewMACAuction(&proto.MACAuctionRequest{
		Market: uint16(market),
		Code:   makeMACCode22Client(code),
		Start:  start,
		Count:  count,
	})
	return client.execute(obj)
}

// GetMACTickCharts 获取 MAC 多日分时数据。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码；参数 `queryDate` 为查询
// 基准交易日，按 `YYYYMMDD` 打包成整数，传 0 表示取最新交易日；参数 `days` 为需要返回的交易日天数，
// 从基准日向前取数，传 0 时由协议构造器使用默认值 5，服务端返回的多余交易日会按该值裁剪。返回
// `*proto.MACTickChartsReply`，内含标的标识、交易日计数、逐日分时序列与汇总行情；连接或协议收发、
// 解析失败时返回 `nil` 与对应错误。
func (client *Client) GetMACTickCharts(market uint8, code string, queryDate uint32, days uint16) (*proto.MACTickChartsReply, error) {
	obj := proto.NewMACTickCharts(&proto.MACTickChartsRequest{
		Market:    uint16(market),
		Code:      makeMACCode22Client(code),
		QueryDate: queryDate,
		Days:      days,
	})
	return client.execute(obj)
}

// GetMACSymbolInfo 获取 MAC 个股摘要（名称、价格、股本等静态与快照字段）。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码。返回个股摘要应答；协议
// 失败时返回 `nil` 与对应错误。
func (client *Client) GetMACSymbolInfo(market uint8, code string) (*proto.MACSymbolInfoReply, error) {
	obj := proto.NewMACSymbolInfo(&proto.MACSymbolInfoRequest{
		Market: uint16(market),
		Code:   makeMACCode22Client(code),
	})
	return client.execute(obj)
}

// GetMACMarketMonitor 获取 MAC 市场监控（异动）数据。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `start` 为起始记录序号（分页偏移），从 0
// 开始；参数 `count` 为请求异动条数。返回异动记录应答；协议失败时返回 `nil` 与对应错误。
func (client *Client) GetMACMarketMonitor(market uint8, start uint16, count uint16) (*proto.MACMarketMonitorReply, error) {
	obj := proto.NewMACMarketMonitor(&proto.MACMarketMonitorRequest{
		Market: uint16(market),
		Start:  start,
		Count:  count,
	})
	return client.execute(obj)
}

// GetMACSymbolBelongBoard 查询指定证券所属的板块列表（底层协议接口）。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `symbol` 为证券代码。返回所属板块条目应答；
// 协议失败时返回 `nil` 与对应错误。
func (client *Client) GetMACSymbolBelongBoard(market uint8, symbol string) (*proto.MACSymbolBelongBoardReply, error) {
	obj := proto.NewMACSymbolBelongBoard(&proto.MACSymbolBelongBoardRequest{
		Market: uint16(market),
		Symbol: makeMACCode8Client(symbol),
	})
	return client.execute(obj)
}

// GetMACSymbolBars 查询 MAC 证券 K 线数据（底层协议接口）。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码；参数 `period` 为 K 线周期
// 类别，取值见 `types.KLINE_TYPE_*`；参数 `times` 为周期倍数，传 0 时按 1 处理；参数 `start` 为起始
// 偏移条数，从 0 开始；参数 `count` 为请求条数；参数 `adjust` 为复权方式，取值为 `types.AdjustNone`、
// `types.AdjustQFQ`、`types.AdjustHFQ`。返回 K 线应答；协议失败时返回 `nil` 与对应错误。
func (client *Client) GetMACSymbolBars(market uint8, code string, period uint16, times uint16, start uint32, count uint16, adjust uint16) (*proto.MACSymbolBarsReply, error) {
	obj := proto.NewMACSymbolBars(&proto.MACSymbolBarsRequest{
		Market: uint16(market),
		Code:   makeMACCode22Client(code),
		Period: period,
		Times:  times,
		Start:  start,
		Count:  count,
		Adjust: adjust,
	})
	return client.execute(obj)
}

// MACBoardCount 连接 MAC 主站并返回指定类型板块的总数。
//
// 参数 `boardType` 为板块类型代码，取值见 `types.BoardType*`；返回服务端板块总数。连接失败或协议
// 失败时返回 0 与对应错误。
func (client *Client) MACBoardCount(boardType uint16) (uint16, error) {
	if err := client.ConnectMAC(); err != nil {
		return 0, err
	}
	reply, err := client.GetMACBoardCount(boardType)
	if err != nil {
		return 0, err
	}
	return reply.Total, nil
}

// MACBoardList 连接 MAC 主站并自动分页拉取板块列表。
//
// 参数 `boardType` 为板块类型代码，取值见 `types.BoardType*`；参数 `count` 为期望获取的板块总数上限，
// 传 0 时使用默认上限 `DefaultMACBoardListCount`；分页单页大小取 `DefaultMACBoardPageSize`，末页
// 自动收缩。返回合并后的板块条目列表；连接或任一页协议失败时返回 `nil` 与对应错误。
func (client *Client) MACBoardList(boardType uint16, count uint32) ([]proto.MACBoardListItem, error) {
	if count == 0 {
		count = DefaultMACBoardListCount
	}
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}

	result := make([]proto.MACBoardListItem, 0)
	for start := uint32(0); start < count; start += uint32(DefaultMACBoardPageSize) {
		pageSize := DefaultMACBoardPageSize
		remaining := count - start
		// 最后一页不足默认大小时把本页大小收缩到剩余条数，避免超取。
		if remaining < uint32(pageSize) {
			pageSize = uint16(remaining)
		}
		reply, err := client.GetMACBoardList(boardType, uint16(start), pageSize)
		if err != nil {
			return nil, err
		}
		// 空页或不足一页均视为已到末尾，停止翻页。
		if len(reply.List) == 0 {
			break
		}
		result = append(result, reply.List...)
		if uint32(len(reply.List)) < uint32(pageSize) {
			break
		}
	}
	return result, nil
}

// MACBoardMembers 连接 MAC 主站并自动分页拉取板块成分股（使用默认排序）。
//
// 参数 `boardSymbol` 为板块符号；参数 `count` 为期望获取的成分股总数上限，传 0 时使用默认上限
// `DefaultMACBoardStockCount`。等价于以默认排序字段与方向调用 `MACBoardMembersWithSort`；连接或协议
// 失败时返回 `nil` 与对应错误。
func (client *Client) MACBoardMembers(boardSymbol string, count uint32) ([]proto.MACBoardMemberItem, error) {
	return client.MACBoardMembersWithSort(boardSymbol, count, defaultMACBoardSortType, defaultMACBoardSortOrder)
}

// MACBoardMembersWithSort 连接 MAC 主站并自动分页拉取板块成分股，透传排序参数。
//
// 参数 `boardSymbol` 为板块符号，内部映射为板块代码；参数 `count` 为期望获取的成分股总数上限，传 0
// 时使用 `DefaultMACBoardStockCount`；参数 `sortType` 为排序字段编号，取值见根包 `MACSortType*`；
// 参数 `sortOrder` 为排序方向编号（1 降序、2 升序、0 不指定）。分页单页大小取
// `DefaultMACBoardStockPage`，末页自动收缩；返回合并后的成分股列表，连接或协议失败时返回 `nil` 与
// 对应错误。
func (client *Client) MACBoardMembersWithSort(boardSymbol string, count uint32, sortType uint16, sortOrder uint16) ([]proto.MACBoardMemberItem, error) {
	if count == 0 {
		count = DefaultMACBoardStockCount
	}
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}

	result := make([]proto.MACBoardMemberItem, 0)
	for start := uint32(0); start < count; start += uint32(DefaultMACBoardStockPage) {
		pageSize := DefaultMACBoardStockPage
		remaining := count - start
		if remaining < uint32(pageSize) {
			pageSize = uint8(remaining)
		}
		reply, err := client.GetMACBoardMembers(boardSymbol, sortType, start, pageSize, sortOrder)
		if err != nil {
			return nil, err
		}
		if len(reply.Stocks) == 0 {
			break
		}
		result = append(result, reply.Stocks...)
		if uint32(len(reply.Stocks)) < uint32(pageSize) {
			break
		}
	}
	return result, nil
}

// MACBoardMembersQuotes 连接 MAC 主站并自动分页拉取板块成分股行情报价（使用默认排序）。
//
// 参数 `boardSymbol` 为板块符号；参数 `count` 为期望获取的报价条目总数上限，传 0 时使用默认上限
// `DefaultMACBoardStockCount`。等价于以默认排序字段与方向调用 `MACBoardMembersQuotesWithSort`；连接或
// 协议失败时返回 `nil` 与对应错误。
func (client *Client) MACBoardMembersQuotes(boardSymbol string, count uint32) ([]proto.MACBoardMemberQuoteItem, error) {
	return client.MACBoardMembersQuotesWithSort(boardSymbol, count, defaultMACBoardSortType, uint8(defaultMACBoardSortOrder))
}

// MACBoardMembersQuotesDynamic 连接 MAC 主站并自动分页拉取按位图动态解析的板块成分报价。
//
// 参数 `boardSymbol` 为板块符号；参数 `count` 为期望获取的条目总数上限，传 0 时使用
// `DefaultMACBoardStockCount`；参数 `sortType` 为排序字段编号；参数 `sortOrder` 为排序方向编号；参数
// `fieldBitmap` 为 20 字节字段位图，声明返回哪些动态字段。等价于不带排除过滤条件调用
// `MACBoardMembersQuotesDynamicWithFilter`；连接或协议失败时返回 `nil` 与对应错误。
func (client *Client) MACBoardMembersQuotesDynamic(boardSymbol string, count uint32, sortType uint16, sortOrder uint8, fieldBitmap [20]byte) (*proto.MACBoardMembersQuotesDynamicReply, error) {
	return client.MACBoardMembersQuotesDynamicWithFilter(boardSymbol, count, sortType, sortOrder, fieldBitmap)
}

// MACBoardMembersQuotesDynamicWithFilter 连接 MAC 主站并自动分页拉取按位图动态解析的板块成分报价，
// 并支持排除过滤条件。
//
// 参数 `boardSymbol` 为板块符号；参数 `count` 为期望获取的条目总数上限，传 0 时使用
// `DefaultMACBoardStockCount`；参数 `sortType` 为排序字段编号，取值见根包 `MACSortType*`；参数
// `sortOrder` 为排序方向编号；参数 `fieldBitmap` 为 20 字节字段位图，跨页必须保持一致；参数 `filters`
// 为可变排除过滤位，由 `MACFilterFlags` 合并为单个过滤字节。跨页位图不一致属于协议异常，直接返回
// 错误；一页都未取到时返回使用默认位图的空应答。
func (client *Client) MACBoardMembersQuotesDynamicWithFilter(boardSymbol string, count uint32, sortType uint16, sortOrder uint8, fieldBitmap [20]byte, filters ...MACFilterType) (*proto.MACBoardMembersQuotesDynamicReply, error) {
	if count == 0 {
		count = DefaultMACBoardStockCount
	}
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}

	// 跨页合并动态报价；各页必须使用同一份字段位图，否则字段含义会错位。
	var merged *proto.MACBoardMembersQuotesDynamicReply
	for start := uint32(0); start < count; start += uint32(DefaultMACBoardStockPage) {
		pageSize := DefaultMACBoardStockPage
		remaining := count - start
		if remaining < uint32(pageSize) {
			pageSize = uint8(remaining)
		}
		reply, err := client.GetMACBoardMembersQuotesDynamicWithFilter(boardSymbol, sortType, start, pageSize, sortOrder, fieldBitmap, MACFilterFlags(filters...))
		if err != nil {
			return nil, err
		}
		if len(reply.Stocks) == 0 {
			break
		}
		if merged == nil {
			// 首页深拷贝应答，`Stocks` 单独分配切片，避免共享底层数组。
			copyReply := *reply
			copyReply.Stocks = append([]proto.MACBoardMemberQuoteDynamicItem(nil), reply.Stocks...)
			copyReply.Count = uint16(len(copyReply.Stocks))
			merged = &copyReply
		} else {
			// 位图跨页不一致属于协议异常，直接返回错误防止按错误字段解析。
			if merged.FieldBitmap != reply.FieldBitmap {
				return nil, fmt.Errorf("mac dynamic field bitmap changed across pages: %x != %x", merged.FieldBitmap, reply.FieldBitmap)
			}
			merged.Stocks = append(merged.Stocks, reply.Stocks...)
			merged.Count = uint16(len(merged.Stocks))
			if reply.Total > merged.Total {
				merged.Total = reply.Total
			}
		}
		if uint32(len(reply.Stocks)) < uint32(pageSize) {
			break
		}
	}
	// 一页都没取到时构造空应答；空位图回退到默认字段位图，保证 ActiveFields 可渲染。
	if merged == nil {
		merged = &proto.MACBoardMembersQuotesDynamicReply{
			FieldBitmap: fieldBitmap,
		}
		if merged.FieldBitmap == ([20]byte{}) {
			merged.FieldBitmap = DefaultMACBoardMembersQuotesFieldBitmap()
		}
		merged.ActiveFields = []proto.MACDynamicFieldDef{}
	}
	return merged, nil
}

// MACBoardMembersQuotesWithSort 连接 MAC 主站并自动分页拉取板块成分报价，透传排序参数。
//
// 参数 `boardSymbol` 为板块符号；参数 `count` 为期望获取的报价条目总数上限，传 0 时使用
// `DefaultMACBoardStockCount`；参数 `sortType` 为排序字段编号，取值见根包 `MACSortType*`；参数
// `sortOrder` 为排序方向编号（1 降序、2 升序、0 不指定）。分页单页大小取 `DefaultMACBoardStockPage`，
// 末页自动收缩；返回合并后的报价列表，连接或协议失败时返回 `nil` 与对应错误。
func (client *Client) MACBoardMembersQuotesWithSort(boardSymbol string, count uint32, sortType uint16, sortOrder uint8) ([]proto.MACBoardMemberQuoteItem, error) {
	if count == 0 {
		count = DefaultMACBoardStockCount
	}
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}

	result := make([]proto.MACBoardMemberQuoteItem, 0)
	for start := uint32(0); start < count; start += uint32(DefaultMACBoardStockPage) {
		pageSize := DefaultMACBoardStockPage
		remaining := count - start
		if remaining < uint32(pageSize) {
			pageSize = uint8(remaining)
		}
		reply, err := client.GetMACBoardMembersQuotes(boardSymbol, sortType, start, pageSize, sortOrder)
		if err != nil {
			return nil, err
		}
		if len(reply.Stocks) == 0 {
			break
		}
		result = append(result, reply.Stocks...)
		if uint32(len(reply.Stocks)) < uint32(pageSize) {
			break
		}
	}
	return result, nil
}

// MACSymbolBelongBoard 连接 MAC 主站并查询指定证券所属的板块列表。
//
// 参数 `symbol` 为证券代码；参数 `market` 为市场代码，取值见 `types.Market*`。返回所属板块条目列表；
// 连接或协议失败时返回 `nil` 与对应错误。
func (client *Client) MACSymbolBelongBoard(symbol string, market uint8) ([]proto.MACBelongBoardItem, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	reply, err := client.GetMACSymbolBelongBoard(market, symbol)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

// MACQuotes 连接 MAC 主站并获取单只标的最新交易日的行情快照与分时采样。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码。等价于
// `MACQuotesWithDate(market, code, 0)`；连接或协议失败时返回 `nil` 与对应错误。
func (client *Client) MACQuotes(market uint8, code string) (*proto.MACQuotesReply, error) {
	return client.MACQuotesWithDate(market, code, 0)
}

// MACSymbolQuotes 连接 MAC 主站并按位图批量获取股票报价。
//
// 参数 `markets` 为市场代码序列，取值见 `types.Market*`；参数 `codes` 为证券代码序列，与 `markets`
// 按下标一一对应；参数 `fieldBitmap` 为 20 字节字段位图，声明每只标的返回哪些动态字段，传全零时
// 使用内置默认位图。两个列表长度不一致时返回 `ErrMarketCodeCount`；连接或协议失败时返回 `nil` 与
// 对应错误。
func (client *Client) MACSymbolQuotes(markets []uint8, codes []string, fieldBitmap [20]byte) (*proto.MACSymbolQuotesReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACSymbolQuotes(markets, codes, fieldBitmap)
}

// MACQuotesWithDate 连接 MAC 主站并获取单只标的行情快照与分时采样，可指定查询日期。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码；参数 `queryDate` 为查询
// 交易日，按 `YYYYMMDD` 打包，传 0 表示最新交易日。连接或协议失败时返回 `nil` 与对应错误。
func (client *Client) MACQuotesWithDate(market uint8, code string, queryDate uint32) (*proto.MACQuotesReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACQuotesWithDate(market, code, queryDate)
}

// MACTransactions 连接 MAC 主站并自动分页获取最新交易日的分时成交明细。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码；参数 `start` 为起始成交
// 序号（分页偏移），从 0 开始；参数 `count` 为期望成交条数上限，传 0 时使用默认上限
// `DefaultMACTransactionsCount`。等价于 `MACTransactionsWithDate(market, code, start, count, 0)`；
// 连接或协议失败时返回 `nil` 与对应错误。
func (client *Client) MACTransactions(market uint8, code string, start uint32, count uint32) ([]proto.MACTransactionItem, error) {
	return client.MACTransactionsWithDate(market, code, start, count, 0)
}

// MACFileList 连接 MAC 主站并获取 MAC 文件列表/元信息。
//
// 参数 `filename` 为协议文件名，内部按定长 70 字节字段填充；参数 `offset` 为协议起始偏移，口径语义
// 待确认。返回文件元信息应答；连接或协议失败时返回 `nil` 与对应错误。
func (client *Client) MACFileList(filename string, offset uint32) (*proto.MACFileListReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACFileList(filename, offset)
}

// MACDownloadFullFile 连接 MAC 主站并按固定块大小循环下载完整 MAC 文件。
//
// 参数 `filename` 为协议文件名；参数 `index` 为文件分片序号，协议从 1 开始，传 0 时归一化为 1；参数
// `size` 为目标总大小（字节），传 0 时先取文件元信息中的大小作为下载上限。服务器返回空块或已下载
// 字节达到 `size` 时结束；返回拼接后的完整文件字节，连接或分块协议失败时返回 `nil` 与对应错误。
func (client *Client) MACDownloadFullFile(filename string, index uint32, size uint32) ([]byte, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	// 协议文件序号从 1 开始，0 归一化为 1。
	if index == 0 {
		index = 1
	}
	// 未指定总大小时先取文件元信息中的大小作为下载上限。
	if size == 0 {
		meta, err := client.GetMACFileList(filename, 0)
		if err == nil {
			size = meta.Size
		}
	}

	var result []byte
	var downloaded uint32
	for {
		reply, err := client.GetMACFileDownload(filename, index, downloaded, types.DefaultDownloadSize)
		if err != nil {
			return nil, err
		}
		// 服务器返回空块表示文件已下载完。
		if reply.Size == 0 {
			break
		}
		result = append(result, reply.Data...)
		downloaded += reply.Size
		// 已达到目标总大小即停止；`size` 为 0 时仅靠空块结束循环。
		if size != 0 && downloaded >= size {
			break
		}
	}
	return result, nil
}

// MACCapitalFlow 连接 MAC 主站并获取个股资金流向。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码。连接或协议失败时返回
// `nil` 与对应错误。
func (client *Client) MACCapitalFlow(market uint8, code string) (*proto.MACCapitalFlowReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACCapitalFlow(market, code)
}

// MACServerInfo 连接 MAC 主站并获取服务端交易日时段与状态信息，无业务入参；连接或协议失败时返回
// `nil` 与对应错误。
func (client *Client) MACServerInfo() (*proto.MACServerInfoReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACServerInfo()
}

// MACKLineOffset 连接 MAC 主站并获取 K 线偏移信息（K 线总数与本次返回条数）。
//
// 参数 `offset` 为起始 K 线条数偏移，从 0 开始；参数 `count` 为期望查询条数，传 0 时使用默认值
// `DefaultMACKLineOffsetCount`。连接或协议失败时返回 `nil` 与对应错误。
func (client *Client) MACKLineOffset(offset uint32, count uint32) (*proto.MACKLineOffsetReply, error) {
	if count == 0 {
		count = DefaultMACKLineOffsetCount
	}
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACKLineOffset(offset, count)
}

// MACTransactionsWithDate 连接 MAC 主站并自动分页获取分时成交明细，可指定查询日期。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码；参数 `queryDate` 为查询
// 交易日，按 `YYYYMMDD` 打包，传 0 表示最新交易日；参数 `start` 为起始成交序号（分页偏移），从 0
// 开始；参数 `count` 为期望成交条数上限，传 0 时使用 `DefaultMACTransactionsCount`；分页单页大小取
// `DefaultMACTransactionsPage`，末页自动收缩。返回合并后的成交明细，连接或协议失败时返回 `nil` 与
// 对应错误。
func (client *Client) MACTransactionsWithDate(market uint8, code string, start uint32, count uint32, queryDate uint32) ([]proto.MACTransactionItem, error) {
	if count == 0 {
		count = DefaultMACTransactionsCount
	}
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}

	result := make([]proto.MACTransactionItem, 0)
	currentStart := start
	remaining := count
	// 按剩余条数翻页：最后一页收缩大小，空页或不足一页提前结束。
	for remaining > 0 {
		pageSize := DefaultMACTransactionsPage
		if remaining < uint32(pageSize) {
			pageSize = uint16(remaining)
		}
		reply, err := client.GetMACTransactionsWithDate(market, code, currentStart, pageSize, queryDate)
		if err != nil {
			return nil, err
		}
		if len(reply.List) == 0 {
			break
		}
		result = append(result, reply.List...)
		if uint32(len(reply.List)) < uint32(pageSize) {
			break
		}
		currentStart += uint32(pageSize)
		remaining -= uint32(pageSize)
	}
	return result, nil
}

// MACAuction 连接 MAC 主站并自动分页获取集合竞价数据。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码；参数 `start` 为起始竞价
// 记录序号（分页偏移），从 0 开始；参数 `count` 为期望竞价条数上限，传 0 时使用
// `DefaultMACAuctionCount`；分页单页大小取 `DefaultMACAuctionPage`，末页自动收缩。返回合并后的竞价
// 快照列表，连接或协议失败时返回 `nil` 与对应错误。
func (client *Client) MACAuction(market uint8, code string, start uint32, count uint32) ([]proto.MACAuctionItem, error) {
	if count == 0 {
		count = DefaultMACAuctionCount
	}
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}

	result := make([]proto.MACAuctionItem, 0)
	currentStart := start
	remaining := count
	// 按剩余条数翻页拉取竞价数据，空页或不足一页结束。
	for remaining > 0 {
		pageSize := DefaultMACAuctionPage
		if remaining < pageSize {
			pageSize = remaining
		}
		reply, err := client.GetMACAuction(market, code, currentStart, pageSize)
		if err != nil {
			return nil, err
		}
		if len(reply.List) == 0 {
			break
		}
		result = append(result, reply.List...)
		if uint32(len(reply.List)) < pageSize {
			break
		}
		currentStart += pageSize
		remaining -= pageSize
	}
	return result, nil
}

// MACTickCharts 连接 MAC 主站并获取多日分时数据。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码；参数 `queryDate` 为查询
// 基准交易日，按 `YYYYMMDD` 打包成整数，传 0 表示取最新交易日；参数 `days` 为需要返回的交易日天数，
// 从基准日向前取数，传 0 时由协议构造器使用默认值 5，服务端返回的多余交易日会按该值裁剪。返回
// `*proto.MACTickChartsReply`，内含标的标识、交易日计数、逐日分时序列与汇总行情；连接或协议失败时
// 返回 `nil` 与对应错误。
func (client *Client) MACTickCharts(market uint8, code string, queryDate uint32, days uint16) (*proto.MACTickChartsReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACTickCharts(market, code, queryDate, days)
}

// MACSymbolInfo 连接 MAC 主站并获取个股摘要。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码。连接或协议失败时返回
// `nil` 与对应错误。
func (client *Client) MACSymbolInfo(market uint8, code string) (*proto.MACSymbolInfoReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACSymbolInfo(market, code)
}

// MACMarketMonitor 连接 MAC 主站并自动分页获取市场监控（异动）数据。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `start` 为起始记录序号，从 0 开始，超过
// `uint16` 上界（65535）时直接返回错误；参数 `count` 为期望异动条数上限，传 0 时使用
// `DefaultMACMarketMonitorCount`；分页单页大小取 `DefaultMACMarketMonitorPage`，翻页过程中再次
// 校验偏移不越界。返回合并后的异动条目列表，连接或协议失败时返回 `nil` 与对应错误。
func (client *Client) MACMarketMonitor(market uint8, start uint32, count uint32) ([]proto.MACMarketMonitorItem, error) {
	if count == 0 {
		count = DefaultMACMarketMonitorCount
	}
	// 协议偏移为 `uint16`，入参超限在翻页前先拦截。
	if start > maxMACMarketMonitorStart {
		return nil, fmt.Errorf("mac market monitor start exceeds uint16: %d", start)
	}
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}

	result := make([]proto.MACMarketMonitorItem, 0)
	currentStart := start
	remaining := count
	for remaining > 0 {
		// 翻页过程中再次校验偏移，防止累加后越出 `uint16` 范围。
		if currentStart > maxMACMarketMonitorStart {
			return nil, fmt.Errorf("mac market monitor start exceeds uint16: %d", currentStart)
		}
		pageSize := DefaultMACMarketMonitorPage
		if remaining < uint32(pageSize) {
			pageSize = uint16(remaining)
		}
		reply, err := client.GetMACMarketMonitor(market, uint16(currentStart), pageSize)
		if err != nil {
			return nil, err
		}
		if len(reply.List) == 0 {
			break
		}
		result = append(result, reply.List...)
		if uint32(len(reply.List)) < uint32(pageSize) {
			break
		}
		currentStart += uint32(pageSize)
		remaining -= uint32(pageSize)
	}
	return result, nil
}

// MACSymbolBars 连接 MAC 主站、自动分页拉取证券 K 线，并补齐换手率、转换期货自然时间。
//
// 参数 `market` 为市场代码，取值见 `types.Market*`；参数 `code` 为证券代码；参数 `period` 为 K 线周期
// 类别，取值见 `types.KLINE_TYPE_*`；参数 `times` 为周期倍数，传 0 时按 1 处理；参数 `start` 为起始
// 偏移条数，从 0 开始；参数 `count` 为期望 K 线条数；参数 `adjust` 为复权方式，取值为
// `types.AdjustNone`、`types.AdjustQFQ`、`types.AdjustHFQ`。分页单页大小取 `DefaultMACSymbolBarsPage`，
// 末页自动收缩；合并后按流通股本回填换手率，并按配置的期货交易日历把期货 K 线时间换算为自然时间。
// 连接、协议或日历转换失败时返回 `nil` 与对应错误。
func (client *Client) MACSymbolBars(market uint8, code string, period uint16, times uint16, start uint32, count uint32, adjust uint16) ([]proto.MACSymbolBar, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}

	result := make([]proto.MACSymbolBar, 0)
	currentStart := start
	remaining := count
	for remaining > 0 {
		pageSize := DefaultMACSymbolBarsPage
		if remaining < uint32(pageSize) {
			pageSize = uint16(remaining)
		}
		reply, err := client.GetMACSymbolBars(market, code, period, times, currentStart, pageSize, adjust)
		if err != nil {
			return nil, err
		}
		if len(reply.List) == 0 {
			break
		}
		result = append(result, reply.List...)
		if uint32(len(reply.List)) < uint32(pageSize) {
			break
		}
		currentStart += uint32(pageSize)
		remaining -= uint32(pageSize)
	}
	// 分页合并后统一回填换手率，并把期货 K 线时间换算为自然时间。
	applyMACSymbolBarTurnover(result)
	if err := resolveMACFuturesDateTimes(result, code, client.opt.FuturesCalendarPath); err != nil {
		return nil, err
	}
	return result, nil
}

// applyMACSymbolBarTurnover 按流通股本回填 MAC K 线换手率。
//
// 参数 `items` 为待回填的 K 线条目序列；流通股本（万股口径换算）或成交量非正的条目跳过，不计算
// 换手率。
func applyMACSymbolBarTurnover(items []proto.MACSymbolBar) {
	for i := range items {
		if items[i].FloatShares <= 0 || items[i].Vol <= 0 {
			continue
		}
		items[i].Turnover = round2(items[i].Vol / (items[i].FloatShares * 10000) * 100)
	}
}

// makeMACCode8Client 把证券代码填充为协议定长 8 字节字段。
//
// 参数 `code` 为原始代码字符串；长度不足 8 字节时尾部补空字符，超过 8 字节时截断。
func makeMACCode8Client(code string) [8]byte {
	var out [8]byte
	copy(out[:], code)
	return out
}

// makeMACCode22Client 把证券代码填充为协议定长 22 字节字段。
//
// 参数 `code` 为原始代码字符串；长度不足 22 字节时尾部补空字符，超过 22 字节时截断。
func makeMACCode22Client(code string) [22]byte {
	var out [22]byte
	copy(out[:], code)
	return out
}

// makeMACFilename70 把协议文件名填充为定长 70 字节字段。
//
// 参数 `filename` 为原始文件名；长度不足 70 字节时尾部补空字符，超过 70 字节时截断。
func makeMACFilename70(filename string) [70]byte {
	var out [70]byte
	copy(out[:], filename)
	return out
}

// protoExchangeBoardCode 把板块符号映射为协议板块代码；无法识别的板块符号由底层映射函数返回错误。
func protoExchangeBoardCode(boardSymbol string) (uint32, error) {
	return proto.ExchangeMACBoardCode(boardSymbol)
}

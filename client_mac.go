package gotdx

import (
	"fmt"

	"github.com/bensema/gotdx/proto"
	"github.com/bensema/gotdx/types"
)

const (
	DefaultMACBoardListCount            = 10000
	DefaultMACBoardPageSize      uint16 = 150
	DefaultMACBoardStockCount    uint32 = 10000
	DefaultMACBoardStockPage     uint8  = 80
	DefaultMACSymbolBarsCount    uint32 = 800
	DefaultMACSymbolBarsPage     uint16 = 700
	DefaultMACTransactionsCount  uint32 = 1000
	DefaultMACTransactionsPage   uint16 = 1000
	DefaultMACAuctionCount       uint32 = 500
	DefaultMACAuctionPage        uint32 = 500
	DefaultMACMarketMonitorCount uint32 = 600
	DefaultMACMarketMonitorPage  uint16 = 600
	DefaultMACKLineOffsetCount   uint32 = 128000
	defaultMACBoardSortType      uint16 = 14
	defaultMACBoardSortOrder     uint16 = 1
)

const maxMACMarketMonitorStart = uint32(^uint16(0))

func (client *Client) ConnectMAC() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.conn != nil {
		return nil
	}
	return client.connect()
}

func (client *Client) GetMACBoardCount(boardType uint16) (*proto.MACBoardCountReply, error) {
	obj := proto.NewMACBoardCount(&proto.MACBoardListRequest{BoardType: boardType})
	return executeProtocol(client, obj)
}

func (client *Client) GetMACBoardList(boardType uint16, start uint16, pageSize uint16) (*proto.MACBoardListReply, error) {
	obj := proto.NewMACBoardList(&proto.MACBoardListRequest{
		BoardType: boardType,
		Start:     start,
		PageSize:  pageSize,
	})
	return executeProtocol(client, obj)
}

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
	return executeProtocol(client, obj)
}

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
	return executeProtocol(client, obj)
}

// GetMACBoardMembersQuotesDynamic 获取按位图动态解析的 MAC 板块成分报价。
func (client *Client) GetMACBoardMembersQuotesDynamic(boardSymbol string, sortType uint16, start uint32, pageSize uint8, sortOrder uint8, fieldBitmap [20]byte) (*proto.MACBoardMembersQuotesDynamicReply, error) {
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
		FieldBitmap: fieldBitmap,
	})
	return executeProtocol(client, obj)
}

// GetMACQuotes 获取 MAC 单只标的快照与分时采样。
func (client *Client) GetMACQuotes(market uint8, code string) (*proto.MACQuotesReply, error) {
	return client.GetMACQuotesWithDate(market, code, 0)
}

// GetMACSymbolQuotes 获取 MAC 按位图批量股票报价。
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
	return executeProtocol(client, obj)
}

// GetMACQuotesWithDate 获取 MAC 单只标的快照与分时采样，并可指定查询日期。
func (client *Client) GetMACQuotesWithDate(market uint8, code string, queryDate uint32) (*proto.MACQuotesReply, error) {
	req := &proto.MACQuotesRequest{
		Market: uint16(market),
		Code:   makeMACCode22Client(code),
	}
	if queryDate != 0 {
		req.Zero1 = uint16(queryDate & 0xffff)
		req.Zero2 = uint16(queryDate >> 16)
	}
	obj := proto.NewMACQuotes(req)
	return executeProtocol(client, obj)
}

// GetMACTransactions 获取 MAC 分时成交。
func (client *Client) GetMACTransactions(market uint8, code string, start uint32, count uint16) (*proto.MACTransactionsReply, error) {
	return client.GetMACTransactionsWithDate(market, code, start, count, 0)
}

// GetMACFileList 获取 MAC 文件列表/元信息。
func (client *Client) GetMACFileList(filename string, offset uint32) (*proto.MACFileListReply, error) {
	obj := proto.NewMACFileList(&proto.MACFileListRequest{
		Offset:   offset,
		Filename: makeMACFilename70(filename),
	})
	return executeProtocol(client, obj)
}

// GetMACFileDownload 下载 MAC 文件片段。
func (client *Client) GetMACFileDownload(filename string, index uint32, offset uint32, size uint32) (*proto.MACFileDownloadReply, error) {
	obj := proto.NewMACFileDownload(&proto.MACFileDownloadRequest{
		Index:    index,
		Offset:   offset,
		Size:     size,
		Filename: makeMACFilename70(filename),
	})
	return executeProtocol(client, obj)
}

// GetMACCapitalFlow 获取 MAC 资金流向。
func (client *Client) GetMACCapitalFlow(market uint8, code string) (*proto.MACCapitalFlowReply, error) {
	obj := proto.NewMACCapitalFlow(&proto.MACCapitalFlowRequest{
		Market: uint16(market),
		Symbol: makeMACCode8Client(code),
	})
	return executeProtocol(client, obj)
}

// GetMACServerInfo 获取 MAC 服务端交易日时段与状态信息。
func (client *Client) GetMACServerInfo() (*proto.MACServerInfoReply, error) {
	obj := proto.NewMACServerInfo(nil)
	return executeProtocol(client, obj)
}

// GetMACKLineOffset 获取 MAC K线偏移信息。
func (client *Client) GetMACKLineOffset(offset uint32, count uint32) (*proto.MACKLineOffsetReply, error) {
	obj := proto.NewMACKLineOffset(&proto.MACKLineOffsetRequest{
		Offset: offset,
		Count:  count,
	})
	return executeProtocol(client, obj)
}

// GetMACTransactionsWithDate 获取 MAC 分时成交，并可指定查询日期。
func (client *Client) GetMACTransactionsWithDate(market uint8, code string, start uint32, count uint16, queryDate uint32) (*proto.MACTransactionsReply, error) {
	obj := proto.NewMACTransactions(&proto.MACTransactionsRequest{
		Market:    uint16(market),
		Code:      makeMACCode22Client(code),
		QueryDate: queryDate,
		Start:     start,
		Count:     count,
	})
	return executeProtocol(client, obj)
}

// GetMACAuction 获取 MAC 竞价数据。
func (client *Client) GetMACAuction(market uint8, code string, start uint32, count uint32) (*proto.MACAuctionReply, error) {
	obj := proto.NewMACAuction(&proto.MACAuctionRequest{
		Market: uint16(market),
		Code:   makeMACCode22Client(code),
		Start:  start,
		Count:  count,
	})
	return executeProtocol(client, obj)
}

// GetMACTickCharts 获取 MAC 多日分时。
func (client *Client) GetMACTickCharts(market uint8, code string, queryDate uint32, days uint16) (*proto.MACTickChartsReply, error) {
	obj := proto.NewMACTickCharts(&proto.MACTickChartsRequest{
		Market:    uint16(market),
		Code:      makeMACCode22Client(code),
		QueryDate: queryDate,
		Days:      days,
	})
	return executeProtocol(client, obj)
}

// GetMACSymbolInfo 获取 MAC 股票摘要。
func (client *Client) GetMACSymbolInfo(market uint8, code string) (*proto.MACSymbolInfoReply, error) {
	obj := proto.NewMACSymbolInfo(&proto.MACSymbolInfoRequest{
		Market: uint16(market),
		Code:   makeMACCode22Client(code),
	})
	return executeProtocol(client, obj)
}

// GetMACMarketMonitor 获取 MAC 市场监控数据。
func (client *Client) GetMACMarketMonitor(market uint8, start uint16, count uint16) (*proto.MACMarketMonitorReply, error) {
	obj := proto.NewMACMarketMonitor(&proto.MACMarketMonitorRequest{
		Market: uint16(market),
		Start:  start,
		Count:  count,
	})
	return executeProtocol(client, obj)
}

func (client *Client) GetMACSymbolBelongBoard(market uint8, symbol string) (*proto.MACSymbolBelongBoardReply, error) {
	obj := proto.NewMACSymbolBelongBoard(&proto.MACSymbolBelongBoardRequest{
		Market: uint16(market),
		Symbol: makeMACCode8Client(symbol),
	})
	return executeProtocol(client, obj)
}

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
	return executeProtocol(client, obj)
}

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
		if remaining < uint32(pageSize) {
			pageSize = uint16(remaining)
		}
		reply, err := client.GetMACBoardList(boardType, uint16(start), pageSize)
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
	}
	return result, nil
}

func (client *Client) MACBoardMembers(boardSymbol string, count uint32) ([]proto.MACBoardMemberItem, error) {
	return client.MACBoardMembersWithSort(boardSymbol, count, defaultMACBoardSortType, defaultMACBoardSortOrder)
}

// MACBoardMembersWithSort 获取 MAC 板块成员，并透传排序参数。
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

func (client *Client) MACBoardMembersQuotes(boardSymbol string, count uint32) ([]proto.MACBoardMemberQuoteItem, error) {
	return client.MACBoardMembersQuotesWithSort(boardSymbol, count, defaultMACBoardSortType, uint8(defaultMACBoardSortOrder))
}

// MACBoardMembersQuotesDynamic 获取按位图动态解析的 MAC 板块成分报价。
func (client *Client) MACBoardMembersQuotesDynamic(boardSymbol string, count uint32, sortType uint16, sortOrder uint8, fieldBitmap [20]byte) (*proto.MACBoardMembersQuotesDynamicReply, error) {
	if count == 0 {
		count = DefaultMACBoardStockCount
	}
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}

	var merged *proto.MACBoardMembersQuotesDynamicReply
	for start := uint32(0); start < count; start += uint32(DefaultMACBoardStockPage) {
		pageSize := DefaultMACBoardStockPage
		remaining := count - start
		if remaining < uint32(pageSize) {
			pageSize = uint8(remaining)
		}
		reply, err := client.GetMACBoardMembersQuotesDynamic(boardSymbol, sortType, start, pageSize, sortOrder, fieldBitmap)
		if err != nil {
			return nil, err
		}
		if len(reply.Stocks) == 0 {
			break
		}
		if merged == nil {
			copyReply := *reply
			copyReply.Stocks = append([]proto.MACBoardMemberQuoteDynamicItem(nil), reply.Stocks...)
			copyReply.Count = uint16(len(copyReply.Stocks))
			merged = &copyReply
		} else {
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

// MACBoardMembersQuotesWithSort 获取 MAC 板块成分报价，并透传排序参数。
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

// MACQuotes 获取 MAC 行情快照与分时采样。
func (client *Client) MACQuotes(market uint8, code string) (*proto.MACQuotesReply, error) {
	return client.MACQuotesWithDate(market, code, 0)
}

// MACSymbolQuotes 获取 MAC 按位图批量股票报价。
func (client *Client) MACSymbolQuotes(markets []uint8, codes []string, fieldBitmap [20]byte) (*proto.MACSymbolQuotesReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACSymbolQuotes(markets, codes, fieldBitmap)
}

// MACQuotesWithDate 获取 MAC 行情快照与分时采样，并可指定查询日期。
func (client *Client) MACQuotesWithDate(market uint8, code string, queryDate uint32) (*proto.MACQuotesReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACQuotesWithDate(market, code, queryDate)
}

// MACTransactions 获取 MAC 分时成交。
func (client *Client) MACTransactions(market uint8, code string, start uint32, count uint32) ([]proto.MACTransactionItem, error) {
	return client.MACTransactionsWithDate(market, code, start, count, 0)
}

// MACFileList 获取 MAC 文件列表/元信息。
func (client *Client) MACFileList(filename string, offset uint32) (*proto.MACFileListReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACFileList(filename, offset)
}

// MACDownloadFullFile 下载完整 MAC 文件。
func (client *Client) MACDownloadFullFile(filename string, index uint32, size uint32) ([]byte, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	if index == 0 {
		index = 1
	}
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
		if reply.Size == 0 {
			break
		}
		result = append(result, reply.Data...)
		downloaded += reply.Size
		if size != 0 && downloaded >= size {
			break
		}
	}
	return result, nil
}

// MACCapitalFlow 获取 MAC 资金流向。
func (client *Client) MACCapitalFlow(market uint8, code string) (*proto.MACCapitalFlowReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACCapitalFlow(market, code)
}

// MACServerInfo 获取 MAC 服务端交易日时段与状态信息。
func (client *Client) MACServerInfo() (*proto.MACServerInfoReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACServerInfo()
}

// MACKLineOffset 获取 MAC K线偏移信息。
func (client *Client) MACKLineOffset(offset uint32, count uint32) (*proto.MACKLineOffsetReply, error) {
	if count == 0 {
		count = DefaultMACKLineOffsetCount
	}
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACKLineOffset(offset, count)
}

// MACTransactionsWithDate 获取 MAC 分时成交，并可指定查询日期。
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

// MACAuction 获取 MAC 竞价数据。
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

// MACTickCharts 获取 MAC 多日分时。
func (client *Client) MACTickCharts(market uint8, code string, queryDate uint32, days uint16) (*proto.MACTickChartsReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACTickCharts(market, code, queryDate, days)
}

// MACSymbolInfo 获取 MAC 股票摘要。
func (client *Client) MACSymbolInfo(market uint8, code string) (*proto.MACSymbolInfoReply, error) {
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACSymbolInfo(market, code)
}

// MACMarketMonitor 获取 MAC 市场监控数据，支持自动分页。
func (client *Client) MACMarketMonitor(market uint8, start uint32, count uint32) ([]proto.MACMarketMonitorItem, error) {
	if count == 0 {
		count = DefaultMACMarketMonitorCount
	}
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
	applyMACSymbolBarTurnover(result)
	return result, nil
}

func applyMACSymbolBarTurnover(items []proto.MACSymbolBar) {
	for i := range items {
		if items[i].FloatShares <= 0 || items[i].Vol <= 0 {
			continue
		}
		items[i].Turnover = round2(items[i].Vol / (items[i].FloatShares * 10000) * 100)
	}
}

func makeMACCode8Client(code string) [8]byte {
	var out [8]byte
	copy(out[:], code)
	return out
}

func makeMACCode22Client(code string) [22]byte {
	var out [22]byte
	copy(out[:], code)
	return out
}

func makeMACFilename70(filename string) [70]byte {
	var out [70]byte
	copy(out[:], filename)
	return out
}

func protoExchangeBoardCode(boardSymbol string) (uint32, error) {
	return proto.ExchangeMACBoardCode(boardSymbol)
}

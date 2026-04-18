package gotdx

import (
	"fmt"

	"github.com/bensema/gotdx/proto"
)

const (
	DefaultMACBoardListCount         = 10000
	DefaultMACBoardPageSize   uint16 = 150
	DefaultMACBoardStockCount uint32 = 10000
	DefaultMACBoardStockPage  uint8  = 80
	DefaultMACSymbolBarsCount uint32 = 800
	DefaultMACSymbolBarsPage  uint16 = 700
	defaultMACBoardSortType   uint16 = 14
	defaultMACBoardSortOrder  uint16 = 1
)

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
	obj := proto.NewMACQuotes(&proto.MACQuotesRequest{
		Market: uint16(market),
		Code:   makeMACCode22Client(code),
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
	if err := client.ConnectMAC(); err != nil {
		return nil, err
	}
	return client.GetMACQuotes(market, code)
}

func (client *Client) MACSymbolBars(market uint8, code string, period uint16, times uint16, start uint32, count uint32, adjust uint16) ([]proto.MACSymbolBar, error) {
	if count == 0 {
		count = DefaultMACSymbolBarsCount
	}
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
	return result, nil
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

func protoExchangeBoardCode(boardSymbol string) (uint32, error) {
	return proto.ExchangeMACBoardCode(boardSymbol)
}

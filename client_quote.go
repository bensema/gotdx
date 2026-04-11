package gotdx

import "github.com/bensema/gotdx/proto"

// GetExchangeAnnouncement 获取交易所公告
func (client *Client) GetExchangeAnnouncement() (*proto.ExchangeAnnouncementReply, error) {
	obj := proto.NewExchangeAnnouncement()
	return executeMsg(client, obj, obj.Reply)
}

// GetServerHeartbeat 获取服务端心跳返回
func (client *Client) GetServerHeartbeat() (*proto.HeartBeatReply, error) {
	obj := proto.NewHeartBeat()
	return executeMsg(client, obj, obj.Reply)
}

// GetAnnouncement 获取服务商公告
func (client *Client) GetAnnouncement() (*proto.AnnouncementReply, error) {
	obj := proto.NewAnnouncement()
	return executeMsg(client, obj, obj.Reply)
}

// GetServerInfo 获取主站服务信息
func (client *Client) GetServerInfo() (*proto.InfoReply, error) {
	obj := proto.NewInfo()
	return executeMsg(client, obj, obj.Reply)
}

// GetTodoB 获取主站试验协议 0x000b 的原始响应
func (client *Client) GetTodoB() (*proto.RawDataReply, error) {
	obj := proto.NewTodoB()
	return executeMsg(client, obj, obj.Reply)
}

// GetTodoFDE 获取主站试验协议 0x0fde 的原始响应
func (client *Client) GetTodoFDE() (*proto.RawDataReply, error) {
	obj := proto.NewTodoFDE()
	return executeMsg(client, obj, obj.Reply)
}

// GetClient264B 获取客户端信息协议 0x264b 的原始响应
func (client *Client) GetClient264B() (*proto.RawDataReply, error) {
	obj := proto.NewClient264B()
	return executeMsg(client, obj, obj.Reply)
}

// GetClient26AC 获取客户端信息协议 0x26ac 的原始响应
func (client *Client) GetClient26AC() (*proto.RawDataReply, error) {
	obj := proto.NewClient26AC()
	return executeMsg(client, obj, obj.Reply)
}

// GetClient26AD 获取客户端信息协议 0x26ad 的原始响应
func (client *Client) GetClient26AD() (*proto.RawDataReply, error) {
	obj := proto.NewClient26AD()
	return executeMsg(client, obj, obj.Reply)
}

// GetClient26AE 获取客户端信息协议 0x26ae 的原始响应
func (client *Client) GetClient26AE() (*proto.RawDataReply, error) {
	obj := proto.NewClient26AE()
	return executeMsg(client, obj, obj.Reply)
}

// GetClient26B1 获取客户端信息协议 0x26b1 的原始响应
func (client *Client) GetClient26B1() (*proto.RawDataReply, error) {
	obj := proto.NewClient26B1()
	return executeMsg(client, obj, obj.Reply)
}

// GetSecurityCount 获取指定市场内的证券数目
func (client *Client) GetSecurityCount(market uint8) (*proto.GetSecurityCountReply, error) {
	obj := proto.NewGetSecurityCount()
	obj.SetParams(&proto.GetSecurityCountRequest{Market: uint16(market)})
	return executeMsg(client, obj, obj.Reply)
}

// GetSecurityQuotes 获取盘口五档报价
func (client *Client) GetSecurityQuotes(markets []uint8, codes []string) (*proto.GetSecurityQuotesReply, error) {
	return client.GetQuotesDetail(markets, codes)
}

// GetQuotesDetail 获取详细行情报价
func (client *Client) GetQuotesDetail(markets []uint8, codes []string) (*proto.GetSecurityQuotesReply, error) {
	stocks, err := makeStocks(markets, codes)
	if err != nil {
		return nil, err
	}
	obj := proto.NewGetSecurityQuotes()
	obj.SetParams(&proto.GetSecurityQuotesRequest{StockList: stocks})
	return executeMsg(client, obj, obj.Reply)
}

// GetSecurityList 获取市场内指定范围内的所有证券代码
func (client *Client) GetSecurityList(market uint8, start uint16) (*proto.GetSecurityListReply, error) {
	return client.GetSecurityListRange(market, uint32(start), DefaultSecurityListCount)
}

// GetSecurityListOld 获取旧版证券列表
func (client *Client) GetSecurityListOld(market uint8, start uint16) (*proto.GetSecurityListOldReply, error) {
	obj := proto.NewGetSecurityListOld()
	obj.SetParams(&proto.GetSecurityListOldRequest{Market: uint16(market), Start: start})
	return executeMsg(client, obj, obj.Reply)
}

// GetSecurityFeature452 获取证券扩展信息
func (client *Client) GetSecurityFeature452(start uint32, count uint32) (*proto.GetSecurityFeature452Reply, error) {
	obj := proto.NewGetSecurityFeature452()
	obj.SetParams(&proto.GetSecurityFeature452Request{Start: start, Count: count})
	return executeMsg(client, obj, obj.Reply)
}

// GetSecurityListRange 获取市场内指定范围内的证券代码
func (client *Client) GetSecurityListRange(market uint8, start uint32, count uint32) (*proto.GetSecurityListReply, error) {
	obj := proto.NewGetSecurityList()
	obj.SetParams(&proto.GetSecurityListRequest{Market: uint16(market), Start: start, Count: count})
	return executeMsg(client, obj, obj.Reply)
}

// GetKLine 获取K线
func (client *Client) GetKLine(category uint16, market uint8, code string, start uint16, count uint16, times uint16, adjust uint16) (*proto.GetSecurityBarsReply, error) {
	obj := proto.NewGetSecurityBars()
	obj.SetParams(&proto.GetSecurityBarsRequest{
		Market:   uint16(market),
		Code:     makeCode6(code),
		Category: category,
		Times:    times,
		Start:    start,
		Count:    count,
		Adjust:   adjust,
	})
	return executeMsg(client, obj, obj.Reply)
}

// GetSecurityBars 获取股票K线
func (client *Client) GetSecurityBars(category uint16, market uint8, code string, start uint16, count uint16) (*proto.GetSecurityBarsReply, error) {
	return client.GetKLine(category, market, code, start, count, 1, AdjustNone)
}

// GetSecurityBarsOffset 获取偏移K线
func (client *Client) GetSecurityBarsOffset(category uint16, market uint8, code string, start uint16, count uint16, times uint16, adjust uint16) (*proto.GetSecurityBarsOffsetReply, error) {
	obj := proto.NewGetSecurityBarsOffset()
	obj.SetParams(&proto.GetSecurityBarsOffsetRequest{
		Market:   uint16(market),
		Code:     makeCode6(code),
		Category: category,
		Times:    times,
		Start:    start,
		Count:    count,
		Adjust:   adjust,
	})
	return executeMsg(client, obj, obj.Reply)
}

// GetIndexBars 获取指数K线
func (client *Client) GetIndexBars(category uint16, market uint8, code string, start uint16, count uint16) (*proto.GetIndexBarsReply, error) {
	obj := proto.NewGetIndexBars()
	obj.SetParams(&proto.GetIndexBarsRequest{
		Market:   uint16(market),
		Code:     makeCode6(code),
		Category: category,
		Times:    1,
		Start:    start,
		Count:    count,
		Adjust:   AdjustNone,
	})
	return executeMsg(client, obj, obj.Reply)
}

// GetIndexMomentum 获取指数动量
func (client *Client) GetIndexMomentum(market uint8, code string) (*proto.GetIndexMomentumReply, error) {
	obj := proto.NewGetIndexMomentum()
	obj.SetParams(&proto.GetIndexMomentumRequest{Market: uint16(market), Code: makeCode6(code)})
	return executeMsg(client, obj, obj.Reply)
}

// GetIndexInfo 获取指数概况
func (client *Client) GetIndexInfo(market uint8, code string) (*proto.GetIndexInfoReply, error) {
	obj := proto.NewGetIndexInfo()
	obj.SetParams(&proto.GetIndexInfoRequest{Market: uint16(market), Code: makeCode6(code)})
	return executeMsg(client, obj, obj.Reply)
}

// GetVolumeProfile 获取成交分布
func (client *Client) GetVolumeProfile(market uint8, code string) (*proto.GetVolumeProfileReply, error) {
	obj := proto.NewGetVolumeProfile()
	obj.SetParams(&proto.GetVolumeProfileRequest{Market: uint16(market), Code: makeCode6(code)})
	return executeMsg(client, obj, obj.Reply)
}

// GetQuotesList 获取排序行情列表
func (client *Client) GetQuotesList(category uint8, start uint16, count uint16, sortType uint16, reverse bool, filter uint16) (*proto.GetQuotesListReply, error) {
	obj := proto.NewGetQuotesList()
	obj.SetParams(&proto.GetQuotesListRequest{
		Category:    uint16(category),
		SortType:    sortType,
		Start:       start,
		Count:       count,
		SortReverse: quotesSortReverse(sortType, reverse),
		Filter:      filter,
	})
	return executeMsg(client, obj, obj.Reply)
}

// GetQuotes 获取批量行情
func (client *Client) GetQuotes(markets []uint8, codes []string) (*proto.GetQuotesReply, error) {
	stocks, err := makeStocks(markets, codes)
	if err != nil {
		return nil, err
	}
	obj := proto.NewGetQuotes()
	obj.SetParams(&proto.GetQuotesRequest{Stocks: stocks})
	return executeMsg(client, obj, obj.Reply)
}

// GetQuotesEncrypt 获取加密行情
func (client *Client) GetQuotesEncrypt(markets []uint8, codes []string) (*proto.GetQuotesEncryptReply, error) {
	stocks, err := makeStocks(markets, codes)
	if err != nil {
		return nil, err
	}
	obj := proto.NewGetQuotesEncrypt()
	obj.SetParams(&proto.GetQuotesEncryptRequest{Stocks: stocks})
	return executeMsg(client, obj, obj.Reply)
}

// GetMinuteTimeData 获取分时图数据
func (client *Client) GetMinuteTimeData(market uint8, code string) (*proto.GetMinuteTimeDataReply, error) {
	return client.GetTickChart(market, code, 0, DefaultTickChartCount)
}

// GetTickChart 获取当日分时图数据
func (client *Client) GetTickChart(market uint8, code string, start uint16, count uint16) (*proto.GetMinuteTimeDataReply, error) {
	obj := proto.NewGetMinuteTimeData()
	obj.SetParams(&proto.GetMinuteTimeDataRequest{
		Market: uint16(market),
		Code:   makeCode6(code),
		Start:  start,
		Count:  count,
	})
	return executeMsg(client, obj, obj.Reply)
}

// GetHistoryMinuteTimeData 获取历史分时图数据
func (client *Client) GetHistoryMinuteTimeData(date uint32, market uint8, code string) (*proto.GetHistoryMinuteTimeDataReply, error) {
	return client.GetHistoryTickChart(date, market, code)
}

// GetHistoryTickChart 获取历史分时图数据
func (client *Client) GetHistoryTickChart(date uint32, market uint8, code string) (*proto.GetHistoryMinuteTimeDataReply, error) {
	obj := proto.NewGetHistoryMinuteTimeData()
	obj.SetParams(&proto.GetHistoryMinuteTimeDataRequest{
		Date:   int32(date),
		Market: market,
		Code:   makeCode6(code),
	})
	return executeMsg(client, obj, obj.Reply)
}

// GetChartSampling 获取抽样图数据
func (client *Client) GetChartSampling(market uint8, code string) (*proto.GetChartSamplingReply, error) {
	obj := proto.NewGetChartSampling()
	obj.SetParams(&proto.GetChartSamplingRequest{Market: uint16(market), Code: makeCode6(code)})
	return executeMsg(client, obj, obj.Reply)
}

// GetAuction 获取集合竞价
func (client *Client) GetAuction(market uint8, code string, start uint32, count uint32) (*proto.GetAuctionReply, error) {
	obj := proto.NewGetAuction()
	obj.SetParams(&proto.GetAuctionRequest{
		Market: uint16(market),
		Code:   makeCode6(code),
		Start:  start,
		Count:  count,
	})
	return executeMsg(client, obj, obj.Reply)
}

// GetTopBoard 获取排行榜
func (client *Client) GetTopBoard(category uint8, size uint8) (*proto.GetTopBoardReply, error) {
	obj := proto.NewGetTopBoard()
	obj.SetParams(&proto.GetTopBoardRequest{Category: category, Size: size})
	return executeMsg(client, obj, obj.Reply)
}

// GetUnusual 获取主力监控
func (client *Client) GetUnusual(market uint8, start uint32, count uint32) (*proto.GetUnusualReply, error) {
	obj := proto.NewGetUnusual()
	obj.SetParams(&proto.GetUnusualRequest{Market: uint16(market), Start: start, Count: count})
	return executeMsg(client, obj, obj.Reply)
}

// GetTransactionData 获取分时成交
func (client *Client) GetTransactionData(market uint8, code string, start uint16, count uint16) (*proto.GetTransactionDataReply, error) {
	obj := proto.NewGetTransactionData()
	obj.SetParams(&proto.GetTransactionDataRequest{
		Market: uint16(market),
		Code:   makeCode6(code),
		Start:  start,
		Count:  count,
	})
	return executeMsg(client, obj, obj.Reply)
}

// GetHistoryOrders 获取历史委托
func (client *Client) GetHistoryOrders(date uint32, market uint8, code string) (*proto.GetHistoryOrdersReply, error) {
	obj := proto.NewGetHistoryOrders()
	obj.SetParams(&proto.GetHistoryOrdersRequest{
		Date:   date,
		Market: market,
		Code:   makeCode6(code),
	})
	return executeMsg(client, obj, obj.Reply)
}

// GetHistoryTransactionData 获取历史分时成交
func (client *Client) GetHistoryTransactionData(date uint32, market uint8, code string, start uint16, count uint16) (*proto.GetHistoryTransactionDataReply, error) {
	obj := proto.NewGetHistoryTransactionData()
	obj.SetParams(&proto.GetHistoryTransactionDataRequest{
		Date:   date,
		Market: uint16(market),
		Code:   makeCode6(code),
		Start:  start,
		Count:  count,
	})
	return executeMsg(client, obj, obj.Reply)
}

// GetHistoryTransactionDataWithTrans 获取带方向的历史分时成交
func (client *Client) GetHistoryTransactionDataWithTrans(date uint32, market uint8, code string, start uint16, count uint16) (*proto.GetHistoryTransactionDataWithTransReply, error) {
	obj := proto.NewGetHistoryTransactionDataWithTrans()
	obj.SetParams(&proto.GetHistoryTransactionDataRequest{
		Date:   date,
		Market: uint16(market),
		Code:   makeCode6(code),
		Start:  start,
		Count:  count,
	})
	return executeMsg(client, obj, obj.Reply)
}

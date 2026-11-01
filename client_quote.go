package gotdx

import (
	"github.com/bensema/gotdx/proto"
	"github.com/bensema/gotdx/types"
)

// GetExchangeAnnouncement 拉取交易所公告内容，返回公告版本号与已解码的公告正文。
//
// 该请求不携带业务参数；连接未建立、协议收发或响应负载为空时返回错误。
func (client *Client) GetExchangeAnnouncement() (*proto.ExchangeAnnouncementReply, error) {
	obj := proto.NewExchangeAnnouncement()
	return client.execute(obj)
}

// GetServerHeartbeat 发送心跳请求并返回服务端应答，用于维持长连接活性；应答中记录服务端
// 返回的打包日期整数。
//
// 该请求不携带业务参数；连接或协议执行失败时返回错误。
func (client *Client) GetServerHeartbeat() (*proto.HeartBeatReply, error) {
	obj := proto.NewHeartBeat()
	return client.execute(obj)
}

// GetAnnouncement 拉取行情服务商公告，返回标题、作者、正文与失效日期；公告无内容时应答中
// `HasContent` 为 `false`。
//
// 该请求不携带业务参数；连接或协议执行失败时返回错误。
func (client *Client) GetAnnouncement() (*proto.AnnouncementReply, error) {
	obj := proto.NewAnnouncement()
	return client.execute(obj)
}

// GetServerInfo 拉取主站服务信息，返回网络时延、服务端时间、服务端文案、签名与标识等字段。
//
// 该请求不携带业务参数；连接或协议执行失败时返回错误。
func (client *Client) GetServerInfo() (*proto.InfoReply, error) {
	obj := proto.NewInfo()
	return client.execute(obj)
}

// GetTodoB 发送主站试验协议 `0x000b`，返回未做字段解析的原始响应；
// 该协议语义待确认，仅用于协议探测。
func (client *Client) GetTodoB() (*proto.RawDataReply, error) {
	obj := proto.NewTodoB()
	return client.execute(obj)
}

// GetTodoFDE 发送主站试验协议 `0x0fde`，返回未做字段解析的原始响应；
// 该协议语义待确认，仅用于协议探测。
func (client *Client) GetTodoFDE() (*proto.RawDataReply, error) {
	obj := proto.NewTodoFDE()
	return client.execute(obj)
}

// GetClient264B 发送客户端信息协议 `0x264b`，返回未做字段解析的原始响应；
// 该协议语义待确认，仅用于协议探测。
func (client *Client) GetClient264B() (*proto.RawDataReply, error) {
	obj := proto.NewClient264B()
	return client.execute(obj)
}

// GetClient26AC 发送客户端信息协议 `0x26ac`，返回未做字段解析的原始响应；
// 该协议语义待确认，仅用于协议探测。
func (client *Client) GetClient26AC() (*proto.RawDataReply, error) {
	obj := proto.NewClient26AC()
	return client.execute(obj)
}

// GetClient26AD 发送客户端信息协议 `0x26ad`，返回未做字段解析的原始响应；
// 该协议语义待确认，仅用于协议探测。
func (client *Client) GetClient26AD() (*proto.RawDataReply, error) {
	obj := proto.NewClient26AD()
	return client.execute(obj)
}

// GetClient26AE 发送客户端信息协议 `0x26ae`，返回未做字段解析的原始响应；
// 该协议语义待确认，仅用于协议探测。
func (client *Client) GetClient26AE() (*proto.RawDataReply, error) {
	obj := proto.NewClient26AE()
	return client.execute(obj)
}

// GetClient26B1 发送客户端信息协议 `0x26b1`，返回未做字段解析的原始响应；
// 该协议语义待确认，仅用于协议探测。
func (client *Client) GetClient26B1() (*proto.RawDataReply, error) {
	obj := proto.NewClient26B1()
	return client.execute(obj)
}

// GetSecurityCount 查询指定市场内的证券总数，通常在分页拉取证券列表前调用以计算分页范围。
//
// `market` 为市场代码，取值 `types.MarketSZ`（0，深圳）、`types.MarketSH`（1，上海）、
// `types.MarketBJ`（2，北京）、`types.MarketHK`（3，香港）、`types.MarketUSA`（4，美国）。
// 返回应答中的 `Count` 即该市场证券总数；连接未建立、协议收发或响应解析失败时返回错误。
func (client *Client) GetSecurityCount(market uint8) (*proto.GetSecurityCountReply, error) {
	obj := proto.NewGetSecurityCount(&proto.GetSecurityCountRequest{Market: uint16(market)})
	return client.execute(obj)
}

// GetSecurityQuotes 批量查询证券盘口报价，内部委托 `GetQuotesDetail` 实现。
//
// `markets` 与 `codes` 等长，按下标一一配对发送；`markets` 元素为市场代码（取值见
// `types.Market*` 常量），`codes` 元素为证券代码。两切片长度不一致时返回
// `ErrMarketCodeCount`；连接或协议执行失败时返回错误。
func (client *Client) GetSecurityQuotes(markets []uint8, codes []string) (*proto.GetSecurityQuotesReply, error) {
	return client.GetQuotesDetail(markets, codes)
}

// GetQuotesDetail 批量查询详细行情报价，返回每只证券的最新价、开盘价、最高价、最低价、
// 成交量、成交额与换手率等盘口字段。
//
// `markets` 与 `codes` 等长，按下标一一配对；`markets` 元素为市场代码（取值见
// `types.Market*` 常量），`codes` 元素为证券代码。两切片长度不一致时返回
// `ErrMarketCodeCount`；连接或协议执行失败时返回错误。
func (client *Client) GetQuotesDetail(markets []uint8, codes []string) (*proto.GetSecurityQuotesReply, error) {
	stocks, err := makeStocks(markets, codes)
	if err != nil {
		return nil, err
	}
	obj := proto.NewGetSecurityQuotes(&proto.GetSecurityQuotesRequest{StockList: stocks})
	return client.execute(obj)
}

// GetSecurityList 从 `start` 偏移开始，按默认条数 `types.DefaultSecurityListCount`
// 拉取指定市场内的证券代码与名称列表。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`start` 为起始偏移条数，从 0 开始
// 计数。返回应答中的 `List` 为分页后的证券条目；连接或协议执行失败时返回错误。
func (client *Client) GetSecurityList(market uint8, start uint16) (*proto.GetSecurityListReply, error) {
	return client.GetSecurityListRange(market, uint32(start), types.DefaultSecurityListCount)
}

// GetSecurityListOld 使用旧版协议从 `start` 偏移开始拉取指定市场内的证券代码与名称列表。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`start` 为起始偏移条数，从 0 开始
// 计数。连接或协议执行失败时返回错误。
func (client *Client) GetSecurityListOld(market uint8, start uint16) (*proto.GetSecurityListOldReply, error) {
	obj := proto.NewGetSecurityListOld(&proto.GetSecurityListOldRequest{Market: uint16(market), Start: start})
	return client.execute(obj)
}

// GetSecurityFeature452 按 `start` 偏移与 `count` 条数批量获取证券扩展信息。
//
// `start` 为起始偏移条数，从 0 开始计数；`count` 为请求条数。返回条目中 `P1`、`P2`
// 字段的具体业务含义语义待确认；连接或协议执行失败时返回错误。
func (client *Client) GetSecurityFeature452(start uint32, count uint32) (*proto.GetSecurityFeature452Reply, error) {
	obj := proto.NewGetSecurityFeature452(&proto.GetSecurityFeature452Request{Start: start, Count: count})
	return client.execute(obj)
}

// GetSecurityListRange 从 `start` 偏移开始拉取 `count` 条指定市场内的证券代码与名称列表。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`start` 为起始偏移条数，从 0 开始
// 计数；`count` 为本次请求条数。连接或协议执行失败时返回错误。
func (client *Client) GetSecurityListRange(market uint8, start uint32, count uint32) (*proto.GetSecurityListReply, error) {
	obj := proto.NewGetSecurityList(&proto.GetSecurityListRequest{Market: uint16(market), Start: start, Count: count})
	return client.execute(obj)
}

// GetKLine 查询证券 K 线序列，返回按周期从早到晚排列的开高低收、成交量、成交额与时间条目。
//
// `category` 为 K 线周期类型，取值 `types.KLINE_TYPE_*` 常量（如 `types.KLINE_TYPE_DAILY`
// 日 K、`types.KLINE_TYPE_5MIN` 五分钟 K）；`market` 为市场代码（取值见 `types.Market*`
// 常量）；`code` 为证券代码，内部按 6 字节定长编码、尾部补零；`start` 为起始偏移条数，
// 从 0 开始计数；`count` 为请求条数；`times` 为周期倍数，通常取 1；`adjust` 为复权方式，
// 取值 `types.AdjustNone`（0，不复权）、`types.AdjustQFQ`（1，前复权）、`types.AdjustHFQ`
// （2，后复权）。连接或协议执行失败时返回错误。
func (client *Client) GetKLine(category uint16, market uint8, code string, start uint16, count uint16, times uint16, adjust uint16) (*proto.GetSecurityBarsReply, error) {
	obj := proto.NewGetSecurityBars(&proto.GetSecurityBarsRequest{
		Market:   uint16(market),
		Code:     makeCode6(code),
		Category: category,
		Times:    times,
		Start:    start,
		Count:    count,
		Adjust:   adjust,
	})
	return client.execute(obj)
}

// GetSecurityBars 查询股票 K 线，固定周期倍数为 1、不复权，返回开高低收、成交量、成交额与
// 时间条目。
//
// `category` 为 K 线周期类型，取值 `types.KLINE_TYPE_*` 常量；`market` 为市场代码（取值见
// `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节定长编码、尾部补零；`start` 为起始
// 偏移条数，从 0 开始计数；`count` 为请求条数。连接或协议执行失败时返回错误。
func (client *Client) GetSecurityBars(category uint16, market uint8, code string, start uint16, count uint16) (*proto.GetSecurityBarsReply, error) {
	return client.GetKLine(category, market, code, start, count, 1, types.AdjustNone)
}

// GetSecurityBarsOffset 使用偏移量协议查询证券 K 线，返回由独立偏移协议响应结构承载的
// 开高低收、成交量、成交额与时间条目。
//
// `category` 为 K 线周期类型，取值 `types.KLINE_TYPE_*` 常量；`market` 为市场代码（取值见
// `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节定长编码、尾部补零；`start` 为起始
// 偏移条数，从 0 开始计数；`count` 为请求条数；`times` 为周期倍数，通常取 1；`adjust` 为复权
// 方式，取值 `types.AdjustNone`、`types.AdjustQFQ`、`types.AdjustHFQ`。连接或协议执行失败时
// 返回错误。
func (client *Client) GetSecurityBarsOffset(category uint16, market uint8, code string, start uint16, count uint16, times uint16, adjust uint16) (*proto.GetSecurityBarsOffsetReply, error) {
	obj := proto.NewGetSecurityBarsOffset(&proto.GetSecurityBarsOffsetRequest{
		Market:   uint16(market),
		Code:     makeCode6(code),
		Category: category,
		Times:    times,
		Start:    start,
		Count:    count,
		Adjust:   adjust,
	})
	return client.execute(obj)
}

// GetIndexBars 查询指数 K 线序列，固定周期倍数为 1、不复权。
//
// `category` 为 K 线周期类型，取值 `types.KLINE_TYPE_*` 常量；`market` 为市场代码（取值见
// `types.Market*` 常量）；`code` 为指数代码，内部按 6 字节定长编码、尾部补零；`start` 为
// 起始偏移条数，从 0 开始计数；`count` 为请求条数。连接或协议执行失败时返回错误。
func (client *Client) GetIndexBars(category uint16, market uint8, code string, start uint16, count uint16) (*proto.GetIndexBarsReply, error) {
	_, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	obj := proto.NewGetIndexBars(&proto.GetIndexBarsRequest{
		Market:   uint16(market),
		Code:     makeCode6(code),
		Category: category,
		Times:    1,
		Start:    start,
		Count:    count,
		Adjust:   types.AdjustNone,
	})
	return client.execute(obj)
}

// GetIndexMomentum 查询指定指数的动量数据，返回累计动量值序列。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`code` 为指数代码，内部按 6 字节
// 定长编码、尾部补零。连接或协议执行失败时返回错误。
func (client *Client) GetIndexMomentum(market uint8, code string) (*proto.GetIndexMomentumReply, error) {
	obj := proto.NewGetIndexMomentum(&proto.GetIndexMomentumRequest{Market: uint16(market), Code: makeCode6(code)})
	return client.execute(obj)
}

// GetIndexInfo 查询指定指数的实时概况，返回点位、涨跌、成交及上涨下跌家数与档位分布。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`code` 为指数代码，内部按 6 字节
// 定长编码、尾部补零。连接或协议执行失败时返回错误。
func (client *Client) GetIndexInfo(market uint8, code string) (*proto.GetIndexInfoReply, error) {
	obj := proto.NewGetIndexInfo(&proto.GetIndexInfoRequest{Market: uint16(market), Code: makeCode6(code)})
	return client.execute(obj)
}

// GetVolumeProfile 查询指定证券的价位成交分布（量价分布），响应前段附带一份盘口快照。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节
// 定长编码、尾部补零。连接或协议执行失败时返回错误。
func (client *Client) GetVolumeProfile(market uint8, code string) (*proto.GetVolumeProfileReply, error) {
	obj := proto.NewGetVolumeProfile(&proto.GetVolumeProfileRequest{Market: uint16(market), Code: makeCode6(code)})
	return client.execute(obj)
}

// GetQuotesList 按分类拉取排序后的行情快照列表，返回排序后的证券快照条目。
//
// `category` 为板块或市场分类编号；`start` 为起始偏移条数，从 0 开始按页递增；`count`
// 为请求条数；`sortType` 为排序字段，取值 `types.Sort*` 常量；`reverse` 为 `true` 时按降序
// 排列、`false` 时按升序排列（按代码排序时不区分升降序）；`filter` 为可按位组合的筛选
// 条件，取值 `types.Filter*` 常量。连接或协议执行失败时返回错误。
func (client *Client) GetQuotesList(category uint8, start uint16, count uint16, sortType uint16, reverse bool, filter uint16) (*proto.GetQuotesListReply, error) {
	obj := proto.NewGetQuotesList(&proto.GetQuotesListRequest{
		Category:    uint16(category),
		SortType:    sortType,
		Start:       start,
		Count:       count,
		SortReverse: quotesSortReverse(sortType, reverse),
		Filter:      filter,
	})
	return client.execute(obj)
}

// GetQuotes 批量查询证券行情快照，返回每只证券的最新行情条目。
//
// `markets` 与 `codes` 等长，按下标一一配对；`markets` 元素为市场代码（取值见
// `types.Market*` 常量），`codes` 元素为证券代码。两切片长度不一致时返回
// `ErrMarketCodeCount`；连接或协议执行失败时返回错误。
func (client *Client) GetQuotes(markets []uint8, codes []string) (*proto.GetQuotesReply, error) {
	stocks, err := makeStocks(markets, codes)
	if err != nil {
		return nil, err
	}
	obj := proto.NewGetQuotes(&proto.GetQuotesRequest{Stocks: stocks})
	return client.execute(obj)
}

// GetQuotesEncrypt 批量查询加密通道行情快照，返回加密协议解析后的每只证券行情条目。
//
// `markets` 与 `codes` 等长，按下标一一配对；`markets` 元素为市场代码（取值见
// `types.Market*` 常量），`codes` 元素为证券代码。两切片长度不一致时返回
// `ErrMarketCodeCount`；连接或协议执行失败时返回错误。
func (client *Client) GetQuotesEncrypt(markets []uint8, codes []string) (*proto.GetQuotesEncryptReply, error) {
	stocks, err := makeStocks(markets, codes)
	if err != nil {
		return nil, err
	}
	obj := proto.NewGetQuotesEncrypt(&proto.GetQuotesEncryptRequest{Stocks: stocks})
	return client.execute(obj)
}

// GetMinuteTimeData 查询当日分时图数据，从第 0 点开始按默认条数 `types.DefaultTickChartCount`
// 拉取。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节
// 定长编码、尾部补零。返回应答中的 `List` 为当日分时价格、均价与成交量采样点；连接或
// 协议执行失败时返回错误。
func (client *Client) GetMinuteTimeData(market uint8, code string) (*proto.GetMinuteTimeDataReply, error) {
	return client.GetTickChart(market, code, 0, types.DefaultTickChartCount)
}

// GetTickChart 查询当日分时图数据，返回分时价格、均价与成交量采样点序列。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节
// 定长编码、尾部补零；`start` 为分时点起始偏移，从 0 开始计数；`count` 为请求点数。
// 连接或协议执行失败时返回错误。
func (client *Client) GetTickChart(market uint8, code string, start uint16, count uint16) (*proto.GetMinuteTimeDataReply, error) {
	obj := proto.NewGetMinuteTimeData(&proto.GetMinuteTimeDataRequest{
		Market: uint16(market),
		Code:   makeCode6(code),
		Start:  start,
		Count:  count,
	})
	return client.execute(obj)
}

// GetHistoryMinuteTimeData 查询指定交易日的历史分时图数据，内部委托 `GetHistoryTickChart` 实现。
//
// `date` 为交易日期，按 `YYYYMMDD` 八位正整数传入（如 `20240102`）；`market` 为市场代码
// （取值见 `types.Market*` 常量）；`code` 为证券代码。连接或协议执行失败时返回错误。
func (client *Client) GetHistoryMinuteTimeData(date uint32, market uint8, code string) (*proto.GetHistoryMinuteTimeDataReply, error) {
	return client.GetHistoryTickChart(date, market, code)
}

// GetHistoryTickChart 查询指定交易日的历史分时图数据，返回当日分时价格、均价与成交量采样点。
//
// `date` 为交易日期，按 `YYYYMMDD` 八位正整数传入；`market` 为市场代码（取值见
// `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节定长编码、尾部补零。连接或
// 协议执行失败时返回错误。
func (client *Client) GetHistoryTickChart(date uint32, market uint8, code string) (*proto.GetHistoryMinuteTimeDataReply, error) {
	obj := proto.NewGetHistoryMinuteTimeData(&proto.GetHistoryMinuteTimeDataRequest{
		Date:   int32(date),
		Market: market,
		Code:   makeCode6(code),
	})
	return client.execute(obj)
}

// GetChartSampling 查询指定证券的分时缩略采样价格序列，返回服务端压缩后的采样价格点。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节
// 定长编码、尾部补零。连接或协议执行失败时返回错误。
func (client *Client) GetChartSampling(market uint8, code string) (*proto.GetChartSamplingReply, error) {
	obj := proto.NewGetChartSampling(&proto.GetChartSamplingRequest{Market: uint16(market), Code: makeCode6(code)})
	return client.execute(obj)
}

// GetAuction 查询指定证券的集合竞价记录，返回竞价时间、价格与量能明细列表。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节
// 定长编码、尾部补零；`start` 为起始偏移条数，从 0 开始计数；`count` 为请求明细条数，
// 传 0 时按默认 500 条处理。连接或协议执行失败时返回错误。
func (client *Client) GetAuction(market uint8, code string, start uint32, count uint32) (*proto.GetAuctionReply, error) {
	obj := proto.NewGetAuction(&proto.GetAuctionRequest{
		Market: uint16(market),
		Code:   makeCode6(code),
		Start:  start,
		Count:  count,
	})
	return client.execute(obj)
}

// GetTopBoard 查询分类排行榜，返回各榜单按排序字段排列的条目集合。
//
// `category` 为榜单分类编号；`size` 为每个榜单返回的条目数量。连接或协议执行失败时返回错误。
func (client *Client) GetTopBoard(category uint8, size uint8) (*proto.GetTopBoardReply, error) {
	obj := proto.NewGetTopBoard(&proto.GetTopBoardRequest{Category: category, Size: size})
	return client.execute(obj)
}

// GetUnusual 查询异动监控（主力监控）记录，返回异动事件明细列表。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`start` 为起始偏移条数，从 0 开始
// 计数；`count` 为请求条数，传 0 时按默认 600 条处理。连接或协议执行失败时返回错误。
func (client *Client) GetUnusual(market uint8, start uint32, count uint32) (*proto.GetUnusualReply, error) {
	obj := proto.NewGetUnusual(&proto.GetUnusualRequest{Market: uint16(market), Start: start, Count: count})
	return client.execute(obj)
}

// GetTransactionData 查询当日分时逐笔成交，返回当日按时间排列的逐笔成交记录。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节
// 定长编码、尾部补零；`start` 为起始偏移条数，从 0 开始计数；`count` 为请求条数。
// 连接或协议执行失败时返回错误。
func (client *Client) GetTransactionData(market uint8, code string, start uint16, count uint16) (*proto.GetTransactionDataReply, error) {
	obj := proto.NewGetTransactionData(&proto.GetTransactionDataRequest{
		Market: uint16(market),
		Code:   makeCode6(code),
		Start:  start,
		Count:  count,
	})
	return client.execute(obj)
}

// GetHistoryOrders 查询指定交易日的历史委托档位记录，返回档位价格与挂单量列表。
//
// `date` 为交易日期，按 `YYYYMMDD` 八位正整数传入；`market` 为市场代码（取值见
// `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节定长编码、尾部补零。连接或
// 协议执行失败时返回错误。
func (client *Client) GetHistoryOrders(date uint32, market uint8, code string) (*proto.GetHistoryOrdersReply, error) {
	obj := proto.NewGetHistoryOrders(&proto.GetHistoryOrdersRequest{
		Date:   date,
		Market: market,
		Code:   makeCode6(code),
	})
	return client.execute(obj)
}

// GetHistoryTransactionData 查询指定交易日的历史分时逐笔成交，返回按时间排列的成交记录。
//
// `date` 为交易日期，按 `YYYYMMDD` 八位正整数传入；`market` 为市场代码（取值见
// `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节定长编码、尾部补零；`start` 为
// 起始偏移条数，从 0 开始计数；`count` 为请求条数。连接或协议执行失败时返回错误。
func (client *Client) GetHistoryTransactionData(date uint32, market uint8, code string, start uint16, count uint16) (*proto.GetHistoryTransactionDataReply, error) {
	obj := proto.NewGetHistoryTransactionData(&proto.GetHistoryTransactionDataRequest{
		Date:   date,
		Market: uint16(market),
		Code:   makeCode6(code),
		Start:  start,
		Count:  count,
	})
	return client.execute(obj)
}

// GetHistoryTransactionDataWithTrans 查询带买卖方向标识的历史分时逐笔成交，每条记录额外
// 携带主动买、主动卖或中性方向编码。
//
// `date` 为交易日期，按 `YYYYMMDD` 八位正整数传入；`market` 为市场代码（取值见
// `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节定长编码、尾部补零；`start` 为
// 起始偏移条数，从 0 开始计数；`count` 为请求条数。连接或协议执行失败时返回错误。
func (client *Client) GetHistoryTransactionDataWithTrans(date uint32, market uint8, code string, start uint16, count uint16) (*proto.GetHistoryTransactionDataWithTransReply, error) {
	obj := proto.NewGetHistoryTransactionDataWithTrans(&proto.GetHistoryTransactionDataRequest{
		Date:   date,
		Market: uint16(market),
		Code:   makeCode6(code),
		Start:  start,
		Count:  count,
	})
	return client.execute(obj)
}

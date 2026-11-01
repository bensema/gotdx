package gotdx

import (
	"fmt"

	"github.com/bensema/gotdx/proto"
)

const (
	// DefaultGoodsHistoryTransactionCount 是 GoodsHistoryTransaction 的 `count` 为 0 时使用的
	// 默认历史成交条数，取值 2000。
	DefaultGoodsHistoryTransactionCount uint32 = 2000
	// MaxGoodsVarietiesCount 是 GoodsVarieties 单次请求允许的最大品种条数，取值 1000；
	// 请求 `count` 超过该上限时在本地直接返回参数错误，不再发送协议请求。
	MaxGoodsVarietiesCount uint32 = 1000
)

// GoodsCount 查询扩展市场商品总数，内部委托 `ExCount` 实现，返回品种总数。
//
// 该请求不携带业务参数；连接或协议执行失败时返回错误。
func (client *Client) GoodsCount() (uint32, error) {
	return client.ExCount()
}

// GoodsCategoryList 查询扩展市场商品分类列表，内部委托 `ExCategoryList` 实现，返回各分类条目。
//
// 该请求不携带业务参数；连接或协议执行失败时返回错误。
func (client *Client) GoodsCategoryList() ([]proto.ExCategoryItem, error) {
	return client.ExCategoryList()
}

// GoodsList 从 `start` 偏移开始拉取扩展市场商品列表，内部委托 `ExList` 实现。
//
// `start` 为起始偏移条数，从 0 开始计数；`count` 为请求条数，传 0 时由扩展协议按默认条数
// 处理。连接或协议执行失败时返回错误。
func (client *Client) GoodsList(start uint32, count uint16) ([]proto.ExListItem, error) {
	return client.ExList(start, count)
}

// GoodsVarieties 查询扩展市场品种列表，内部委托 `ExMapping2562` 实现。
//
// `market` 为市场代码；`start` 为起始偏移条数，从 0 开始计数；`count` 为请求条数，超过
// `MaxGoodsVarietiesCount` 时在本地返回参数错误。连接或协议执行失败时返回错误。
func (client *Client) GoodsVarieties(market uint16, start uint32, count uint32) ([]proto.ExMapping2562Item, error) {
	if count > MaxGoodsVarietiesCount {
		return nil, fmt.Errorf("goods varieties count exceeds %d: %d", MaxGoodsVarietiesCount, count)
	}
	return client.ExMapping2562(market, start, count)
}

// GoodsQuote 查询单个扩展市场商品报价，内部委托 `ExQuote` 实现，返回该商品的行情快照条目。
//
// `market` 为扩展市场类别编号；`code` 为商品代码。连接或协议执行失败时返回错误。
func (client *Client) GoodsQuote(market uint8, code string) (*proto.ExQuoteItem, error) {
	return client.ExQuote(market, code)
}

// GoodsQuotes 批量查询扩展市场商品报价，内部委托 `ExQuotes` 实现。
//
// `markets` 与 `codes` 等长，按下标一一配对；两切片长度不一致时返回 `ErrMarketCodeCount`；
// 连接或协议执行失败时返回错误。
func (client *Client) GoodsQuotes(markets []uint8, codes []string) ([]proto.ExQuoteItem, error) {
	return client.ExQuotes(markets, codes)
}

// GoodsQuotesList 查询扩展市场商品排序行情列表，内部委托 `ExQuotesList` 实现。
//
// `market` 为扩展市场类别编号；`start` 为起始偏移条数，从 0 开始计数；`count` 为请求条数，
// 传 0 时按扩展协议默认条数处理；`sortType` 为排序字段编号；`reverse` 为 `true` 时降序、
// `false` 时升序。连接或协议执行失败时返回错误。
func (client *Client) GoodsQuotesList(market uint8, start uint16, count uint16, sortType uint16, reverse bool) ([]proto.ExQuoteItem, error) {
	return client.ExQuotesList(market, start, count, sortType, reverse)
}

// GoodsKLine 查询扩展市场商品 MAC K 线，内部委托 `MACSymbolBars` 实现，返回 K 线条目序列。
//
// `market` 为市场代码；`code` 为商品代码；`period` 为 K 线周期类别；`start` 为起始偏移条数，
// 从 0 开始计数；`count` 为请求条数；`times` 为周期倍数，通常取 1；`adjust` 为复权方式，
// 取值 `types.AdjustNone`、`types.AdjustQFQ`、`types.AdjustHFQ`。连接或协议执行失败时
// 返回错误。
func (client *Client) GoodsKLine(market uint8, code string, period uint16, start uint32, count uint32, times uint16, adjust uint16) ([]proto.MACSymbolBar, error) {
	return client.MACSymbolBars(market, code, period, times, start, count, adjust)
}

// GoodsTickChart 查询扩展市场商品分时图，内部委托 `MACQuotesWithDate` 实现，返回分时图表数据点。
//
// `market` 为市场代码；`code` 为商品代码；`queryDate` 为查询日期，按 `YYYYMMDD` 八位正
// 整数传入，传 0 时返回实时分时，否则返回指定交易日的历史分时。连接或协议执行失败时
// 返回错误。
func (client *Client) GoodsTickChart(market uint8, code string, queryDate uint32) ([]proto.MACQuoteChartItem, error) {
	reply, err := client.MACQuotesWithDate(market, code, queryDate)
	if err != nil {
		return nil, err
	}
	return reply.ChartData, nil
}

// GoodsChartSampling 查询扩展市场商品分时缩略采样价格序列，内部委托 `ExChartSampling` 实现，
// 返回压缩后的采样价格浮点序列。
//
// `market` 为扩展市场类别编号；`code` 为商品代码。连接或协议执行失败时返回错误。
func (client *Client) GoodsChartSampling(market uint8, code string) ([]float64, error) {
	return client.ExChartSampling(market, code)
}

// GoodsHistoryTransaction 查询扩展市场商品历史成交，内部委托 `MACTransactionsWithDate` 实现，
// 返回按时间排列的逐笔成交条目。
//
// `market` 为市场代码；`code` 为商品代码；`queryDate` 为交易日期，按 `YYYYMMDD` 八位正
// 整数传入；`count` 为请求成交条数，传 0 时取 `DefaultGoodsHistoryTransactionCount` 条。
// 连接或协议执行失败时返回错误。
func (client *Client) GoodsHistoryTransaction(market uint8, code string, queryDate uint32, count uint32) ([]proto.MACTransactionItem, error) {
	if count == 0 {
		count = DefaultGoodsHistoryTransactionCount
	}
	return client.MACTransactionsWithDate(market, code, 0, count, queryDate)
}

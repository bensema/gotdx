package gotdx

import (
	"math"

	"github.com/bensema/gotdx/proto"
)

func (client *Client) quotationClient() (*Client, error) {
	if client.mode == clientModeMain {
		if client.conn == nil {
			if _, err := client.Connect(); err != nil {
				return nil, err
			}
		}
		return client, nil
	}

	client.mu.Lock()
	if client.main == nil {
		client.main = newClientWithOptions(client.opt, clientModeMain)
	}
	main := client.main
	client.mu.Unlock()

	if main.conn == nil {
		if _, err := main.Connect(); err != nil {
			return nil, err
		}
	}
	return main, nil
}

func (client *Client) exQuotationClient() (*Client, error) {
	if client.mode == clientModeEx {
		if client.conn == nil {
			if _, err := client.ConnectEx(); err != nil {
				return nil, err
			}
		}
		return client, nil
	}

	client.mu.Lock()
	if client.ex == nil {
		client.ex = newClientWithOptions(client.opt, clientModeEx)
	}
	ex := client.ex
	client.mu.Unlock()

	if ex.conn == nil {
		if _, err := ex.ConnectEx(); err != nil {
			return nil, err
		}
	}
	return ex, nil
}

func (client *Client) StockCount(market uint8) (uint16, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return 0, err
	}
	reply, err := qc.GetSecurityCount(market)
	if err != nil {
		return 0, err
	}
	return reply.Count, nil
}

func (client *Client) StockList(market uint8, start uint32, count uint32) ([]proto.Security, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetSecurityListRange(market, start, count)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) StockListOld(market uint8, start uint16) ([]proto.Security, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetSecurityListOld(market, start)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) StockFeature452(start uint32, count uint32) ([]proto.SecurityFeature452Item, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetSecurityFeature452(start, count)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) StockKLine(category uint16, market uint8, code string, start uint16, count uint16, times uint16, adjust uint16) ([]proto.SecurityBar, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetKLine(category, market, code, start, count, times, adjust)
	if err != nil {
		return nil, err
	}
	applyTurnoverToBars(reply.List, client.loadFloatShares(qc, market, code))
	return reply.List, nil
}

func (client *Client) StockKLineOffset(category uint16, market uint8, code string, start uint16, count uint16, times uint16, adjust uint16) ([]proto.SecurityBar, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetSecurityBarsOffset(category, market, code, start, count, times, adjust)
	if err != nil {
		return nil, err
	}
	applyTurnoverToBars(reply.List, client.loadFloatShares(qc, market, code))
	return reply.List, nil
}

func (client *Client) StockTickChart(market uint8, code string, start uint16, count uint16) ([]proto.MinuteTimeData, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetTickChart(market, code, start, count)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) StockHistoryTickChart(date uint32, market uint8, code string) ([]proto.HistoryMinuteTimeData, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetHistoryTickChart(date, market, code)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

// StockIndexInfo 获取指数概况。
func (client *Client) StockIndexInfo(market uint8, code string) (*proto.GetIndexInfoReply, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	return qc.GetIndexInfo(market, code)
}

// StockIndexMomentum 获取指数动量序列。
func (client *Client) StockIndexMomentum(market uint8, code string) ([]int, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetIndexMomentum(market, code)
	if err != nil {
		return nil, err
	}
	return reply.Values, nil
}

// StockChartSampling 获取分时缩略采样价格。
func (client *Client) StockChartSampling(market uint8, code string) ([]float64, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetChartSampling(market, code)
	if err != nil {
		return nil, err
	}
	return reply.Prices, nil
}

func (client *Client) StockQuotesDetail(markets []uint8, codes []string) ([]proto.SecurityQuote, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetQuotesDetail(markets, codes)
	if err != nil {
		return nil, err
	}
	applyTurnoverToSecurityQuotes(reply.List, client.loadFloatSharesMap(qc, stockKeysFromPairLists(markets, codes)))
	return reply.List, nil
}

func (client *Client) StockQuotesList(category uint8, start uint16, count uint16, sortType uint16, reverse bool, filter uint16) ([]proto.QuoteListItem, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetQuotesList(category, start, count, sortType, reverse, filter)
	if err != nil {
		return nil, err
	}
	applyTurnoverToQuoteList(reply.List, client.loadFloatSharesMap(qc, stockKeysFromQuoteItems(reply.List)))
	return reply.List, nil
}

func (client *Client) StockQuotes(markets []uint8, codes []string) ([]proto.QuoteListItem, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetQuotes(markets, codes)
	if err != nil {
		return nil, err
	}
	applyTurnoverToQuoteList(reply.List, client.loadFloatSharesMap(qc, stockKeysFromQuoteItems(reply.List)))
	return reply.List, nil
}

func (client *Client) StockQuotesEncrypt(markets []uint8, codes []string) ([]proto.EncryptedQuoteItem, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetQuotesEncrypt(markets, codes)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

// StockAuction 获取集合竞价数据。
func (client *Client) StockAuction(market uint8, code string, start uint32, count uint32) ([]proto.AuctionData, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetAuction(market, code, start, count)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

// StockTopBoard 获取主行情排行榜。
func (client *Client) StockTopBoard(category uint8, size uint8) (*proto.GetTopBoardReply, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	return qc.GetTopBoard(category, size)
}

// StockUnusual 获取主力监控异动数据。
func (client *Client) StockUnusual(market uint8, start uint32, count uint32) ([]proto.UnusualData, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetUnusual(market, start, count)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

// StockVolumeProfile 获取成交分布与盘口快照。
func (client *Client) StockVolumeProfile(market uint8, code string) (*proto.GetVolumeProfileReply, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetVolumeProfile(market, code)
	if err != nil {
		return nil, err
	}
	applyTurnoverToVolumeProfile(reply, client.loadFloatShares(qc, market, code))
	return reply, nil
}

func (client *Client) StockTransaction(market uint8, code string, start uint16, count uint16) ([]proto.TransactionData, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetTransactionData(market, code, start, count)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

// StockHistoryOrders 获取历史委托分布。
func (client *Client) StockHistoryOrders(date uint32, market uint8, code string) ([]proto.HistoryOrderData, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetHistoryOrders(date, market, code)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) StockHistoryTransaction(date uint32, market uint8, code string, start uint16, count uint16) ([]proto.HistoryTransactionData, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetHistoryTransactionData(date, market, code, start, count)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) StockHistoryTransactionWithTrans(date uint32, market uint8, code string, start uint16, count uint16) ([]proto.HistoryTransactionDataWithTrans, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetHistoryTransactionDataWithTrans(date, market, code, start, count)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) StockF10(market uint8, code string) (*CompanyInfoBundle, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	return qc.GetCompanyInfo(market, code)
}

func (client *Client) StockBlock(filename string) ([]BlockFlatItem, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	return qc.GetParsedBlockFile(filename)
}

func (client *Client) MainTodoB() (*proto.RawDataReply, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	return qc.GetTodoB()
}

func (client *Client) MainTodoFDE() (*proto.RawDataReply, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	return qc.GetTodoFDE()
}

func (client *Client) MainClient264B() (*proto.RawDataReply, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	return qc.GetClient264B()
}

func (client *Client) MainClient26AC() (*proto.RawDataReply, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	return qc.GetClient26AC()
}

func (client *Client) MainClient26AD() (*proto.RawDataReply, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	return qc.GetClient26AD()
}

func (client *Client) MainClient26AE() (*proto.RawDataReply, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	return qc.GetClient26AE()
}

func (client *Client) MainClient26B1() (*proto.RawDataReply, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	return qc.GetClient26B1()
}

func (client *Client) ExCount() (uint32, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return 0, err
	}
	reply, err := eqc.ExGetCount()
	if err != nil {
		return 0, err
	}
	return reply.Count, nil
}

func (client *Client) ExCategoryList() ([]proto.ExCategoryItem, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetCategoryList()
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExList(start uint32, count uint16) ([]proto.ExListItem, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetList(start, count)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExListExtra(a uint16, b uint16, count uint16) ([]proto.ExExtraListItem, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetListExtra(a, b, count)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExQuotesList(category uint8, start uint16, count uint16, sortType uint16, reverse bool) ([]proto.ExQuoteItem, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetQuotesList(category, start, count, sortType, reverse)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExQuote(category uint8, code string) (*proto.ExQuoteItem, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetQuote(category, code)
	if err != nil {
		return nil, err
	}
	return &reply.Item, nil
}

func (client *Client) ExQuotes(categories []uint8, codes []string) ([]proto.ExQuoteItem, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetQuotes(categories, codes)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExQuotes2(categories []uint8, codes []string) ([]proto.ExQuoteItem, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetQuotes2(categories, codes)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExKLine(category uint8, code string, period uint16, start uint32, count uint16, times uint16) ([]proto.ExKLineItem, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetKLine(category, code, period, start, count, times)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExExperiment2487(category uint8, code string) (*proto.ExExperiment2487Reply, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	return eqc.ExGetExperiment2487(category, code)
}

func (client *Client) ExExperiment2488(category uint8, code string, mode uint16) ([]proto.ExExperiment2488Item, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetExperiment2488(category, code, mode)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExKLine2(category uint8, code string, period uint16, start uint32, count uint32, times uint16) ([]proto.ExKLineItem, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetKLine2(category, code, period, start, count, times)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExHistoryTransaction(date uint32, category uint8, code string) ([]proto.ExHistoryTransactionItem, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetHistoryTransaction(date, category, code)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExTickChart(category uint8, code string, date uint32) ([]proto.ExTickChartData, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	if date == 0 {
		reply, err := eqc.ExGetTickChart(category, code)
		if err != nil {
			return nil, err
		}
		return reply.List, nil
	}
	reply, err := eqc.ExGetHistoryTickChart(date, category, code)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExChartSampling(category uint8, code string) ([]float64, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetChartSampling(category, code)
	if err != nil {
		return nil, err
	}
	return reply.Prices, nil
}

func (client *Client) ExBoardList(boardType uint16, start uint16, pageSize uint16) ([]proto.ExBoardListItem, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetBoardList(boardType, start, pageSize)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExMapping2562(market uint16, start uint32, count uint32) ([]proto.ExMapping2562Item, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := eqc.ExGetMapping2562(market, start, count)
	if err != nil {
		return nil, err
	}
	return reply.List, nil
}

func (client *Client) ExTable() (string, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return "", err
	}
	return eqc.ExGetTable()
}

func (client *Client) ExTableDetail() (string, error) {
	eqc, err := client.exQuotationClient()
	if err != nil {
		return "", err
	}
	return eqc.ExGetTableDetail()
}

type stockKey struct {
	Market uint8
	Code   string
}

func stockKeysFromPairLists(markets []uint8, codes []string) []stockKey {
	items := make([]stockKey, 0, len(codes))
	for i, code := range codes {
		if i >= len(markets) {
			break
		}
		items = append(items, stockKey{Market: markets[i], Code: code})
	}
	return items
}

func stockKeysFromQuoteItems(items []proto.QuoteListItem) []stockKey {
	keys := make([]stockKey, 0, len(items))
	for _, item := range items {
		keys = append(keys, stockKey{Market: item.Market, Code: item.Code})
	}
	return keys
}

func (client *Client) loadFloatShares(qc *Client, market uint8, code string) float64 {
	if code == "" {
		return 0
	}
	return client.loadFloatSharesMap(qc, []stockKey{{Market: market, Code: code}})[stockKey{Market: market, Code: code}]
}

func (client *Client) loadFloatSharesMap(qc *Client, keys []stockKey) map[stockKey]float64 {
	out := make(map[stockKey]float64, len(keys))
	seen := make(map[stockKey]struct{}, len(keys))
	for _, key := range keys {
		if key.Code == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		finance, err := qc.GetFinanceInfo(key.Market, key.Code)
		if err != nil {
			continue
		}
		if finance == nil || finance.FloatShares <= 0 {
			continue
		}
		out[key] = float64(finance.FloatShares)
	}
	return out
}

func applyTurnoverToSecurityQuotes(items []proto.SecurityQuote, shares map[stockKey]float64) {
	for i := range items {
		if floatShares := shares[stockKey{Market: items[i].Market, Code: items[i].Code}]; floatShares > 0 {
			items[i].Turnover = round2(float64(items[i].Vol) * 10000 / floatShares)
		}
	}
}

func applyTurnoverToQuoteList(items []proto.QuoteListItem, shares map[stockKey]float64) {
	for i := range items {
		if floatShares := shares[stockKey{Market: items[i].Market, Code: items[i].Code}]; floatShares > 0 {
			items[i].Turnover = round2(float64(items[i].Vol) * 10000 / floatShares)
		}
	}
}

func applyTurnoverToBars(items []proto.SecurityBar, floatShares float64) {
	if floatShares <= 0 {
		return
	}
	for i := range items {
		items[i].Turnover = round2(items[i].Vol * 100 / floatShares)
	}
}

func applyTurnoverToVolumeProfile(reply *proto.GetVolumeProfileReply, floatShares float64) {
	if reply == nil || floatShares <= 0 {
		return
	}
	reply.Turnover = round2(float64(reply.Vol) * 10000 / floatShares)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

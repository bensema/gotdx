package gotdx

import "github.com/bensema/gotdx/proto"

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

func (client *Client) StockQuotesDetail(markets []uint8, codes []string) ([]proto.SecurityQuote, error) {
	qc, err := client.quotationClient()
	if err != nil {
		return nil, err
	}
	reply, err := qc.GetQuotesDetail(markets, codes)
	if err != nil {
		return nil, err
	}
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

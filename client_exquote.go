package gotdx

import "github.com/bensema/gotdx/proto"

// GetExServerInfo 获取扩展市场服务信息
func (client *Client) GetExServerInfo() (*proto.ExServerInfoReply, error) {
	obj := proto.NewExServerInfo()
	return executeMsg(client, obj, obj.Reply)
}

// ExGetCount 获取扩展市场标的数量
func (client *Client) ExGetCount() (*proto.ExGetCountReply, error) {
	obj := proto.NewExGetCount()
	return executeMsg(client, obj, obj.Reply)
}

// ExGetCategoryList 获取扩展市场分类列表
func (client *Client) ExGetCategoryList() (*proto.ExGetCategoryListReply, error) {
	obj := proto.NewExGetCategoryList()
	return executeMsg(client, obj, obj.Reply)
}

// ExGetList 获取扩展市场标的列表
func (client *Client) ExGetList(start uint32, count uint16) (*proto.ExGetListReply, error) {
	if count == 0 {
		count = DefaultExListCount
	}
	obj := proto.NewExGetList()
	obj.SetParams(&proto.ExGetListRequest{Start: start, Count: count})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetListExtra 获取扩展市场试验列表
func (client *Client) ExGetListExtra(a uint16, b uint16, count uint16) (*proto.ExGetListExtraReply, error) {
	obj := proto.NewExGetListExtra()
	obj.SetParams(&proto.ExGetListExtraRequest{A: a, B: b, Count: count})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetQuotesList 获取扩展市场行情列表
func (client *Client) ExGetQuotesList(category uint8, start uint16, count uint16, sortType uint16, reverse bool) (*proto.ExGetQuotesListReply, error) {
	if count == 0 {
		count = DefaultExQuotesCount
	}
	obj := proto.NewExGetQuotesList()
	obj.SetParams(&proto.ExGetQuotesListRequest{
		Category:    category,
		SortType:    sortType,
		Start:       start,
		Count:       count,
		SortReverse: quotesSortReverse(sortType, reverse),
	})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetQuote 获取单个扩展市场行情
func (client *Client) ExGetQuote(category uint8, code string) (*proto.ExGetQuoteReply, error) {
	obj := proto.NewExGetQuote()
	obj.SetParams(&proto.ExGetQuoteRequest{Category: category, Code: makeCode9(code)})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetQuotes 获取批量扩展市场行情
func (client *Client) ExGetQuotes(categories []uint8, codes []string) (*proto.ExGetQuotesReply, error) {
	stocks, err := makeExStocks(categories, codes)
	if err != nil {
		return nil, err
	}

	obj := proto.NewExGetQuotes()
	obj.SetParams(&proto.ExGetQuotesRequest{Stocks: stocks})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetQuotes2 获取批量扩展市场行情，兼容 pytdx2 的第二种批量接口
func (client *Client) ExGetQuotes2(categories []uint8, codes []string) (*proto.ExGetQuotesReply, error) {
	stocks, err := makeExStocks(categories, codes)
	if err != nil {
		return nil, err
	}

	obj := proto.NewExGetQuotes2()
	obj.SetParams(&proto.ExGetQuotesRequest{Stocks: stocks})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetKLine 获取扩展市场K线
func (client *Client) ExGetKLine(category uint8, code string, period uint16, start uint32, count uint16, times uint16) (*proto.ExGetKLineReply, error) {
	obj := proto.NewExGetKLine()
	obj.SetParams(&proto.ExGetKLineRequest{
		Category: category,
		Code:     makeCode9(code),
		Period:   period,
		Times:    times,
		Start:    start,
		Count:    count,
	})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetExperiment2487 获取扩展市场试验报价 0x2487
func (client *Client) ExGetExperiment2487(category uint8, code string) (*proto.ExExperiment2487Reply, error) {
	obj := proto.NewExExperiment2487()
	obj.SetParams(&proto.ExExperiment2487Request{Category: category, Code: makeCode23(code)})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetExperiment2488 获取扩展市场试验报价 0x2488
func (client *Client) ExGetExperiment2488(category uint8, code string, mode uint16) (*proto.ExExperiment2488Reply, error) {
	obj := proto.NewExExperiment2488()
	obj.SetParams(&proto.ExExperiment2488Request{
		Category: category,
		Code:     makeCode23(code),
		Mode:     mode,
	})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetKLine2 获取扩展市场 K 线协议 0x2489
func (client *Client) ExGetKLine2(category uint8, code string, period uint16, start uint32, count uint32, times uint16) (*proto.ExGetKLine2Reply, error) {
	obj := proto.NewExGetKLine2()
	obj.SetParams(&proto.ExGetKLine2Request{
		Category: category,
		Code:     makeCode23(code),
		Period:   period,
		Times:    times,
		Start:    start,
		Count:    count,
	})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetHistoryTransaction 获取扩展市场历史成交
func (client *Client) ExGetHistoryTransaction(date uint32, category uint8, code string) (*proto.ExGetHistoryTransactionReply, error) {
	obj := proto.NewExGetHistoryTransaction()
	obj.SetParams(&proto.ExGetHistoryTransactionRequest{
		Date:     date,
		Category: category,
		Code:     makeFixed43(code),
	})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetTickChart 获取扩展市场当日分时图
func (client *Client) ExGetTickChart(category uint8, code string) (*proto.ExGetTickChartReply, error) {
	obj := proto.NewExGetTickChart()
	obj.SetParams(&proto.ExGetTickChartRequest{
		Category: category,
		Code:     makeCode23(code),
	})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetHistoryTickChart 获取扩展市场历史分时图
func (client *Client) ExGetHistoryTickChart(date uint32, category uint8, code string) (*proto.ExGetHistoryTickChartReply, error) {
	obj := proto.NewExGetHistoryTickChart()
	obj.SetParams(&proto.ExGetHistoryTickChartRequest{
		Date:     date,
		Category: category,
		Code:     makeCode23(code),
	})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetChartSampling 获取扩展市场抽样图
func (client *Client) ExGetChartSampling(category uint8, code string) (*proto.ExGetChartSamplingReply, error) {
	obj := proto.NewExGetChartSampling()
	obj.SetParams(&proto.ExGetChartSamplingRequest{
		Category: uint16(category),
		Code:     makeCode22(code),
	})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetBoardList 获取扩展市场板块榜单
func (client *Client) ExGetBoardList(boardType uint16, start uint16, pageSize uint16) (*proto.ExGetBoardListReply, error) {
	obj := proto.NewExGetBoardList()
	obj.SetParams(&proto.ExGetBoardListRequest{
		PageSize:  pageSize,
		BoardType: boardType,
		Start:     start,
	})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetMapping2562 获取扩展市场映射信息
func (client *Client) ExGetMapping2562(market uint16, start uint32, count uint32) (*proto.ExMapping2562Reply, error) {
	obj := proto.NewExMapping2562()
	obj.SetParams(&proto.ExMapping2562Request{Market: market, Start: start, Count: count})
	return executeMsg(client, obj, obj.Reply)
}

// ExGetFileMeta 获取扩展市场文件元信息
func (client *Client) ExGetFileMeta(filename string) (*proto.GetFileMetaReply, error) {
	obj := proto.NewExGetFileMeta()
	obj.SetParams(&proto.GetFileMetaRequest{Filename: makeFixed40(filename)})
	return executeMsg(client, obj, obj.Reply)
}

// ExDownloadFile 下载扩展市场文件片段
func (client *Client) ExDownloadFile(filename string, start uint32, size uint32) (*proto.DownloadFileReply, error) {
	obj := proto.NewExDownloadFile()
	obj.SetParams(&proto.ExDownloadFileRequest{
		Start:    start,
		Size:     size,
		Filename: makeFixed40(filename),
	})
	return executeMsg(client, obj, obj.Reply)
}

// ExDownloadFullFile 下载完整扩展市场文件
func (client *Client) ExDownloadFullFile(filename string, size uint32) ([]byte, error) {
	if size == 0 {
		meta, err := client.ExGetFileMeta(filename)
		if err == nil {
			size = meta.Size
		}
	}

	var result []byte
	var downloaded uint32
	for {
		reply, err := client.ExDownloadFile(filename, downloaded, DefaultDownloadSize)
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

// ExGetTable 获取扩展市场表格内容
func (client *Client) ExGetTable() (string, error) {
	return client.getExTable(false)
}

// ExGetTableDetail 获取扩展市场详细表格内容
func (client *Client) ExGetTableDetail() (string, error) {
	return client.getExTable(true)
}

func (client *Client) getExTable(detail bool) (string, error) {
	start := uint32(0)
	content := ""

	for {
		var (
			reply *proto.ExGetTableChunkReply
			err   error
		)
		if detail {
			obj := proto.NewExGetTableDetail()
			obj.SetParams(start)
			reply, err = executeMsg(client, obj.ExGetTableChunk, obj.Reply)
		} else {
			obj := proto.NewExGetTable()
			obj.SetParams(start)
			reply, err = executeMsg(client, obj.ExGetTableChunk, obj.Reply)
		}
		if err != nil {
			return "", err
		}
		content += reply.Content
		if reply.Count == 0 {
			break
		}
		start += reply.Count
	}

	return content, nil
}

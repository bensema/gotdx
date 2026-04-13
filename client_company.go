package gotdx

import "github.com/bensema/gotdx/proto"

// GetCompanyCategories 获取公司信息分类
func (client *Client) GetCompanyCategories(market uint8, code string) (*proto.GetCompanyCategoryReply, error) {
	obj := proto.NewGetCompanyCategory(&proto.GetCompanyCategoryRequest{
		Market: uint16(market),
		Code:   makeCode6(code),
	})
	return executeProtocol(client, obj)
}

// GetCompanyContent 获取公司信息内容
func (client *Client) GetCompanyContent(market uint8, code string, filename string, start uint32, length uint32) (*proto.GetCompanyContentReply, error) {
	obj := proto.NewGetCompanyContent(&proto.GetCompanyContentRequest{
		Market:   uint16(market),
		Code:     makeCode6(code),
		Filename: makeFixed80(filename),
		Start:    start,
		Length:   length,
	})
	return executeProtocol(client, obj)
}

// GetFinanceInfo 获取财务信息
func (client *Client) GetFinanceInfo(market uint8, code string) (*proto.GetFinanceInfoReply, error) {
	obj := proto.NewGetFinanceInfo(&proto.GetFinanceInfoRequest{
		Market: market,
		Code:   makeCode6(code),
	})
	return executeProtocol(client, obj)
}

// GetXDXRInfo 获取除权除息信息
func (client *Client) GetXDXRInfo(market uint8, code string) (*proto.GetXDXRInfoReply, error) {
	obj := proto.NewGetXDXRInfo(&proto.GetXDXRInfoRequest{
		Market: market,
		Code:   makeCode6(code),
	})
	return executeProtocol(client, obj)
}

// GetFileMeta 获取文件元信息
func (client *Client) GetFileMeta(filename string) (*proto.GetFileMetaReply, error) {
	obj := proto.NewGetFileMeta(&proto.GetFileMetaRequest{Filename: makeFixed40(filename)})
	return executeProtocol(client, obj)
}

// DownloadFile 下载文件片段
func (client *Client) DownloadFile(filename string, start uint32, size uint32) (*proto.DownloadFileReply, error) {
	obj := proto.NewDownloadFile(&proto.DownloadFileRequest{
		Start:    start,
		Size:     size,
		Filename: makeFixed300(filename),
	})
	return executeProtocol(client, obj)
}

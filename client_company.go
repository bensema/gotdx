package gotdx

import "github.com/bensema/gotdx/proto"

// GetCompanyCategories 查询指定证券的 F10 公司信息章节目录，返回各章节的名称、对应文件名、
// 文件内起始偏移与正文长度。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节
// 定长编码、尾部补零。返回目录条目的 `Filename`、`Start`、`Length` 必须原样传给
// `GetCompanyContent` 逐章拉取正文；连接或协议执行失败时返回错误。
func (client *Client) GetCompanyCategories(market uint8, code string) (*proto.GetCompanyCategoryReply, error) {
	obj := proto.NewGetCompanyCategory(&proto.GetCompanyCategoryRequest{
		Market: uint16(market),
		Code:   makeCode6(code),
	})
	return client.execute(obj)
}

// GetCompanyContent 按目录条目拉取指定证券的 F10 章节正文，返回该区间的解码文本与实际长度。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节
// 定长编码、尾部补零；`filename`、`start`、`length` 必须取自 `GetCompanyCategories` 返回的
// 目录条目，`filename` 内部按 80 字节定长编码、尾部补零，`start` 为文件内起始偏移（字节），
// `length` 为本次请求的正文字节数。连接或协议执行失败时返回错误。
func (client *Client) GetCompanyContent(market uint8, code string, filename string, start uint32, length uint32) (*proto.GetCompanyContentReply, error) {
	obj := proto.NewGetCompanyContent(&proto.GetCompanyContentRequest{
		Market:   uint16(market),
		Code:     makeCode6(code),
		Filename: makeFixed80(filename),
		Start:    start,
		Length:   length,
	})
	return client.execute(obj)
}

// GetFinanceInfo 查询指定证券的财务概况，返回流通股本、省份、行业、每股指标、资产负债与
// 利润等字段。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节
// 定长编码、尾部补零。连接或协议执行失败时返回错误。
func (client *Client) GetFinanceInfo(market uint8, code string) (*proto.GetFinanceInfoReply, error) {
	obj := proto.NewGetFinanceInfo(&proto.GetFinanceInfoRequest{
		Market: market,
		Code:   makeCode6(code),
	})
	return client.execute(obj)
}

// GetXDXRInfo 查询指定证券的除权除息历史记录，返回按日期倒序排列的权益变动条目。
//
// `market` 为市场代码（取值见 `types.Market*` 常量）；`code` 为证券代码，内部按 6 字节
// 定长编码、尾部补零。连接或协议执行失败时返回错误。
func (client *Client) GetXDXRInfo(market uint8, code string) (*proto.GetXDXRInfoReply, error) {
	obj := proto.NewGetXDXRInfo(&proto.GetXDXRInfoRequest{
		Market: market,
		Code:   makeCode6(code),
	})
	return client.execute(obj)
}

// GetFileMeta 查询主站文件的元信息，返回文件大小、哈希校验值等字段；部分未确认字段保留
// 原始数值。
//
// `filename` 为远端文件名，内部按 40 字节定长编码、尾部补零。连接或协议执行失败时返回
// 错误。
func (client *Client) GetFileMeta(filename string) (*proto.GetFileMetaReply, error) {
	obj := proto.NewGetFileMeta(&proto.GetFileMetaRequest{Filename: makeFixed40(filename)})
	return client.execute(obj)
}

// DownloadFile 从 `start` 字节偏移下载 `size` 字节的主站文件片段，返回本次片段的原始字节与
// 实际返回长度。
//
// `filename` 为远端文件名，内部按 300 字节定长编码、尾部补零；`start` 为分块起始偏移，
// 单位字节，0 表示从文件头开始；`size` 为本次请求字节数。服务端返回片段大小为 0 表示文件
// 已取完；连接或协议执行失败时返回错误。
func (client *Client) DownloadFile(filename string, start uint32, size uint32) (*proto.DownloadFileReply, error) {
	obj := proto.NewDownloadFile(&proto.DownloadFileRequest{
		Start:    start,
		Size:     size,
		Filename: makeFixed300(filename),
	})
	return client.execute(obj)
}

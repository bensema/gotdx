package gotdx

import (
	"encoding/csv"
	"strings"

	"github.com/bensema/gotdx/proto"
	"github.com/bensema/gotdx/types"
)

// CompanyInfoSection 表示 F10 公司信息中的一个章节。
type CompanyInfoSection struct {
	// Name 为章节显示名称，取自公司信息目录中的章节标题。
	Name string
	// Content 为章节正文，GBK 解码后的纯文本内容。
	Content string
}

// CompanyInfoBundle 聚合返回的公司信息，包含 F10 章节、除权除息记录与财务信息。
type CompanyInfoBundle struct {
	// Sections 为按目录顺序拉取到的 F10 章节列表；目录为空时该切片为空。
	Sections []CompanyInfoSection
	// XDXR 为除权除息记录列表；拉取失败时保持 `nil`，不影响章节结果。
	XDXR []proto.XDXRItem
	// Finance 为财务信息应答；拉取失败时保持 `nil`，不影响章节结果。
	Finance *proto.GetFinanceInfoReply
}

// GetCompanyInfo 聚合获取指定证券的公司信息，包含 F10 章节、除权除息记录与财务信息。
//
// 参数 `market` 为市场代码，取值见 `types.Market*` 常量；参数 `code` 为六位证券代码。先按目录逐项
// 拉取章节正文，任一章节拉取失败即作为硬性错误直接返回；除权除息与财务信息为尽力而为，拉取失败时
// 保留已取到的章节，对应字段保持 `nil` 或空切片。
func (client *Client) GetCompanyInfo(market uint8, code string) (*CompanyInfoBundle, error) {
	categories, err := client.GetCompanyCategories(market, code)
	if err != nil {
		return nil, err
	}

	result := &CompanyInfoBundle{}
	// 按目录逐项拉取章节正文，章节内容失败属于硬性错误，直接返回。
	for _, category := range categories.Categories {
		content, err := client.GetCompanyContent(market, code, category.Filename, category.Start, category.Length)
		if err != nil {
			return nil, err
		}
		result.Sections = append(result.Sections, CompanyInfoSection{
			Name:    category.Name,
			Content: content.Content,
		})
	}

	// 除权除息、财务信息允许缺失：仅在成功时填入结果。
	xdxr, err := client.GetXDXRInfo(market, code)
	if err == nil {
		result.XDXR = xdxr.List
	}

	finance, err := client.GetFinanceInfo(market, code)
	if err == nil {
		result.Finance = finance
	}

	return result, nil
}

// DownloadFullFile 按固定块大小循环下载完整文件。
//
// 参数 `filename` 为服务端协议文件名；参数 `size` 为目标总大小（字节），传 0 时先拉取文件元信息
// 中的大小作为下载上限，元信息获取失败则仅依靠空块结束循环。服务器返回空块，或已下载字节达到
// `size` 时结束；返回拼接后的完整文件字节。任一协议请求失败即返回 `nil` 与该错误。
func (client *Client) DownloadFullFile(filename string, size uint32) ([]byte, error) {
	// 未指定大小时从元信息获取文件大小作为下载上限。
	if size == 0 {
		meta, err := client.GetFileMeta(filename)
		if err == nil {
			size = meta.Size
		}
	}

	var result []byte
	var downloaded uint32
	for {
		reply, err := client.DownloadFile(filename, downloaded, types.DefaultDownloadSize)
		if err != nil {
			return nil, err
		}
		// 空块表示文件已下载完毕。
		if reply.Size == 0 {
			break
		}
		result = append(result, reply.Data...)
		downloaded += reply.Size
		// 达到目标大小停止；`size` 为 0 时仅靠空块结束循环。
		if size != 0 && downloaded >= size {
			break
		}
	}
	return result, nil
}

// GetBlockFile 下载并返回指定板块文件的完整字节内容。
//
// 参数 `filename` 为服务端板块文件名；先取文件元信息获得大小，再按块下载。元信息获取失败或分块
// 下载失败时返回 `nil` 与对应错误。
func (client *Client) GetBlockFile(filename string) ([]byte, error) {
	meta, err := client.GetFileMeta(filename)
	if err != nil {
		return nil, err
	}
	return client.DownloadFullFile(filename, meta.Size)
}

// GetTableFile 下载并解析以竖线 `|` 分隔的表格文件，返回二维字符串表格；每个内层切片对应一行，
// 元素为该行各列文本，空行被跳过。
//
// 参数 `filename` 为服务端表格文件名；下载失败时返回 `nil` 与该错误。
func (client *Client) GetTableFile(filename string) ([][]string, error) {
	content, err := client.DownloadFullFile(filename, 0)
	if err != nil {
		return nil, err
	}
	return parsePipeTableContent(content), nil
}

// GetCSVFile 下载并解析指定 CSV 文件，返回二维字符串表格；每行允许字段数不一致。
//
// 参数 `filename` 为服务端 CSV 文件名；下载或解析失败时返回 `nil` 与对应错误。
func (client *Client) GetCSVFile(filename string) ([][]string, error) {
	content, err := client.DownloadFullFile(filename, 0)
	if err != nil {
		return nil, err
	}
	return parseCSVContent(content)
}

// parsePipeTableContent 把 GBK 编码的竖线分隔表格转成二维字符串。
//
// 参数 `content` 为下载得到的原始文件字节；解码后按换行切行，跳过空行与首尾空白，再按竖线 `|`
// 分列。
func parsePipeTableContent(content []byte) [][]string {
	lines := strings.Split(proto.Utf8ToGbk(content), "\n")
	result := make([][]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, strings.Split(line, "|"))
	}
	return result
}

// parseCSVContent 解析 GBK 编码的 CSV 文本。
//
// 参数 `content` 为下载得到的原始文件字节；解析器把 `FieldsPerRecord` 置为 -1，允许每行字段数不
// 一致；返回解析出的二维表格与 CSV 读取错误。
func parseCSVContent(content []byte) ([][]string, error) {
	reader := csv.NewReader(strings.NewReader(proto.Utf8ToGbk(content)))
	reader.FieldsPerRecord = -1
	return reader.ReadAll()
}

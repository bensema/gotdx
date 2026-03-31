package gotdx

import (
	"encoding/csv"
	"strings"

	"github.com/bensema/gotdx/proto"
)

type CompanyInfoSection struct {
	Name    string
	Content string
}

type CompanyInfoBundle struct {
	Sections []CompanyInfoSection
	XDXR     []proto.XDXRItem
	Finance  *proto.GetFinanceInfoReply
}

// GetCompanyInfo 获取公司信息聚合结果
func (client *Client) GetCompanyInfo(market uint8, code string) (*CompanyInfoBundle, error) {
	categories, err := client.GetCompanyCategories(market, code)
	if err != nil {
		return nil, err
	}

	result := &CompanyInfoBundle{}
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

// DownloadFullFile 下载完整文件
func (client *Client) DownloadFullFile(filename string, size uint32) ([]byte, error) {
	if size == 0 {
		meta, err := client.GetFileMeta(filename)
		if err == nil {
			size = meta.Size
		}
	}

	var result []byte
	var downloaded uint32
	for {
		reply, err := client.DownloadFile(filename, downloaded, DefaultDownloadSize)
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

// GetBlockFile 获取完整板块文件
func (client *Client) GetBlockFile(filename string) ([]byte, error) {
	meta, err := client.GetFileMeta(filename)
	if err != nil {
		return nil, err
	}
	return client.DownloadFullFile(filename, meta.Size)
}

// GetTableFile 获取以竖线分隔的表格文件
func (client *Client) GetTableFile(filename string) ([][]string, error) {
	content, err := client.DownloadFullFile(filename, 0)
	if err != nil {
		return nil, err
	}
	return parsePipeTableContent(content), nil
}

// GetCSVFile 获取 CSV 文件
func (client *Client) GetCSVFile(filename string) ([][]string, error) {
	content, err := client.DownloadFullFile(filename, 0)
	if err != nil {
		return nil, err
	}
	return parseCSVContent(content)
}

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

func parseCSVContent(content []byte) ([][]string, error) {
	reader := csv.NewReader(strings.NewReader(proto.Utf8ToGbk(content)))
	reader.FieldsPerRecord = -1
	return reader.ReadAll()
}

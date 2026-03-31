package gotdx

import (
	"fmt"

	"github.com/bensema/gotdx/proto"
)

type BlockFlatItem struct {
	BlockName string
	BlockType uint16
	CodeIndex int
	Code      string
}

type BlockGroup struct {
	BlockName  string
	BlockType  uint16
	StockCount int
	Codes      []string
}

func ParseBlockFlat(data []byte) ([]BlockFlatItem, error) {
	if len(data) < 386 {
		return nil, fmt.Errorf("invalid block data length: %d", len(data))
	}

	pos := 384
	total := int(uint16(data[pos]) | uint16(data[pos+1])<<8)
	pos += 2

	result := make([]BlockFlatItem, 0, total)
	for i := 0; i < total; i++ {
		if pos+13 > len(data) {
			return nil, fmt.Errorf("invalid block header %d", i)
		}
		blockName := proto.Utf8ToGbk(data[pos : pos+9])
		pos += 9
		stockCount := int(uint16(data[pos]) | uint16(data[pos+1])<<8)
		blockType := uint16(data[pos+2]) | uint16(data[pos+3])<<8
		pos += 4

		blockStockBegin := pos
		for codeIndex := 0; codeIndex < stockCount; codeIndex++ {
			if pos+7 > len(data) {
				return nil, fmt.Errorf("invalid block code %d:%d", i, codeIndex)
			}
			result = append(result, BlockFlatItem{
				BlockName: blockName,
				BlockType: blockType,
				CodeIndex: codeIndex,
				Code:      proto.Utf8ToGbk(data[pos : pos+7]),
			})
			pos += 7
		}
		pos = blockStockBegin + 2800
	}

	return result, nil
}

func ParseBlockGroups(data []byte) ([]BlockGroup, error) {
	if len(data) < 386 {
		return nil, fmt.Errorf("invalid block data length: %d", len(data))
	}

	pos := 384
	total := int(uint16(data[pos]) | uint16(data[pos+1])<<8)
	pos += 2

	result := make([]BlockGroup, 0, total)
	for i := 0; i < total; i++ {
		if pos+13 > len(data) {
			return nil, fmt.Errorf("invalid block header %d", i)
		}
		blockName := proto.Utf8ToGbk(data[pos : pos+9])
		pos += 9
		stockCount := int(uint16(data[pos]) | uint16(data[pos+1])<<8)
		blockType := uint16(data[pos+2]) | uint16(data[pos+3])<<8
		pos += 4

		blockStockBegin := pos
		group := BlockGroup{
			BlockName:  blockName,
			BlockType:  blockType,
			StockCount: stockCount,
			Codes:      make([]string, 0, stockCount),
		}
		for codeIndex := 0; codeIndex < stockCount; codeIndex++ {
			if pos+7 > len(data) {
				return nil, fmt.Errorf("invalid block code %d:%d", i, codeIndex)
			}
			group.Codes = append(group.Codes, proto.Utf8ToGbk(data[pos:pos+7]))
			pos += 7
		}
		result = append(result, group)
		pos = blockStockBegin + 2800
	}

	return result, nil
}

// GetParsedBlockFile 获取并解析板块文件
func (client *Client) GetParsedBlockFile(filename string) ([]BlockFlatItem, error) {
	content, err := client.GetBlockFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseBlockFlat(content)
}

// GetGroupedBlockFile 获取并按板块分组解析板块文件
func (client *Client) GetGroupedBlockFile(filename string) ([]BlockGroup, error) {
	content, err := client.GetBlockFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseBlockGroups(content)
}

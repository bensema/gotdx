package gotdx

import (
	"fmt"

	"github.com/bensema/gotdx/proto"
)

// BlockFlatItem 表示板块文件中一只成分股的扁平记录，将所属板块名称、板块类型、板块内序号与
// 股票代码平铺为一条，便于按股票维度遍历成分股列表。
//
// 该结构体仅承载解析结果，由解析函数构造，调用方一般不需要直接设置字段零值。
type BlockFlatItem struct {
	BlockName string // 所属板块名称，9 字节槽位 GBK 解码后的 UTF-8 字符串，尾部空字节已剔除
	BlockType uint16 // 板块类型代码，2 字节原始值，取值对照 `types.BoardType*` 常量
	CodeIndex int    // 成分股在所属板块内的序号，从 0 开始递增
	Code      string // 成分股证券代码，7 字节槽位 GBK 解码后的字符串，尾部空字节已剔除
}

// BlockGroup 表示按板块聚合后的一组成分股，记录板块名称、板块类型、成员数量与成员代码列表，
// 便于按板块维度查看该板块包含哪些股票。
//
// 该结构体仅承载解析结果，由解析函数构造，调用方一般不需要直接设置字段零值。
type BlockGroup struct {
	BlockName  string   // 板块名称，9 字节槽位 GBK 解码后的 UTF-8 字符串，尾部空字节已剔除
	BlockType  uint16   // 板块类型代码，2 字节原始值，取值对照 `types.BoardType*` 常量
	StockCount int      // 板块成员数量，来自槽头 2 字节成员数字段
	Codes      []string // 成员证券代码列表，每个为 7 字节 GBK 解码字符串；切片由解析函数分配并持有所有权
}

// ParseBlockFlat 解析板块文件二进制数据，将每个板块下的成分股逐条展开为扁平条目列表。
//
// 参数 `data` 为下载完成的板块文件完整字节内容，长度至少为 386 字节（384 字节固定文件头加
// 2 字节板块总数）；函数不会保留对该切片的引用。文件采用 GBK 编码与定长成员区布局。
//
// 解析成功时返回扁平条目切片，错误返回 `nil`；输入长度不足、槽头或成员代码剩余字节不足时
// 返回错误，避免越界读取。
func ParseBlockFlat(data []byte) ([]BlockFlatItem, error) {
	// 最短合法内容为 384 字节文件头加 2 字节板块总数，长度不足说明文件不完整。
	if len(data) < 386 {
		return nil, fmt.Errorf("invalid block data length: %d", len(data))
	}

	// 跳过 384 字节固定文件头，定位到板块总数所在位置。
	pos := 384
	// 手工拼出 `uint16` 板块总数。
	total := int(uint16(data[pos]) | uint16(data[pos+1])<<8)
	pos += 2

	result := make([]BlockFlatItem, 0, total)
	for i := 0; i < total; i++ {
		// 槽头定长 13 字节（9 字节名称、2 字节成员数、2 字节类型），剩余长度不足则报错。
		if pos+13 > len(data) {
			return nil, fmt.Errorf("invalid block header %d", i)
		}
		// 工具函数历史命名保留旧名 `Utf8ToGbk`，实际把 GBK 字节解码为内部 UTF-8 字符串并剔除尾部空字节填充。
		blockName := proto.Utf8ToGbk(data[pos : pos+9])
		pos += 9
		// 成员数与板块类型紧接名称之后，均为 2 字节。
		stockCount := int(uint16(data[pos]) | uint16(data[pos+1])<<8)
		blockType := uint16(data[pos+2]) | uint16(data[pos+3])<<8
		pos += 4

		// 记录成员代码区起点：成员区按 2800 字节定长对齐，读完成员后按槽长回跳。
		blockStockBegin := pos
		for codeIndex := 0; codeIndex < stockCount; codeIndex++ {
			// 成员代码固定 7 字节，剩余长度不足则报错。
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
		// 成员代码区按固定 2800 字节对齐，跳过剩余填充后定位到下一个板块槽头。
		pos = blockStockBegin + 2800
	}

	return result, nil
}

// ParseBlockGroups 解析板块文件二进制数据，按板块分组返回成分股代码列表。
//
// 参数 `data` 为下载完成的板块文件完整字节内容，长度至少为 386 字节；文件布局与解析步骤同
// `ParseBlockFlat`，区别在于成员代码聚合进对应 `BlockGroup`，而不是展开成扁平条目。
//
// 解析成功时返回板块分组切片，错误返回 `nil`；输入长度不足或槽头、成员代码剩余字节不足时
// 返回错误。
func ParseBlockGroups(data []byte) ([]BlockGroup, error) {
	// 最短合法内容为 384 字节文件头加 2 字节板块总数，长度不足说明文件不完整。
	if len(data) < 386 {
		return nil, fmt.Errorf("invalid block data length: %d", len(data))
	}

	// 跳过 384 字节固定文件头，定位到板块总数所在位置。
	pos := 384
	// 手工拼出 `uint16` 板块总数。
	total := int(uint16(data[pos]) | uint16(data[pos+1])<<8)
	pos += 2

	result := make([]BlockGroup, 0, total)
	for i := 0; i < total; i++ {
		// 槽头定长 13 字节（9 字节名称、2 字节成员数、2 字节类型），剩余长度不足则报错。
		if pos+13 > len(data) {
			return nil, fmt.Errorf("invalid block header %d", i)
		}
		// 工具函数历史命名保留旧名 `Utf8ToGbk`，实际把 GBK 字节解码为内部 UTF-8 字符串并剔除尾部空字节填充。
		blockName := proto.Utf8ToGbk(data[pos : pos+9])
		pos += 9
		// 成员数与板块类型紧接名称之后，均为 2 字节。
		stockCount := int(uint16(data[pos]) | uint16(data[pos+1])<<8)
		blockType := uint16(data[pos+2]) | uint16(data[pos+3])<<8
		pos += 4

		// 记录成员代码区起点：成员区按 2800 字节定长对齐，读完成员后按槽长回跳。
		blockStockBegin := pos
		group := BlockGroup{
			BlockName:  blockName,
			BlockType:  blockType,
			StockCount: stockCount,
			Codes:      make([]string, 0, stockCount),
		}
		for codeIndex := 0; codeIndex < stockCount; codeIndex++ {
			// 成员代码固定 7 字节，剩余长度不足则报错。
			if pos+7 > len(data) {
				return nil, fmt.Errorf("invalid block code %d:%d", i, codeIndex)
			}
			group.Codes = append(group.Codes, proto.Utf8ToGbk(data[pos:pos+7]))
			pos += 7
		}
		result = append(result, group)
		// 成员代码区按固定 2800 字节对齐，跳过剩余填充后定位到下一个板块槽头。
		pos = blockStockBegin + 2800
	}

	return result, nil
}

// GetParsedBlockFile 下载指定板块文件并解析为扁平成分股条目列表。
//
// 参数 `filename` 为服务端板块文件名；先通过 `GetBlockFile` 下载完整文件内容，再交给
// `ParseBlockFlat` 解析。下载失败时直接返回底层错误；文件内容不合法时返回解析错误；解析成功
// 返回扁平条目切片，错误为 `nil`。
func (client *Client) GetParsedBlockFile(filename string) ([]BlockFlatItem, error) {
	content, err := client.GetBlockFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseBlockFlat(content)
}

// GetGroupedBlockFile 下载指定板块文件并按板块分组解析为成分股代码列表。
//
// 参数 `filename` 为服务端板块文件名；先通过 `GetBlockFile` 下载完整文件内容，再交给
// `ParseBlockGroups` 解析。下载失败时直接返回底层错误；文件内容不合法时返回解析错误；解析成功
// 返回板块分组切片，错误为 `nil`。
func (client *Client) GetGroupedBlockFile(filename string) ([]BlockGroup, error) {
	content, err := client.GetBlockFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseBlockGroups(content)
}

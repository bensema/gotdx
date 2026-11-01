// Command stock_f10_block 演示查询证券 F10 资料打包结果，并加载概念板块清单打印前若干条板块成分。
//
// 运行示例：`go run ./examples/stock_f10_block`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// `market` 取 `types.MarketSZ`（深市），代码 `000001`；返回聚合信息包：章节拉取失败即报错，
	// 除权除息与财务信息为尽力而为，缺失时对应字段保持 `nil` 或空切片。
	f10, err := client.StockF10(types.MarketSZ.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("sections=%d xdxr=%d finance_nil=%v", len(f10.Sections), len(f10.XDXR), f10.Finance == nil)

	// `filename` 取 `types.BlockFileGN`（`block_gn.dat`，概念板块文件）；下载并解析为扁平化板块条目列表。
	blocks, err := client.StockBlock(types.BlockFileGN)
	if err != nil {
		log.Fatalln(err)
	}

	limit := 5
	if len(blocks) < limit {
		limit = len(blocks)
	}
	// 板块数量可能不足 5 条，先截断再遍历，仅打印前 5 条。
	for _, item := range blocks[:limit] {
		log.Printf("block=%s type=%d code=%s", item.BlockName, item.BlockType, item.Code)
	}
}

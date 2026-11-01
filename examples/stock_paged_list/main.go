// Command stock_paged_list 演示先查询市场证券总数，再按固定页大小逐页翻页遍历深市证券列表，并打印每页首尾证券。
//
// 运行示例：`go run ./examples/stock_paged_list`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

// 分页遍历示例使用的本地翻页参数。
const (
	// pageSize 每页请求的证券条数，作为 `StockList` 的 `count` 参数；单次请求条数受协议单页上限约束。
	pageSize = 800
	// maxPages 最多翻页次数（含起始页）；达到该页数后即使列表尚未取完也停止遍历，避免一次性拉取全量。
	maxPages = 3
	// startPage 起始页序号，0 表示从证券列表最前面（偏移 0）开始翻页。
	startPage = 0
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// 先查询深市证券总数，作为分页遍历的上界参考。
	total, err := client.StockCount(types.MarketSZ.Uint8())
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("market=SZ total=%d page_size=%d max_pages=%d", total, pageSize, maxPages)

	// 逐页翻取深市证券列表；`start` 为起始偏移序号，等于页码乘以每页条数；某页返回为空时说明已到列表末尾，提前结束。
	var fetched int
	for page := startPage; page < maxPages; page++ {
		start := uint32(page * pageSize)
		items, err := client.StockList(types.MarketSZ.Uint8(), start, pageSize)
		if err != nil {
			log.Fatalf("page=%d start=%d failed: %v", page, start, err)
		}
		if len(items) == 0 {
			break
		}

		// 记录本页首尾证券，便于观察分页衔接是否连续。
		first := items[0]
		last := items[len(items)-1]
		fetched += len(items)
		log.Printf("page=%d fetched=%d first=%s/%s last=%s/%s",
			page, len(items), first.Code, first.Name, last.Code, last.Name)
	}

	log.Printf("paged_fetch_done fetched=%d of total=%d", fetched, total)
}

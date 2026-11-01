// Command ex_paged_list 演示先查询扩展市场标的总数，再按固定页大小循环分页拉取 ExList 列表，
// 直到取空或达到最大页数限制，并汇总实际拉取条数。
//
// 运行示例：`go run ./examples/ex_paged_list`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

const (
	// `pageSize` 为单次分页请求的标的条数，同时用于按页码推算下一页起始偏移。
	pageSize = 1000
	// `maxPages` 为本次演示允许翻页的最大页数；达到该页数后即使列表未取完也停止循环。
	maxPages = 3
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	// 先取标的总数，用于校验分页拉取的覆盖范围。
	total, err := client.ExCount()
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("ex_total=%d page_size=%d max_pages=%d", total, pageSize, maxPages)

	var fetched int
	for page := 0; page < maxPages; page++ {
		// 起始偏移 = 当前页码 × 每页条数，即跳过前面已拉取的记录。
		start := uint32(page * pageSize)
		// 从 `start` 偏移开始请求 `pageSize` 条证券列表记录。
		items, err := client.ExList(start, pageSize)
		if err != nil {
			log.Fatalf("page=%d start=%d failed: %v", page, start, err)
		}
		// 服务端返回空页即表示已到列表末尾，提前结束翻页。
		if len(items) == 0 {
			break
		}

		// 记录每页首条与末条，便于核对翻页边界是否连续。
		first := items[0]
		last := items[len(items)-1]
		fetched += len(items)
		log.Printf("page=%d fetched=%d first=%s/%s last=%s/%s",
			page, len(items), first.Code, first.Name, last.Code, last.Name)
	}

	log.Printf("paged_fetch_done fetched=%d of total=%d", fetched, total)
}

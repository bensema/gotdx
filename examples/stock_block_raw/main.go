// Command stock_block_raw 演示板块文件的元信息查询、分块下载与整包下载，并分别按扁平列表、按板块分组两种方式解析板块文件。
//
// 运行示例：`go run ./examples/stock_block_raw`
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bensema/gotdx/cache"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
	"github.com/redis/go-redis/v9"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	if _, err := client.Connect(); err != nil {
		log.Fatalln(err)
	}

	// 查询默认板块文件 `BlockFileDefault`（`block.dat`）的元信息，返回文件总大小 `Size`（字节）。
	meta, err := client.GetFileMeta(types.BlockFileDefault)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("file_meta name=%s size=%d", types.BlockFileDefault, meta.Size)

	// 演示分块下载：第二参 `offset=0` 为文件内起始字节偏移，第三参 `1024` 为本次请求的字节数。
	chunk, err := client.DownloadFile(types.BlockFileDefault, 0, 1024)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("download_chunk size=%d", chunk.Size)

	// 演示整包下载：第二参 `size=meta.Size` 为目标总字节数；传 0 时会先拉取元信息中的大小作为下载上限。
	full, err := client.DownloadFullFile(types.BlockFileDefault, meta.Size)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("download_full size=%d", len(full))

	// 平铺解析概念板块文件 `BlockFileGN`（`block_gn.dat`），返回成分股条目扁平列表；
	// `min(10, len(...))` 防止切片越界，仅打印前 10 条。
	flat, err := client.GetParsedBlockFile(types.BlockFileGN)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range flat[:min(10, len(flat))] {
		log.Printf("flat block=%s type=%d code=%s", item.BlockName, item.BlockType, item.Code)
	}

	// 分组解析风格板块文件 `BlockFileFG`（`block_fg.dat`），每条为一个板块及其成分股计数；
	// `min(5, len(...))` 防止切片越界，仅打印前 5 条。
	grouped, err := client.GetGroupedBlockFile(types.BlockFileFG)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range grouped[:min(5, len(grouped))] {
		log.Printf("grouped block=%s type=%d stocks=%d", item.BlockName, item.BlockType, item.StockCount)
	}

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379", DB: 2})
	cached := cache.NewCacheClient(rdb, client)
	ctx := context.Background()
	flatCache, err := cached.GetParsedBlockFile(ctx, types.BlockFileGN)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range flatCache[:min(10, len(flatCache))] {
		log.Printf("flat block=%s type=%d code=%s", item.BlockName, item.BlockType, item.Code)
	}
	stats := cached.Stats() // 命中/回源/熔断拒绝/空标记命中 各计数
	fmt.Printf("命中=%d 空标记命中=%d 回源=%d 回源失败=%d 熔断拒绝=%d\n",
		stats.Hits, stats.EmptyHits, stats.Fetches, stats.SourceErrors, stats.CircuitRejects)
}

// min 返回整数 a 与 b 中较小的一个。
//
// 参数 a、b 为待比较的两个整数；返回值为二者较小值，用于在切片截断时把长度限制在有效范围内，避免越界。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
